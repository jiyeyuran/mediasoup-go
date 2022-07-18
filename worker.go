package mediasoup

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-version"
	"github.com/jiyeyuran/mediasoup-go/netcodec"
)

const (
	// From this version to up, mediasoup-worker uses LV protocol to communicate with wrappers.
	defaultWorkerVersion = "3.9.0"
)

var (
	// Deprecated, use WithWorkerBin option
	WorkerBin = getDefaultWorkerBin()
	// Deprecated, use WithWorkerVersion option
	WorkerVersion = getDefaultWorkerVersion()
)

func getDefaultWorkerBin() string {
	workerBin := os.Getenv("MEDIASOUP_WORKER_BIN")
	if len(workerBin) > 0 {
		return workerBin
	}
	return "/usr/local/lib/node_modules/mediasoup/worker/out/Release/mediasoup-worker"
}

func getDefaultWorkerVersion() string {
	workerVersion := os.Getenv("MEDIASOUP_WORKER_VERSION")
	if len(workerVersion) > 0 {
		return workerVersion
	}
	return defaultWorkerVersion
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
	logger Logger
	// Worker process PID.
	pid int
	// Channel instance.
	channel *Channel
	// PayloadChannel instance.
	payloadChannel *PayloadChannel
	// Closed flag.
	closed uint32
	// Died flag.
	died bool
	// Custom app data.
	appData interface{}
	// Routers map.
	routers sync.Map
	// Observer instance.
	observer IEventEmitter

	// spawnDone indices child is started
	spawnDone uint32
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

	logger.Debug("constructor()")

	producerReader, producerWriter, err := os.Pipe()
	if err != nil {
		return
	}
	consumerReader, consumerWriter, err := os.Pipe()
	if err != nil {
		return
	}
	payloadProducerReader, payloadProducerWriter, err := os.Pipe()
	if err != nil {
		return
	}
	payloadConsumerReader, payloadConsumerWriter, err := os.Pipe()
	if err != nil {
		return
	}
	var (
		channelCodec, payloadChannelCodec netcodec.Codec
		workerVersion                     = settings.WorkerVersion
	)
	defautVerionFormatted, _ := version.NewVersion(defaultWorkerVersion)
	workerVersionFormatted, err := version.NewVersion(workerVersion)
	if err != nil {
		return
	}
	// use netstring codec for old mediasoup-worker version
	if workerVersionFormatted.LessThan(defautVerionFormatted) {
		channelCodec = netcodec.NewNetstringCodec(producerWriter, consumerReader)
		payloadChannelCodec = netcodec.NewNetstringCodec(payloadProducerWriter, payloadConsumerReader)
	} else {
		channelCodec = netcodec.NewNetLVCodec(producerWriter, consumerReader, hostByteOrder())
		payloadChannelCodec = netcodec.NewNetLVCodec(payloadProducerWriter, payloadConsumerReader, hostByteOrder())
	}

	bin := settings.WorkerBin
	args := settings.Args()
	if wrappedArgs := strings.Fields(settings.WorkerBin); len(wrappedArgs) > 1 {
		bin = wrappedArgs[0]
		args = append(wrappedArgs[1:], args...)
	}
	logger.Debug("spawning worker process: %s %s", bin, strings.Join(args, " "))

	child := exec.Command(bin, args...)
	child.ExtraFiles = []*os.File{producerReader, consumerWriter, payloadProducerReader, payloadConsumerWriter}
	child.Env = []string{"MEDIASOUP_VERSION=" + workerVersion}

	stderr, err := child.StderrPipe()
	if err != nil {
		return
	}
	stdout, err := child.StdoutPipe()
	if err != nil {
		return
	}
	if err = child.Start(); err != nil {
		return
	}

	pid := child.Process.Pid
	channel := newChannel(channelCodec, pid)
	payloadChannel := newPayloadChannel(payloadChannelCodec)
	workerLogger := NewLogger(fmt.Sprintf("worker[pid:%d]", pid))

	go func() {
		r := bufio.NewReader(stderr)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}
			workerLogger.Error("(stderr) %s", line)
		}
	}()

	go func() {
		r := bufio.NewReader(stdout)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}
			workerLogger.Debug("(stdout) %s", line)
		}
	}()

	worker = &Worker{
		IEventEmitter:  NewEventEmitter(),
		logger:         logger,
		pid:            pid,
		channel:        channel,
		payloadChannel: payloadChannel,
		appData:        settings.AppData,
		observer:       NewEventEmitter(),
	}

	doneCh := make(chan error)

	channel.Once(strconv.Itoa(pid), func(event string) {
		if atomic.CompareAndSwapUint32(&worker.spawnDone, 0, 1) && event == "running" {
			logger.Debug("worker process running [pid:%d]", pid)
			worker.Emit("@success")
			close(doneCh)
		}
	})
	worker.Once("@failure", func(err error) { doneCh <- err })

	go worker.wait(child)

	// start to read channel data
	channel.Start()
	// start to read payload channel data
	payloadChannel.Start()

	waitTimer := time.NewTimer(time.Second)
	defer waitTimer.Stop()

	select {
	case err = <-doneCh:
	case <-waitTimer.C:
		err = errors.New("channel timeout")
	}

	return
}

func (w *Worker) wait(child *exec.Cmd) {
	if w.Closed() {
		return
	}

	// clean up unix descriptors
	defer func() {
		w.channel.Close()
		w.payloadChannel.Close()
		for _, extraFile := range child.ExtraFiles {
			extraFile.Close()
		}
	}()

	var code int
	var signal = os.Interrupt

	if exiterr, ok := child.Wait().(*exec.ExitError); ok {
		// The worker has exited with an exit code != 0
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			code = status.ExitStatus()

			if status.Signaled() {
				signal = status.Signal()
			} else {
				signal = status.StopSignal()
			}
		}
	}

	if atomic.CompareAndSwapUint32(&w.spawnDone, 0, 1) {
		if code == 42 {
			w.logger.Error("worker process failed due to wrong settings [pid:%d]", w.pid)

			w.Close()
			w.Emit("@failure", NewTypeError("wrong settings"))
		} else {
			w.logger.Error("worker process failed unexpectedly [pid:%d, code:%d, signal:%s]",
				w.pid, code, signal)

			w.Close()
			w.Emit("@failure", fmt.Errorf(`[pid:%d, code:%d, signal:%s]`, w.pid, code, signal))
		}
	} else {
		w.logger.Error("worker process died unexpectedly [pid:%d, code:%d, signal:%s]", w.pid, code, signal)
		w.workerDied(fmt.Errorf("[pid:%d, code:%d, signal:%s]", w.pid, code, signal))
	}
}

/**
 * Worker process identifier (PID).
 */
func (w *Worker) Pid() int {
	return w.pid
}

/**
 * Whether the Worker is closed.
 */
func (w *Worker) Closed() bool {
	return atomic.LoadUint32(&w.closed) > 0
}

/**
 * Whether the Worker is died.
 */
func (w *Worker) Died() bool {
	return w.died
}

/**
 * App custom data.
 */
func (w *Worker) AppData() interface{} {
	return w.appData
}

// Observer
func (w *Worker) Observer() IEventEmitter {
	return w.observer
}

/**
 * Close the Worker.
 */
func (w *Worker) Close() {
	if !atomic.CompareAndSwapUint32(&w.closed, 0, 1) {
		return
	}

	w.logger.Debug("close()")

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

	// Close every Router.
	w.routers.Range(func(key, value interface{}) bool {
		router := value.(*Router)
		router.workerClosed()
		return true
	})
	w.routers = sync.Map{}

	// Emit observer event.
	w.observer.SafeEmit("close")
}

// Dump Worker.
func (w *Worker) Dump() (dump WorkerDump, err error) {
	w.logger.Debug("dump()")

	err = w.channel.Request("worker.dump", nil).Unmarshal(&dump)

	return
}

/**
 * Get mediasoup-worker process resource usage.
 */
func (w *Worker) GetResourceUsage() (usage WorkerResourceUsage, err error) {
	w.logger.Debug("getResourceUsage()")

	resp := w.channel.Request("worker.getResourceUsage", nil)
	err = resp.Unmarshal(&usage)

	return
}

// UpdateSettings Update settings.
func (w *Worker) UpdateSettings(settings WorkerUpdateableSettings) error {
	w.logger.Debug("updateSettings()")

	return w.channel.Request("worker.updateSettings", nil, settings).Err()
}

// CreateRouter creates a router.
func (w *Worker) CreateRouter(options RouterOptions) (router *Router, err error) {
	w.logger.Debug("createRouter()")

	internal := internalData{RouterId: uuid.NewString()}

	rsp := w.channel.Request("worker.createRouter", internal, nil)
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

func (w *Worker) workerDied(err error) {
	if !atomic.CompareAndSwapUint32(&w.closed, 0, 1) {
		return
	}
	w.died = true
	w.logger.Debug(`died() [error:%s]`, err)

	// Close the Channel instance.
	w.channel.Close()

	// Close the PayloadChannel instance.
	w.payloadChannel.Close()

	// Close every Router.
	w.routers.Range(func(key, value interface{}) bool {
		value.(*Router).workerClosed()
		return true
	})
	w.routers = sync.Map{}

	w.SafeEmit("died", err)

	// Emit observer event.
	w.observer.SafeEmit("close")
}
