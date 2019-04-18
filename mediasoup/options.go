package mediasoup

import (
	"fmt"
	"os"
)

// Options to start worker
type Options struct {
	Version             string   `json:"version,omitempty"`
	LogLevel            string   `json:"logLevel,omitempty"`
	LogTags             []string `json:"logTags,omitempty"`
	RTCMinPort          uint16   `json:"rtcMinPort,omitempty"`
	RTCMaxPort          uint16   `json:"rtcMaxPort,omitempty"`
	DTLSCertificateFile string   `json:"dtlsCertificateFile,omitempty"`
	DTLSPrivateKeyFile  string   `json:"dtlsPrivateKeyFile,omitempty"`
}

func NewOptions() *Options {
	version := os.Getenv("MEDIASOUP_WORKER_VERSION")

	if len(version) == 0 {
		version = "latest"
	}

	return &Options{
		Version:    version,
		LogLevel:   "error",
		RTCMinPort: 10000,
		RTCMaxPort: 59999,
	}
}

func (o *Options) WorkerArgs() []string {
	workerArgs := []string{}

	if len(o.LogLevel) > 0 {
		workerArgs = append(workerArgs, "--logLevel="+o.LogLevel)
	}

	for _, logTag := range o.LogTags {
		if len(logTag) > 0 {
			workerArgs = append(workerArgs, "--logTags="+logTag)
		}
	}

	workerArgs = append(workerArgs, fmt.Sprintf("--rtcMinPort=%d", o.RTCMinPort))
	workerArgs = append(workerArgs, fmt.Sprintf("--rtcMaxPort=%d", o.RTCMaxPort))

	if len(o.DTLSCertificateFile) > 0 && len(o.DTLSPrivateKeyFile) > 0 {
		workerArgs = append(workerArgs, "--dtlsCertificateFile="+o.DTLSCertificateFile)
		workerArgs = append(workerArgs, "--dtlsPrivateKeyFile="+o.DTLSPrivateKeyFile)
	}

	return workerArgs
}

type Option func(o *Options)

func WithVersion(version string) Option {
	return func(o *Options) {
		o.Version = version
	}
}

func WithLogLevel(logLevel string) Option {
	return func(o *Options) {
		o.LogLevel = logLevel
	}
}

func WithLogTags(logTags []string) Option {
	return func(o *Options) {
		o.LogTags = logTags
	}
}

func WithRTCMinPort(rtcMinPort uint16) Option {
	return func(o *Options) {
		o.RTCMinPort = rtcMinPort
	}
}

func WithRTCMaxPort(rtcMaxPort uint16) Option {
	return func(o *Options) {
		o.RTCMaxPort = rtcMaxPort
	}
}

func WithDTLSCert(dtlsCertificateFile, dtlsPrivateKeyFile string) Option {
	return func(o *Options) {
		o.DTLSCertificateFile = dtlsCertificateFile
		o.DTLSPrivateKeyFile = dtlsPrivateKeyFile
	}
}
