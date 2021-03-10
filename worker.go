package mediasoup

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"

	uuid "github.com/satori/go.uuid"
)

const VERSION = "3.6.33"

type WorkerLogLevel string

const (
	WorkerLogLevel_Debug WorkerLogLevel = "debug"
	WorkerLogLevel_Warn                 = "warn"
	WorkerLogLevel_Error                = "error"
	WorkerLogLevel_None                 = "none"
)

type WorkerLogTag string

const (
	WorkerLogTag_INFO      WorkerLogTag = "info"
	WorkerLogTag_ICE                    = "ice"
	WorkerLogTag_DTLS                   = "dtls"
	WorkerLogTag_RTP                    = "rtp"
	WorkerLogTag_SRTP                   = "srtp"
	WorkerLogTag_RTCP                   = "rtcp"
	WorkerLogTag_RTX                    = "rtx"
	WorkerLogTag_BWE                    = "bwe"
	WorkerLogTag_Score                  = "score"
	WorkerLogTag_Simulcast              = "simulcast"
	WorkerLogTag_SVC                    = "svc"
	WorkerLogTag_SCTP                   = "sctp"
	WorkerLogTag_Message                = "message"
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

var WorkerBin string = os.Getenv("MEDIASOUP_WORKER_BIN")

func init() {
	if len(WorkerBin) == 0 {
		buildType := os.Getenv("MEDIASOUP_BUILDTYPE")

		if buildType != "Debug" {
			buildType = "Release"
		}

		var mediasoupHome = os.Getenv("MEDIASOUP_HOME")

		if len(mediasoupHome) == 0 {
			if runtime.GOOS == "windows" {
				homeDir, _ := os.UserHomeDir()
				mediasoupHome = filepath.Join(homeDir, "AppData", "Roaming", "npm", "node_modules", "mediasoup")
			} else {
				mediasoupHome = "/usr/local/lib/node_modules/mediasoup"
			}
		}

		WorkerBin = filepath.Join(mediasoupHome, "worker", "out", buildType, "mediasoup-worker")
	}
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
	// mediasoup-worker child process.
	child *exec.Cmd
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
	// Routers map.
	routers sync.Map
	// Observer instance.
	observer IEventEmitter

	// spawnDone indices child is started
	spawnDone bool
}

func NewWorker(options ...Option) (worker *Worker, err error) {
	logger := NewLogger("Worker")
	settings := &WorkerSettings{
		LogLevel:   WorkerLogLevel_Error,
		RtcMinPort: 10000,
		RtcMaxPort: 59999,
		AppData:    H{},
	}

	for _, option := range options {
		option(settings)
	}

	logger.Debug("constructor()")

	producerPair, err := createSocketPair()
	if err != nil {
		return
	}
	consumerPair, err := createSocketPair()
	if err != nil {
		return
	}
	payloadProducerPair, err := createSocketPair()
	if err != nil {
		return
	}
	payloadConsumerPair, err := createSocketPair()
	if err != nil {
		return
	}

	producerSocket, err := fileToConn(producerPair[0])
	if err != nil {
		return
	}
	consumerSocket, err := fileToConn(consumerPair[0])
	if err != nil {
		return
	}
	payloadProducerSocket, err := fileToConn(payloadProducerPair[0])
	if err != nil {
		return
	}
	payloadConsumerSocket, err := fileToConn(payloadConsumerPair[0])
	if err != nil {
		return
	}

	logger.Debug("spawning worker process: %s %s", WorkerBin, strings.Join(settings.Args(), " "))

	child := exec.Command(WorkerBin, settings.Args()...)
	child.ExtraFiles = []*os.File{producerPair[1], consumerPair[1], payloadProducerPair[1], payloadConsumerPair[1]}
	child.Env = []string{"MEDIASOUP_VERSION=" + VERSION}

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
	channel := newChannel(producerSocket, consumerSocket, pid)
	payloadChannel := newPayloadChannel(payloadProducerSocket, payloadConsumerSocket)
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
		child:          child,
		pid:            pid,
		channel:        channel,
		payloadChannel: payloadChannel,
		appData:        settings.AppData,
		observer:       NewEventEmitter(),
	}

	doneCh := make(chan error)

	channel.Once(strconv.Itoa(pid), func(event string) {
		if !worker.spawnDone && event == "running" {
			worker.spawnDone = true
			logger.Debug("worker process running [pid:%d]", pid)
			worker.Emit("@success")
			close(doneCh)
		}
	})

	go worker.wait()

	worker.Once("@failure", func(err error) { doneCh <- err })

	err = <-doneCh

	return
}

func (w *Worker) wait() {
	err := w.child.Wait()

	var code int
	var signal = os.Kill

	if exiterr, ok := err.(*exec.ExitError); ok {
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

	if !w.spawnDone {
		w.spawnDone = true

		if code == 42 {
			w.logger.Error("worker process failed due to wrong settings [pid:%d]", w.pid)
			w.Emit("@failure", NewTypeError("wrong settings"))
		} else {
			w.logger.Error("worker process failed unexpectedly [pid:%d, code:%d, signal:%s]",
				w.pid, code, signal)
			w.Emit("@failure", fmt.Errorf(`[pid:%d, code:%d, signal:%s]`, w.pid, code, signal))
		}
	} else {
		w.logger.Error("worker process died unexpectedly [pid:%d, code:%d, signal:%s]", w.pid, code, signal)
		w.SafeEmit("died", fmt.Errorf("[pid:%d, code:%d, signal:%s]", w.pid, code, signal))
	}

	w.child = nil
	w.Close()
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
	if w.child != nil {
		w.child.Process.Signal(syscall.SIGTERM)
		w.child = nil
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
	w.RemoveAllListeners()

	// Emit observer event.
	w.observer.SafeEmit("close")
	w.observer.RemoveAllListeners()
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

	internal := internalData{RouterId: uuid.NewV4().String()}

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

func createSocketPair() (file [2]*os.File, err error) {
	fd, err := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
	if err != nil {
		return
	}
	file[0] = os.NewFile(uintptr(fd[0]), "")
	file[1] = os.NewFile(uintptr(fd[1]), "")

	return
}

func fileToConn(file *os.File) (net.Conn, error) {
	defer file.Close()

	return net.FileConn(file)
}
