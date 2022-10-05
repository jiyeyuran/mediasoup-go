package mediasoup

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
	"github.com/jiyeyuran/mediasoup-go/netcodec"
)

var (
	// WorkerBin indicates the worker binary path
	WorkerBin = getDefaultWorkerBin()
	// WorkerVersion indicates the worker binary version
	WorkerVersion = os.Getenv("MEDIASOUP_WORKER_VERSION")
)

func getDefaultWorkerBin() string {
	workerBin := os.Getenv("MEDIASOUP_WORKER_BIN")
	if len(workerBin) > 0 {
		return workerBin
	}
	return "/usr/local/lib/node_modules/mediasoup/worker/out/Release/mediasoup-worker"
}

type WorkerLogLevel string

const (
	WorkerLogLevel_Debug WorkerLogLevel = "debug"
	WorkerLogLevel_Warn  WorkerLogLevel = "warn"
	WorkerLogLevel_Error WorkerLogLevel = "error"
	WorkerLogLevel_None  WorkerLogLevel = "none"
)

type WorkerLogTag string

const (
	WorkerLogTag_INFO      WorkerLogTag = "info"
	WorkerLogTag_ICE       WorkerLogTag = "ice"
	WorkerLogTag_DTLS      WorkerLogTag = "dtls"
	WorkerLogTag_RTP       WorkerLogTag = "rtp"
	WorkerLogTag_SRTP      WorkerLogTag = "srtp"
	WorkerLogTag_RTCP      WorkerLogTag = "rtcp"
	WorkerLogTag_RTX       WorkerLogTag = "rtx"
	WorkerLogTag_BWE       WorkerLogTag = "bwe"
	WorkerLogTag_Score     WorkerLogTag = "score"
	WorkerLogTag_Simulcast WorkerLogTag = "simulcast"
	WorkerLogTag_SVC       WorkerLogTag = "svc"
	WorkerLogTag_SCTP      WorkerLogTag = "sctp"
	WorkerLogTag_Message   WorkerLogTag = "message"
)

/**
 * An object with the fields of the uv_rusage_t struct.
 *
 * - http//docs.libuv.org/en/v1.x/misc.html#c.uv_rusage_t
 * - https//linux.die.net/man/2/getrusage
 */
type WorkerResourceUsage struct {
	/**
	 * User CPU time used (in ms).
	 */
	RU_Utime int64 `json:"ru_utime"`

	/**
	 * System CPU time used (in ms).
	 */
	RU_Stime int64 `json:"ru_stime"`

	/**
	 * Maximum resident set size.
	 */
	RU_Maxrss int64 `json:"ru_maxrss"`

	/**
	 * Integral shared memory size.
	 */
	RU_Ixrss int64 `json:"ru_ixrss"`

	/**
	 * Integral unshared data size.
	 */
	RU_Idrss int64 `json:"ru_idrss"`

	/**
	 * Integral unshared stack size.
	 */
	RU_Isrss int64 `json:"ru_isrss"`

	/**
	 * Page reclaims (soft page faults).
	 */
	RU_Minflt int64 `json:"ru_minflt"`

	/**
	 * Page faults (hard page faults).
	 */
	RU_Majflt int64 `json:"ru_majflt"`

	/**
	 * Swaps.
	 */
	RU_Nswap int64 `json:"ru_nswap"`

	/**
	 * Block input operations.
	 */
	RU_Inblock int64 `json:"ru_inblock"`

	/**
	 * Block output operations.
	 */
	RU_Oublock int64 `json:"ru_oublock"`

	/**
	 * IPC messages sent.
	 */
	RU_Msgsnd int64 `json:"ru_msgsnd"`

	/**
	 * IPC messages received.
	 */
	RU_Msgrcv int64 `json:"ru_msgrcv"`

	/**
	 * Signals received.
	 */
	RU_Nsignals int64 `json:"ru_nsignals"`

	/**
	 * Voluntary context switches.
	 */
	RU_Nvcsw int64 `json:"ru_nvcsw"`

	/**
	 * Involuntary context switches.
	 */
	RU_Nivcsw int64 `json:"ru_nivcsw"`
}

type Option func(w *WorkerSettings)

/**
 * Worker
 * @emits died - (error: Error)
 * @emits @success
 * @emits @failure - (error: Error)
 */
type Worker struct {
	IEventEmitter
	// Worker logger.
	logger logr.Logger
	// Worker process PID.
	pid int
	// Channel instance.
	channel *Channel
	// PayloadChannel instance.
	payloadChannel *PayloadChannel
	// Closed flag.
	closed uint32
	// Custom app data.
	appData interface{}
	// WebRtcServers map
	webRtcServers sync.Map
	// Routers map.
	routers sync.Map
	// child is the worker process
	child *exec.Cmd
	// diedErr indices worker process stopped unexpectly
	diedErr error
	// waitCh notify worker process stopped expectly or not
	waitCh chan error

	// Observer instance.
	observer IEventEmitter
}

func NewWorker(options ...Option) (worker *Worker, err error) {
	logger := NewLogger("Worker")
	settings := &WorkerSettings{
		WorkerBin:     WorkerBin,
		WorkerVersion: WorkerVersion,
		LogLevel:      WorkerLogLevel_Error,
		RtcMinPort:    10000,
		RtcMaxPort:    59999,
		AppData:       H{},
	}

	for _, option := range options {
		option(settings)
	}

	logger.V(1).Info("constructor()")

	var (
		useLVCodec         bool
		useNewCloseMethods bool
		workerVersion      = settings.WorkerVersion
	)

	if len(workerVersion) == 0 {
		useLVCodec = detectNetCodec(settings, netcodec.NewNetLVCodec)
		useNewCloseMethods = detectNewCloseMethods(settings.WorkerBin)
		workerVersion = "0.0.0"
	} else {
		formatedWorkerVersion, err := version.NewVersion(workerVersion)
		if err != nil {
			return nil, err
		}
		// From this version to up, mediasoup-worker uses LV protocol to communicate with wrappers.
		usingLVVerion, _ := version.NewVersion("3.9.0")
		useLVCodec = formatedWorkerVersion.GreaterThanOrEqual(usingLVVerion)

		// From this version to up, mediasoup-worker uses new close methods to interact with wrappers.
		usingNewCloseMethodsVerion, _ := version.NewVersion("3.10.6")
		useNewCloseMethods = formatedWorkerVersion.GreaterThanOrEqual(usingNewCloseMethodsVerion)
	}

	var closeIfError []io.Closer
	defer func() {
		if err != nil {
			for i := len(closeIfError) - 1; i >= 0; i-- {
				closeIfError[i].Close()
			}
		}
	}()

	producerReader, producerWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	closeIfError = append(closeIfError, producerReader, producerWriter)

	consumerReader, consumerWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	closeIfError = append(closeIfError, consumerReader, consumerWriter)

	payloadProducerReader, payloadProducerWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	closeIfError = append(closeIfError, payloadProducerReader, payloadProducerWriter)

	payloadConsumerReader, payloadConsumerWriter, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	closeIfError = append(closeIfError, payloadConsumerReader, payloadConsumerWriter)

	var newCodec func(w io.WriteCloser, r io.ReadCloser) netcodec.Codec

	if useLVCodec {
		newCodec = netcodec.NewNetLVCodec
	} else {
		newCodec = netcodec.NewNetStringCodec
	}

	channelCodec := newCodec(producerWriter, consumerReader)
	payloadChannelCodec := newCodec(payloadProducerWriter, payloadConsumerReader)

	bin := settings.WorkerBin
	args := settings.Args()
	if wrappedArgs := strings.Fields(settings.WorkerBin); len(wrappedArgs) > 1 {
		bin = wrappedArgs[0]
		args = append(wrappedArgs[1:], args...)
	}
	logger.V(1).Info("spawning worker process", "bin", bin, "args", strings.Join(args, " "))

	child := exec.Command(bin, args...)
	child.ExtraFiles = []*os.File{producerReader, consumerWriter, payloadProducerReader, payloadConsumerWriter}
	child.Env = []string{"MEDIASOUP_VERSION=" + workerVersion}

	// pipe is closed by cmd
	stderr, err := child.StderrPipe()
	if err != nil {
		return nil, err
	}
	// pipe is closed by cmd
	stdout, err := child.StdoutPipe()
	if err != nil {
		return nil, err
	}

	// notify the worker process is running or stopped with error
	doneCh := make(chan error)
	// spawnDone indices the worker process is started
	spawnDone := uint32(0)

	// start worker process
	if err = child.Start(); err != nil {
		return nil, err
	}

	// the worker process id
	pid := child.Process.Pid
	channel := newChannel(channelCodec, pid, useNewCloseMethods)
	payloadChannel := newPayloadChannel(payloadChannelCodec)

	channel.Once(strconv.Itoa(pid), func(event string) {
		if atomic.CompareAndSwapUint32(&spawnDone, 0, 1) && event == "running" {
			logger.V(1).Info("worker process running", "pid", pid)
			close(doneCh)
		}
	})

	// start to read channel data
	channel.Start()
	// start to read payload channel data
	payloadChannel.Start()

	closeIfError = append(closeIfError, channel, payloadChannel)

	worker = &Worker{
		IEventEmitter:  NewEventEmitter(),
		logger:         logger,
		pid:            pid,
		channel:        channel,
		payloadChannel: payloadChannel,
		appData:        settings.AppData,
		child:          child,
		waitCh:         make(chan error, 1),
		observer:       NewEventEmitter(),
	}

	go worker.wait(child, &spawnDone, doneCh)

	waitTimer := time.NewTimer(time.Second)
	defer waitTimer.Stop()

	select {
	case err = <-doneCh:
	case <-waitTimer.C:
		err = errors.New("channel timeout")
	}
	if err != nil {
		return nil, err
	}

	workerLogger := NewLogger(fmt.Sprintf("worker[pid:%d]", pid))

	go func() {
		r := bufio.NewReader(stderr)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}
			workerLogger.Error(nil, "(stderr) "+string(line))
		}
	}()

	go func() {
		r := bufio.NewReader(stdout)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}
			workerLogger.V(1).Info("(stdout) " + string(line))
		}
	}()

	return worker, nil
}

func (w *Worker) wait(child *exec.Cmd, spawnDone *uint32, doneCh chan error) {
	var (
		code   int
		signal = os.Interrupt
	)

	if exiterr, ok := child.Wait().(*exec.ExitError); ok {
		// The worker has exited with an exit code != 0
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			if code = status.ExitStatus(); status.Signaled() {
				signal = status.Signal()
			} else {
				signal = status.StopSignal()
			}
		}
	}

	if atomic.CompareAndSwapUint32(spawnDone, 0, 1) {
		if code == 42 {
			err := NewTypeError("worker process failed due to wrong settings")
			w.logger.Error(err, "process failed", "pid", w.pid)
			doneCh <- err
		} else {
			err := errors.New("worker process failed unexpectedly")
			w.logger.Error(err, "process failed", "pid", w.pid, "code", code, "signal", signal)
			doneCh <- err
		}
	} else if !w.Closed() {
		w.diedErr = errors.New("worker process died unexpectedly")
		w.logger.Error(w.diedErr, "process died", "pid", w.pid, "code", code, "signal", signal)
		w.Close()
	}
}

// Wait blocks until the worker process is stopped
func (w *Worker) Wait() error {
	return <-w.waitCh
}

// Pid is the worker process identifier.
func (w *Worker) Pid() int {
	return w.pid
}

// Closed indices if the worker process is closed
func (w *Worker) Closed() bool {
	return atomic.LoadUint32(&w.closed) > 0
}

// Died indices if the worker process died
func (w *Worker) Died() bool {
	return w.diedErr != nil
}

// AppData returns the custom app data.
func (w *Worker) AppData() interface{} {
	return w.appData
}

// Observer return
func (w *Worker) Observer() IEventEmitter {
	return w.observer
}

// Just for testing purposes.
func (w *Worker) webRtcServersForTesting() (servers []*WebRtcServer) {
	w.webRtcServers.Range(func(key, value interface{}) bool {
		servers = append(servers, value.(*WebRtcServer))
		return true
	})
	return
}

// Just for testing purposes.
func (w *Worker) routersForTesting() (routers []*Router) {
	w.routers.Range(func(key, value interface{}) bool {
		routers = append(routers, value.(*Router))
		return true
	})
	return
}

// Close terminal the worker process and all allocated resouces
func (w *Worker) Close() {
	if !atomic.CompareAndSwapUint32(&w.closed, 0, 1) {
		return
	}

	w.logger.V(1).Info("close()")

	// Kill the worker process.
	if pid := w.Pid(); pid > 0 {
		if process, err := os.FindProcess(pid); err == nil {
			process.Signal(syscall.SIGTERM)
			process.Signal(os.Kill)
		}
	}

	// Close the Channel instance.
	w.channel.Close()

	// Close the PayloadChannel instance.
	w.payloadChannel.Close()

	// close all pipes
	for _, file := range w.child.ExtraFiles {
		file.Close()
	}

	// Close every Router.
	w.routers.Range(func(key, value interface{}) bool {
		router := value.(*Router)
		router.workerClosed()
		return true
	})
	w.routers = sync.Map{}

	// Close every WebRtcServer.
	w.webRtcServers.Range(func(key, value interface{}) bool {
		webRtcServer := value.(*WebRtcServer)
		webRtcServer.workerClosed()
		return true
	})
	w.webRtcServers = sync.Map{}

	// notify caller
	w.waitCh <- w.diedErr

	if w.diedErr != nil {
		w.SafeEmit("died", w.diedErr)
	}
	w.observer.SafeEmit("close")
}

// Dump returns the resources allocated by the worker.
func (w *Worker) Dump() (dump WorkerDump, err error) {
	w.logger.V(1).Info("dump()")

	err = w.channel.Request("worker.dump", internalData{}).Unmarshal(&dump)
	return
}

/**
 * GetResourceUsage returns the worker process resource usage.
 */
func (w *Worker) GetResourceUsage() (usage WorkerResourceUsage, err error) {
	w.logger.V(1).Info("getResourceUsage()")

	resp := w.channel.Request("worker.getResourceUsage", internalData{})
	err = resp.Unmarshal(&usage)
	return
}

// UpdateSettings updates settings.
func (w *Worker) UpdateSettings(settings WorkerUpdateableSettings) error {
	w.logger.V(1).Info("updateSettings()")

	return w.channel.Request("worker.updateSettings", internalData{}, settings).Err()
}

// CreateWebRtcServer creates a WebRtcServer.
func (w *Worker) CreateWebRtcServer(options WebRtcServerOptions) (webRtcServer *WebRtcServer, err error) {
	w.logger.V(1).Info("createWebRtcServer()")

	internal := internalData{WebRtcServerId: uuid.NewString()}
	reqData := H{
		"webRtcServerId": internal.WebRtcServerId,
		"listenInfos":    options.ListenInfos,
	}
	err = w.channel.Request("worker.createWebRtcServer", internal, reqData).Err()
	if err != nil {
		return
	}
	webRtcServer = NewWebRtcServer(webrtcServerParams{
		internal: internal,
		channel:  w.channel,
		appData:  options.AppData,
	})

	w.webRtcServers.Store(webRtcServer.Id(), webRtcServer)
	webRtcServer.On("@close", func() {
		w.webRtcServers.Delete(webRtcServer.Id())
	})

	// Emit observer event.
	w.observer.SafeEmit("newwebrtcserver", webRtcServer)

	return
}

// CreateRouter creates a router.
func (w *Worker) CreateRouter(options RouterOptions) (router *Router, err error) {
	w.logger.V(1).Info("createRouter()")

	internal := internalData{RouterId: uuid.NewString()}
	reqData := H{
		"routerId": internal.RouterId,
	}
	rsp := w.channel.Request("worker.createRouter", internal, reqData)
	if err = rsp.Err(); err != nil {
		return
	}

	rtpCapabilities, err := generateRouterRtpCapabilities(options.MediaCodecs)
	if err != nil {
		return
	}
	data := routerData{RtpCapabilities: rtpCapabilities}
	router = newRouter(routerParams{
		internal:       internal,
		data:           data,
		channel:        w.channel,
		payloadChannel: w.payloadChannel,
		appData:        options.AppData,
	})

	w.routers.Store(internal.RouterId, router)
	router.On("@close", func() {
		w.routers.Delete(internal.RouterId)
	})

	// Emit observer event.
	w.observer.SafeEmit("newrouter", router)

	return
}
