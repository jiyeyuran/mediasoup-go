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

	FbsNotification "github.com/jiyeyuran/mediasoup-go/internal/FBS/Notification"
	FbsRequest "github.com/jiyeyuran/mediasoup-go/internal/FBS/Request"
	FbsTransport "github.com/jiyeyuran/mediasoup-go/internal/FBS/Transport"
	FbsWorker "github.com/jiyeyuran/mediasoup-go/internal/FBS/Worker"
	"github.com/jiyeyuran/mediasoup-go/internal/channel"
)

// Worker represents a mediasoup C++ subprocess that runs in a single CPU core and handles Router
// instances.
type Worker struct {
	baseNotifier
	cmd           *exec.Cmd
	channel       *channel.Channel
	logger        *slog.Logger
	routers       sync.Map
	webRtcServers sync.Map
	appData       H
}

// NewWorker create a Worker.
func NewWorker(workerBinaryPath string, options ...Option) (*Worker, error) {
	opts := &WorkerSettings{
		LogLevel: WorkerLogLevelWarn,
	}
	for _, opt := range options {
		opt(opts)
	}

	if opts.Logger == nil {
		output := io.Discard
		level := slog.LevelWarn
		if opts.LogLevel != WorkerLogLevelNone {
			output = os.Stdout
			level.UnmarshalText([]byte(opts.LogLevel))
		}
		opts.Logger = slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{
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

	opts.Logger.Debug(fmt.Sprintf("starting worker process: %s %s", workerBinaryPath, strings.Join(args, " ")))

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
	logger := opts.Logger.With("pid", pid)
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
		r := bufio.NewReader(stderr)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}
			logger.Error(string(line))
		}
	}()
	go func() {
		r := bufio.NewReader(stdout)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}
			logger.Info(string(line))
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
	} else {
		w.logger.Info("worker process quited")
	}

	w.doClose()
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

func (w *Worker) Close() {
	w.logger.Debug("Close()")

	if w.channel.Closed() {
		return
	}

	if process := w.cmd.Process; process != nil && w.cmd.ProcessState == nil {
		process.Signal(os.Interrupt)
		// force kill the worker process.
		time.AfterFunc(time.Second, func() {
			if w.cmd.ProcessState == nil {
				w.logger.Warn("force kill worker process")
				w.cmd.Cancel()
			}
		})
	}

	w.channel.Close()
	w.doClose()
}

func (w *Worker) doClose() {
	// close all pipes
	for _, file := range w.cmd.ExtraFiles {
		file.Close()
	}

	w.routers.Range(func(key, value interface{}) bool {
		value.(*Router).workerClosed()
		return true
	})

	w.webRtcServers.Range(func(key, value interface{}) bool {
		value.(*WebRtcServer).workerClosed()
		return true
	})

	w.notifyClosed()
}

func (w *Worker) Closed() bool {
	return w.channel.Closed()
}

// Dump returns the resources allocated by the worker.
func (w *Worker) Dump() (dump *WorkerDump, err error) {
	w.logger.Debug("Dump()")

	respAny, err := w.channel.Request(&FbsRequest.RequestT{
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
	w.logger.Debug("GetResourceUsage()")

	respAny, err := w.channel.Request(&FbsRequest.RequestT{
		Method: FbsRequest.MethodWORKER_GET_RESOURCE_USAGE,
	})
	if err != nil {
		return
	}
	resp := respAny.(*FbsWorker.ResourceUsageResponseT)

	return &WorkerResourceUsage{
		RuUtime:    resp.RuUtime,
		RuStime:    resp.RuStime,
		RuMaxrss:   resp.RuMaxrss,
		RuIxrss:    resp.RuIxrss,
		RuIdrss:    resp.RuIdrss,
		RuIsrss:    resp.RuIsrss,
		RuMinflt:   resp.RuMinflt,
		RuMajflt:   resp.RuMajflt,
		RuNswap:    resp.RuNswap,
		RuInblock:  resp.RuInblock,
		RuOublock:  resp.RuOublock,
		RuMsgsnd:   resp.RuMsgsnd,
		RuMsgrcv:   resp.RuMsgrcv,
		RuNsignals: resp.RuNsignals,
		RuNvcsw:    resp.RuNvcsw,
		RuNivcsw:   resp.RuNivcsw,
	}, nil
}

// UpdateSettings updates worker settings.
func (w *Worker) UpdateSettings(settings *WorkerUpdatableSettings) (err error) {
	w.logger.Debug("UpdateSettings()")

	_, err = w.channel.Request(&FbsRequest.RequestT{
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
	w.logger.Debug("CreateWebRtcServer()")

	id := uuid()

	req := &FbsWorker.CreateWebRtcServerRequestT{
		WebRtcServerId: id,
		ListenInfos:    make([]*FbsTransport.ListenInfoT, len(options.ListenInfos)),
	}

	for i, info := range options.ListenInfos {
		req.ListenInfos[i] = &FbsTransport.ListenInfoT{
			Protocol:         orElse(info.Protocol == TransportProtocolTCP, FbsTransport.ProtocolTCP, FbsTransport.ProtocolUDP),
			Ip:               info.IP,
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

	_, err := w.channel.Request(&FbsRequest.RequestT{
		Method: FbsRequest.MethodWORKER_CREATE_WEBRTCSERVER,
		Body: &FbsRequest.BodyT{
			Type:  FbsRequest.BodyWorker_CreateWebRtcServerRequest,
			Value: req,
		},
	})
	if err != nil {
		return nil, err
	}
	server := NewWebRtcServer(w, id, options.AppData)
	w.webRtcServers.Store(id, server)
	server.OnClose(func() {
		w.webRtcServers.Delete(id)
	})
	return server, nil
}

// CreateRouter creates a router.
func (w *Worker) CreateRouter(options *RouterOptions) (*Router, error) {
	w.logger.Debug("CreateRouter()")

	rtpCapabilities, err := generateRouterRtpCapabilities(options.MediaCodecs)
	if err != nil {
		return nil, err
	}

	routerId := uuid()
	req := &FbsWorker.CreateRouterRequestT{
		RouterId: routerId,
	}
	_, err = w.channel.Request(&FbsRequest.RequestT{
		Method: FbsRequest.MethodWORKER_CREATE_ROUTER,
		Body: &FbsRequest.BodyT{
			Type:  FbsRequest.BodyWorker_CreateRouterRequest,
			Value: req,
		},
	})
	if err != nil {
		return nil, err
	}
	data := &RouterData{
		RouterId:        routerId,
		RtpCapabilities: rtpCapabilities,
		AppData:         options.AppData,
	}
	router := newRouter(w.channel, w.logger, data)
	w.routers.Store(router.Id(), router)
	router.OnClose(func() {
		w.routers.Delete(router.Id())
	})
	return router, nil
}
