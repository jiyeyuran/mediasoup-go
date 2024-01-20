package mediasoup

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomNumber(t *testing.T) {
	ssrc := generateRandomNumber()
	ssrcStr := strconv.FormatUint(uint64(ssrc), 10)
	assert.Len(t, ssrcStr, 9)
}
