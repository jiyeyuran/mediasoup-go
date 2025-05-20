package mediasoup

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	FbsNotification "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Notification"
	FbsRequest "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Request"
	FbsTransport "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Transport"
	FbsWorker "github.com/jiyeyuran/mediasoup-go/v2/internal/FBS/Worker"
	"github.com/jiyeyuran/mediasoup-go/v2/internal/channel"
)

// Worker represents a mediasoup C++ subprocess that runs in a single CPU core and handles Router
// instances.
type Worker struct {
	baseListener
	cmd                      *exec.Cmd
	channel                  *channel.Channel
	logger                   *slog.Logger
	routers                  sync.Map
	webRtcServers            sync.Map
	appData                  H
	newWebRtcServerListeners []func(*WebRtcServer)
	newRouterListeners       []func(*Router)
	closed                   bool
	err                      error
}

// NewWorker create a Worker.
func NewWorker(workerBinaryPath string, options ...Option) (*Worker, error) {
	opts := &WorkerSettings{
		LogLevel: WorkerLogLevelWarn,
	}
	for _, opt := range options {
		opt(opts)
	}

	logger := opts.Logger

	if logger == nil {
		output := io.Writer(os.Stdout)
		level := slog.LevelInfo

		switch opts.LogLevel {
		case WorkerLogLevelDebug:
			level = slog.LevelDebug

		case WorkerLogLevelWarn:
			level = slog.LevelWarn

		case WorkerLogLevelError:
			level = slog.LevelError

		case WorkerLogLevelNone:
			output = io.Discard
		}

		logger = slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{
			AddSource: true,
			Level:     level,
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					source := a.Value.Any().(*slog.Source)
					source.File = filepath.Base(source.File)
				}
				return a
			},
		}))
	}

	producerReader, producerWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			producerWriter.Close()
			producerReader.Close()
		}
	}()

	consumerReader, consumerWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			consumerWriter.Close()
			consumerReader.Close()
		}
	}()

	args := opts.CustomArgs

	if len(opts.LogLevel) > 0 {
		args = append(args, "--logLevel="+string(opts.LogLevel))
	}

	for _, logTag := range opts.LogTags {
		args = append(args, fmt.Sprintf("--logTags=%s", logTag))
	}

	if len(opts.DtlsCertificateFile) > 0 && len(opts.DtlsPrivateKeyFile) > 0 {
		args = append(args,
			"--dtlsCertificateFile="+opts.DtlsCertificateFile,
			"--dtlsPrivateKeyFile="+opts.DtlsPrivateKeyFile,
		)
	}

	if len(opts.LibwebrtcFieldTrials) > 0 {
		args = append(args, "--libwebrtcFieldTrials="+opts.LibwebrtcFieldTrials)
	}

	if opts.DisableLiburing {
		args = append(args, "--disableLiburing=true")
	}

	logger.Debug(fmt.Sprintf("starting worker process: %s %s", workerBinaryPath, strings.Join(args, " ")))

	cmd := exec.CommandContext(context.Background(), workerBinaryPath, args...)
	cmd.ExtraFiles = []*os.File{producerReader, consumerWriter}
	cmd.Env = append(opts.Env, "MEDIASOUP_VERSION=default", "ULIMIT_CORE=unlimited")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	// stderr is closed by command
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	// stdout is closed by command
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// start worker process
	if err = cmd.Start(); err != nil {
		return nil, err
	}

	pid := cmd.Process.Pid
	logger = logger.With("pid", pid)
	channel := channel.NewChannel(producerWriter, consumerReader, logger)

	// spawnDone indices the worker process is started
	spawnDone := uint32(0)
	// notify the worker process is running or stopped with error
	doneCh := make(chan error)

	sub := channel.Subscribe(strconv.Itoa(pid), func(event FbsNotification.Event, body *FbsNotification.BodyT) {
		if event == FbsNotification.EventWORKER_RUNNING && atomic.CompareAndSwapUint32(&spawnDone, 0, 1) {
			logger.Debug("worker process is running")
			close(doneCh)
		}
	})
	defer sub.Unsubscribe()

	go func() {
		stderrLogger := logger.With("stderr", true)
		r := bufio.NewReader(stderr)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}
			stderrLogger.Info(string(line))
		}
	}()
	go func() {
		stdoutLogger := logger.With("stdout", true)
		r := bufio.NewReader(stdout)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}
			stdoutLogger.Debug(string(line))
		}
	}()

	// start channel after setting up a listener on pid
	channel.Start()

	w := &Worker{
		cmd:     cmd,
		channel: channel,
		logger:  logger,
		appData: opts.AppData,
	}

	go w.wait(cmd, &spawnDone, doneCh)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	select {
	case err = <-doneCh:
		if err != nil {
			return nil, err
		}
		return w, nil

	case <-ctx.Done():
		return nil, ErrWorkerStartTimeout
	}
}

func (w *Worker) wait(cmd *exec.Cmd, spawnDone *uint32, doneCh chan error) {
	err := cmd.Wait()
	if err != nil {
		code := cmd.ProcessState.ExitCode()
		err = fmt.Errorf("worker process failed unexpectedly, code: %d, %w", code, err)
	}

	if atomic.CompareAndSwapUint32(spawnDone, 0, 1) {
		if err == nil {
			err = errors.New("worker process failed unexpectedly")
		}
		doneCh <- err
		return
	}

	if err != nil {
		w.logger.Error(err.Error())
		w.err = err
	} else {
		w.logger.Info("worker process quited")
	}

	w.processQuited()
}

func (w *Worker) Pid() int {
	if w.cmd == nil || w.cmd.Process == nil {
		return 0
	}
	return w.cmd.Process.Pid
}

func (w *Worker) AppData() H {
	return w.appData
}

func (w *Worker) Err() error {
	return w.err
}

func (w *Worker) Close() {
	w.CloseContext(context.Background())
}

func (w *Worker) CloseContext(ctx context.Context) {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.logger.DebugContext(ctx, "Close()")

	if w.cmd.ProcessState == nil {
		go func() {
			now := time.Now()
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if w.cmd.ProcessState != nil {
						return
					}
					if time.Since(now) > time.Second {
						w.logger.WarnContext(ctx, "force kill worker process")
						w.cmd.Process.Kill()
						return
					}
				}
			}
		}()

		// w.channel.Request(ctx, &FbsRequest.RequestT{
		// 	Method: FbsRequest.MethodWORKER_CLOSE,
		// })
		w.cmd.Process.Signal(os.Interrupt)
	}

	w.closed = true
	w.mu.Unlock()

	w.cleanupAfterClosed(ctx)
}

func (w *Worker) Closed() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()

	return w.closed
}

// Dump returns the resources allocated by the worker.
func (w *Worker) Dump() (dump *WorkerDump, err error) {
	return w.DumpContext(context.Background())
}

func (w *Worker) DumpContext(ctx context.Context) (dump *WorkerDump, err error) {
	w.logger.DebugContext(ctx, "Dump()")

	respAny, err := w.channel.Request(ctx, &FbsRequest.RequestT{
		Method: FbsRequest.MethodWORKER_DUMP,
	})
	if err != nil {
		return
	}
	resp := respAny.(*FbsWorker.DumpResponseT)
	dump = &WorkerDump{
		Pid:             resp.Pid,
		WebRtcServerIds: resp.WebRtcServerIds,
		RouterIds:       resp.RouterIds,
	}
	if resp.ChannelMessageHandlers != nil {
		dump.ChannelMessageHandlers = &WorkerDumpChannelMessageHandlers{
			ChannelRequestHandlers:      resp.ChannelMessageHandlers.ChannelRequestHandlers,
			ChannelNotificationHandlers: resp.ChannelMessageHandlers.ChannelNotificationHandlers,
		}
	}
	if resp.Liburing != nil {
		dump.Liburing = &WorkerDumpLiburing{
			SqeProcessCount:   resp.Liburing.SqeProcessCount,
			SqeMissCount:      resp.Liburing.SqeMissCount,
			UserDataMissCount: resp.Liburing.UserDataMissCount,
		}
	}
	return dump, nil
}

// GetResourceUsage returns the worker process resource usage.
func (w *Worker) GetResourceUsage() (usage *WorkerResourceUsage, err error) {
	return w.GetResourceUsageContext(context.Background())
}

func (w *Worker) GetResourceUsageContext(ctx context.Context) (usage *WorkerResourceUsage, err error) {
	w.logger.DebugContext(ctx, "GetResourceUsage()")

	respAny, err := w.channel.Request(ctx, &FbsRequest.RequestT{
		Method: FbsRequest.MethodWORKER_GET_RESOURCE_USAGE,
	})
	if err != nil {
		return
	}
	resp := respAny.(*FbsWorker.ResourceUsageResponseT)

	return (*WorkerResourceUsage)(resp), nil
}

// UpdateSettings updates worker settings.
func (w *Worker) UpdateSettings(settings *WorkerUpdatableSettings) (err error) {
	return w.UpdateSettingsContext(context.Background(), settings)
}

func (w *Worker) UpdateSettingsContext(ctx context.Context, settings *WorkerUpdatableSettings) (err error) {
	w.logger.DebugContext(ctx, "UpdateSettings()")

	_, err = w.channel.Request(ctx, &FbsRequest.RequestT{
		Method: FbsRequest.MethodWORKER_UPDATE_SETTINGS,
		Body: &FbsRequest.BodyT{
			Type: FbsRequest.BodyWorker_UpdateSettingsRequest,
			Value: &FbsWorker.UpdateSettingsRequestT{
				LogLevel: string(settings.LogLevel),
				LogTags: collect(settings.LogTags, func(tag WorkerLogTag) string {
					return string(tag)
				}),
			},
		},
	})

	return
}

// CreateWebRtcServer creates a WebRtcServer.
func (w *Worker) CreateWebRtcServer(options *WebRtcServerOptions) (*WebRtcServer, error) {
	return w.CreateWebRtcServerContext(context.Background(), options)
}

func (w *Worker) CreateWebRtcServerContext(ctx context.Context, options *WebRtcServerOptions) (*WebRtcServer, error) {
	w.logger.DebugContext(ctx, "CreateWebRtcServer()")

	id := uuid(webRtcServerPrefix)

	req := &FbsWorker.CreateWebRtcServerRequestT{
		WebRtcServerId: id,
		ListenInfos:    make([]*FbsTransport.ListenInfoT, len(options.ListenInfos)),
	}

	for i, info := range options.ListenInfos {
		req.ListenInfos[i] = &FbsTransport.ListenInfoT{
			Protocol:         orElse(info.Protocol == TransportProtocolTCP, FbsTransport.ProtocolTCP, FbsTransport.ProtocolUDP),
			Ip:               info.Ip,
			Port:             info.Port,
			AnnouncedAddress: info.AnnouncedAddress,
			SendBufferSize:   info.SendBufferSize,
			RecvBufferSize:   info.RecvBufferSize,
			PortRange: &FbsTransport.PortRangeT{
				Min: info.PortRange.Min,
				Max: info.PortRange.Max,
			},
			Flags: &FbsTransport.SocketFlagsT{
				Ipv6Only:     info.Flags.IPv6Only,
				UdpReusePort: info.Flags.UDPReusePort,
			},
		}
	}

	_, err := w.channel.Request(ctx, &FbsRequest.RequestT{
		Method: FbsRequest.MethodWORKER_CREATE_WEBRTCSERVER,
		Body: &FbsRequest.BodyT{
			Type:  FbsRequest.BodyWorker_CreateWebRtcServerRequest,
			Value: req,
		},
	})
	if err != nil {
		return nil, err
	}
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil, ErrWorkerClosed
	}
	server := NewWebRtcServer(w, id, orElse(options.AppData == nil, H{}, options.AppData))
	w.webRtcServers.Store(id, server)
	server.OnClose(func() {
		w.webRtcServers.Delete(id)
	})
	listeners := w.newWebRtcServerListeners
	w.mu.Unlock()

	for _, listener := range listeners {
		listener(server)
	}

	return server, nil
}

// CreateRouter creates a router.
func (w *Worker) CreateRouter(options *RouterOptions) (*Router, error) {
	return w.CreateRouterContext(context.Background(), options)
}

func (w *Worker) CreateRouterContext(ctx context.Context, options *RouterOptions) (*Router, error) {
	w.logger.DebugContext(ctx, "CreateRouter()")

	rtpCapabilities, err := generateRouterRtpCapabilities(options.MediaCodecs)
	if err != nil {
		return nil, err
	}

	routerId := uuid(routerPrefix)
	req := &FbsWorker.CreateRouterRequestT{
		RouterId: routerId,
	}
	_, err = w.channel.Request(ctx, &FbsRequest.RequestT{
		Method: FbsRequest.MethodWORKER_CREATE_ROUTER,
		Body: &FbsRequest.BodyT{
			Type:  FbsRequest.BodyWorker_CreateRouterRequest,
			Value: req,
		},
	})
	if err != nil {
		return nil, err
	}
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil, ErrWorkerClosed
	}
	data := &routerData{
		RouterId:        routerId,
		RtpCapabilities: rtpCapabilities,
		AppData:         orElse(options.AppData == nil, H{}, options.AppData),
	}
	router := newRouter(w.channel, w.logger, data)
	w.routers.Store(router.Id(), router)
	router.OnClose(func() {
		w.routers.Delete(router.Id())
	})

	listeners := w.newRouterListeners
	w.mu.Unlock()

	for _, listener := range listeners {
		listener(router)
	}

	return router, nil
}

func (w *Worker) OnNewWebRtcServer(listener func(*WebRtcServer)) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	w.newWebRtcServerListeners = append(w.newWebRtcServerListeners, listener)
}

func (w *Worker) OnNewRouter(listener func(*Router)) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	w.newRouterListeners = append(w.newRouterListeners, listener)
}

func (w *Worker) processQuited() {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return
	}
	w.closed = true
	w.mu.Unlock()

	w.cleanupAfterClosed(context.Background())
}

func (w *Worker) cleanupAfterClosed(ctx context.Context) {
	w.channel.Close(ctx)

	// close all pipes
	for _, file := range w.cmd.ExtraFiles {
		file.Close()
	}

	w.routers.Range(func(key, value any) bool {
		value.(*Router).workerClosed()
		w.routers.Delete(key)
		return true
	})

	w.webRtcServers.Range(func(key, value any) bool {
		value.(*WebRtcServer).workerClosed()
		w.webRtcServers.Delete(key)
		return true
	})

	w.notifyClosed()
}
