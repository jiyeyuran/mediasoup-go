package mediasoup

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
)

type Worker struct {
	EventEmitter
	pid          int
	closed       bool
	channel      *Channel
	observer     EventEmitter
	logger       logrus.FieldLogger
	workerLogger logrus.FieldLogger
	child        *exec.Cmd
	spawnDone    bool
	routers      map[string]*Router
}

func NewWorker(workerBin string, options ...Option) (worker *Worker, err error) {
	if len(workerBin) == 0 {
		workerBin = os.Getenv("MEDIASOUP_WORKER_BIN")
	}

	opts := NewOptions()

	for _, option := range options {
		option(opts)
	}

	logger := TypeLogger("Worker")

	logger.Debug("constructor()")

	fds, err := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
	if err != nil {
		return
	}
	fd1, fd2 := fds[0], fds[1]

	socket, err := fdToFileConn(fd1)
	if err != nil {
		return
	}

	child := exec.Command(workerBin, opts.WorkerArgs()...)
	child.ExtraFiles = []*os.File{os.NewFile(uintptr(fd2), "")}
	child.Env = []string{"MEDIASOUP_VERSION=" + opts.Version}

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

	channel := NewChannel(socket, pid)

	workerLogger := TypeLogger(fmt.Sprintf(`worker[pid:%d]`, pid))

	go func() {
		r := bufio.NewReader(stderr)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}
			workerLogger.Errorf(`(stderr) %s`, line)
		}
	}()

	go func() {
		r := bufio.NewReader(stdout)
		for {
			line, _, err := r.ReadLine()
			if err != nil {
				break
			}
			workerLogger.Errorf(`(stdout) %s`, line)
		}
	}()

	worker = &Worker{
		EventEmitter: NewEventEmitter(logger),
		pid:          pid,
		channel:      channel,
		observer:     NewEventEmitter(AppLogger()),
		logger:       logger,
		workerLogger: workerLogger,
		child:        child,
		routers:      make(map[string]*Router),
	}

	channel.Once(strconv.Itoa(pid), func(event string) {
		if !worker.spawnDone && event == "running" {
			worker.spawnDone = true

			logger.Debugf("worker process running [pid:%d]", pid)

			worker.Emit("@success")
		}
	})

	go worker.wait(child)

	return
}

func (w *Worker) Pid() int {
	return w.pid
}

func (w *Worker) Closed() bool {
	return w.closed
}

func (w Worker) Observer() EventEmitter {
	return w.observer
}

func (w *Worker) Close() {
	if w.closed {
		return
	}

	w.logger.Debugln("close()")

	w.closed = true

	// Kill the worker process.
	if w.child != nil {
		w.child.Process.Signal(syscall.SIGTERM)
		w.child = nil
	}

	// Close the Channel instance.
	w.channel.Close()

	// Close every Router.
	for _, router := range w.routers {
		router.workerClosed()
	}
	w.routers = make(map[string]*Router)

	// Emit observer event.
	w.observer.SafeEmit("close")
}

// Dump Worker.
func (w *Worker) Dump() Response {
	w.logger.Debugln("dump()")

	return w.channel.Request("worker.dump", nil, nil)
}

// UpdateSettings Update settings.
func (w *Worker) UpdateSettings(logLevel string, logTags []string) Response {
	w.logger.Debugln("updateSettings()")

	return w.channel.Request("worker.updateSettings", nil, H{
		"logLevel": logLevel,
		"logTags":  logTags,
	})
}

// CreateRouter creates a router.
func (w *Worker) CreateRouter(mediaCodecs []RtpCodecCapability) (router *Router, err error) {
	w.logger.Debug("createRouter()")

	internal := Internal{RouterId: uuid.NewV4().String()}

	rsp := w.channel.Request("worker.createRouter", internal, nil)
	if err = rsp.Err(); err != nil {
		return
	}

	rtpCapabilities, err := GenerateRouterRtpCapabilities(mediaCodecs)
	if err != nil {
		return
	}
	data := RouterData{RtpCapabilities: rtpCapabilities}

	router = NewRouter(internal, data, w.channel)

	w.routers[internal.RouterId] = router
	router.On("@close", func() {
		delete(w.routers, internal.RouterId)
	})

	// Emit observer event.
	w.observer.SafeEmit("newrouter", router)

	return
}

func (w *Worker) wait(child *exec.Cmd) {
	err := child.Wait()

	w.child = nil
	w.Close()

	code, signal := 0, ""

	if exiterr, ok := err.(*exec.ExitError); ok {
		// The worker has exited with an exit code != 0
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			code = status.ExitStatus()

			if status.Signaled() {
				signal = status.Signal().String()
			} else if status.Stopped() {
				signal = status.StopSignal().String()
			}
		}
	}

	if !w.spawnDone {
		w.spawnDone = true

		if code == 42 {
			w.logger.Errorf("worker process failed due to wrong settings [pid:%d]", w.pid)

			w.Emit("@failure", NewTypeError("wrong settings"))
		} else {
			w.logger.Errorf("worker process failed unexpectedly [pid:%d, code:%d, signal:%s]",
				w.pid, code, signal)

			w.Emit("@failure", fmt.Errorf(`[pid:%d, code:%d, signal:%s]`, w.pid, code, signal))
		}
	} else {
		w.logger.Errorf("worker process died unexpectedly [pid:%d, code:%d, signal:%s]", w.pid, code, signal)

		w.SafeEmit("died", fmt.Errorf("[pid:%d, code:%d, signal:%s]", w.pid, code, signal))
	}
}

func fdToFileConn(fd int) (net.Conn, error) {
	f := os.NewFile(uintptr(fd), "")
	defer f.Close()
	return net.FileConn(f)
}
