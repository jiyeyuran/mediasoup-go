package mediasoup

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/jiyeyuran/mediasoup-go/netcodec"
)

func detectNewCloseMethods(workerBin string) bool {
	data, err := os.ReadFile(workerBin)
	if err != nil {
		return false
	}
	return bytes.Contains(data, []byte("worker.closeRouter"))
}

func detectNetCodec(settings *WorkerSettings, newCodec func(io.WriteCloser, io.ReadCloser) netcodec.Codec) (ok bool) {
	bin := settings.WorkerBin
	args := settings.Args()
	if wrappedArgs := strings.Fields(settings.WorkerBin); len(wrappedArgs) > 1 {
		bin = wrappedArgs[0]
		args = append(wrappedArgs[1:], args...)
	}

	var pipeFiles []io.Closer
	defer func() {
		for i := len(pipeFiles) - 1; i >= 0; i-- {
			pipeFiles[i].Close()
		}
	}()

	producerReader, producerWriter, err := os.Pipe()
	if err != nil {
		return
	}
	pipeFiles = append(pipeFiles, producerReader, producerWriter)

	consumerReader, consumerWriter, err := os.Pipe()
	if err != nil {
		return
	}
	pipeFiles = append(pipeFiles, consumerReader, consumerWriter)

	payloadProducerReader, payloadProducerWriter, err := os.Pipe()
	if err != nil {
		return
	}
	pipeFiles = append(pipeFiles, payloadProducerReader, payloadProducerWriter)

	payloadConsumerReader, payloadConsumerWriter, err := os.Pipe()
	if err != nil {
		return
	}
	pipeFiles = append(pipeFiles, payloadConsumerReader, payloadConsumerWriter)

	child := exec.Command(bin, args...)
	child.ExtraFiles = []*os.File{producerReader, consumerWriter, payloadProducerReader, payloadConsumerWriter}
	child.Env = []string{"MEDIASOUP_VERSION=" + settings.WorkerVersion}

	if err = child.Start(); err != nil {
		return
	}
	defer child.Process.Kill()

	waitTimer := time.NewTimer(time.Second)
	waitTimerCancel := make(chan struct{})
	defer func() {
		waitTimer.Stop()
		close(waitTimerCancel)
	}()

	codec := newCodec(producerWriter, consumerReader)

	go func() {
		select {
		case <-waitTimer.C:
		case <-waitTimerCancel:
		}
		codec.Close()
	}()

	for {
		data, err := codec.ReadPayload()
		if err != nil {
			return false
		}
		if len(data) > 0 && data[0] != '{' {
			continue
		}
		var msg struct {
			TargetId int    `json:"targetId,omitempty"`
			Event    string `json:"event,omitempty"`
		}
		if err = json.Unmarshal(data, &msg); err != nil {
			return false
		}
		return msg.TargetId == child.Process.Pid
	}
}
