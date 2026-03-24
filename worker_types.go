package mediasoup

import (
	"log/slog"
)

type WorkerLogLevel string

const (
	WorkerLogLevelDebug WorkerLogLevel = "debug"
	WorkerLogLevelWarn  WorkerLogLevel = "warn"
	WorkerLogLevelError WorkerLogLevel = "error"
	WorkerLogLevelNone  WorkerLogLevel = "none"
)

type WorkerLogTag string

const (
	WorkerLogTagInfo      WorkerLogTag = "info"
	WorkerLogTagIce       WorkerLogTag = "ice"
	WorkerLogTagDtls      WorkerLogTag = "dtls"
	WorkerLogTagRtp       WorkerLogTag = "rtp"
	WorkerLogTagSrtp      WorkerLogTag = "srtp"
	WorkerLogTagRtcp      WorkerLogTag = "rtcp"
	WorkerLogTagRtx       WorkerLogTag = "rtx"
	WorkerLogTagBwe       WorkerLogTag = "bwe"
	WorkerLogTagScore     WorkerLogTag = "score"
	WorkerLogTagSimulcast WorkerLogTag = "simulcast"
	WorkerLogTagSvc       WorkerLogTag = "svc"
	WorkerLogTagSctp      WorkerLogTag = "sctp"
	WorkerLogTagMessage   WorkerLogTag = "message"
)

// WorkerSettings represents the configuration settings for a worker.
type WorkerSettings struct {
	// LogLevel defines the log level for media worker subprocess logs.
	// Valid values: 'debug', 'warn', 'error', 'none'. Defaults to 'error'.
	LogLevel WorkerLogLevel `json:"logLevel,omitempty"`

	// LogTags defines debug log tags. See debugging documentation for tag details.
	LogTags []WorkerLogTag `json:"logTags,omitempty"`

	// DtlsCertificateFile is the path to PEM formatted DTLS public certificate.
	// If empty, a certificate is generated dynamically.
	DtlsCertificateFile string `json:"dtlsCertificateFile,omitempty"`

	// DtlsPrivateKeyFile is the path to PEM formatted DTLS private key.
	// If empty, a certificate is generated dynamically.
	DtlsPrivateKeyFile string `json:"dtlsPrivateKeyFile,omitempty"`

	// LibwebrtcFieldTrials sets libwebrtc field trials (advanced).
	// WARNING: Invalid values will crash the worker.
	// Defaults to "WebRTC-Bwe-AlrLimitedBackoff/Enabled/".
	LibwebrtcFieldTrials string `json:"libwebrtcFieldTrials,omitempty"`

	// DisableLiburing disables io_uring even if supported by host.
	DisableLiburing bool `json:"disableLiburing,omitempty"`

	// AppData holds custom application data.
	AppData H `json:"appData,omitempty"`

	// CustomArgs prepends additional command arguments.
	// Example for valgrind:
	//   NewWorker("valgrind", func(s *WorkerSettings) {
	//     s.CustomArgs = []string{"--leak-check=full", "./mediasoup-worker"}
	//   })
	CustomArgs []string `json:"customArgs,omitempty"`

	// Env sets environment variables for the worker process.
	Env []string

	// Logger sets the logger for API wrapper.
	Logger *slog.Logger

	// WorkerLogger sets the logger for worker process, default to Logger.
	WorkerLogger *slog.Logger
}

type WorkerUpdatableSettings struct {
	LogLevel WorkerLogLevel `json:"logLevel,omitempty"`
	LogTags  []WorkerLogTag `json:"logTags,omitempty"`
}

type WorkerDump struct {
	Pid                    uint32                            `json:"pid,omitempty"`
	WebRtcServerIds        []string                          `json:"webRtcServerIds,omitempty"`
	RouterIds              []string                          `json:"routerIds,omitempty"`
	ChannelMessageHandlers *WorkerDumpChannelMessageHandlers `json:"channelMessageHandlers,omitempty"`
	Liburing               *WorkerDumpLiburing               `json:"liburing,omitempty"`
}

type WorkerDumpChannelMessageHandlers struct {
	ChannelRequestHandlers      []string `json:"channelRequestHandlers,omitempty"`
	ChannelNotificationHandlers []string `json:"channelNotificationHandlers,omitempty"`
}

type WorkerDumpLiburing struct {
	SqeProcessCount   uint64 `json:"sqeProcessCount,omitempty"`
	SqeMissCount      uint64 `json:"sqeMissCount,omitempty"`
	UserDataMissCount uint64 `json:"userDataMissCount,omitempty"`
}

// WorkerResourceUsage represents the resource usage statistics of a worker.
// It includes various metrics related to CPU usage, memory usage, and I/O operations.
//
// http://docs.libuv.org/en/v1.x/misc.html#c.uv_rusage_t
// https://linux.die.net/man/2/getrusage
type WorkerResourceUsage struct {
	// User CPU time used (in milliseconds).
	RuUtime uint64 `json:"ru_utime"`

	// System CPU time used (in milliseconds).
	RuStime uint64 `json:"ru_stime"`

	// Maximum resident set size.
	RuMaxrss uint64 `json:"ru_maxrss"`

	// Integral shared memory size.
	RuIxrss uint64 `json:"ru_ixrss"`

	// Integral unshared data size.
	RuIdrss uint64 `json:"ru_idrss"`

	// Integral unshared stack size.
	RuIsrss uint64 `json:"ru_isrss"`

	// Page reclaims (soft page faults).
	RuMinflt uint64 `json:"ru_minflt"`

	// Page faults (hard page faults).
	RuMajflt uint64 `json:"ru_majflt"`

	// Number of swaps.
	RuNswap uint64 `json:"ru_nswap"`

	// Block input operations.
	RuInblock uint64 `json:"ru_inblock"`

	// Block output operations.
	RuOublock uint64 `json:"ru_oublock"`

	// IPC messages sent.
	RuMsgsnd uint64 `json:"ru_msgsnd"`

	// IPC messages received.
	RuMsgrcv uint64 `json:"ru_msgrcv"`

	// Signals received.
	RuNsignals uint64 `json:"ru_nsignals"`

	// Voluntary context switches.
	RuNvcsw uint64 `json:"ru_nvcsw"`

	// Involuntary context switches.
	RuNivcsw uint64 `json:"ru_nivcsw"`
}

type Option func(*WorkerSettings)
