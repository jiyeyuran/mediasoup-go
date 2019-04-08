package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSupportedRtpCapabilities(t *testing.T) {
	rtpCapabilities1 := GetSupportedRtpCapabilities()
	rtpCapabilities2 := GetSupportedRtpCapabilities()

	rtpCapabilities2.Codecs = nil

	assert.NotEqual(t, rtpCapabilities1, rtpCapabilities2)
}
