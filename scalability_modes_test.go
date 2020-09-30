package mediasoup

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseScalabilityMode(t *testing.T) {
	testCases := []struct {
		scalabilityMode string
		want            ScalabilityMode
	}{
		{
			scalabilityMode: "S3T3_KEY",
			want: ScalabilityMode{
				SpatialLayers:  3,
				TemporalLayers: 3,
				Ksvc:           true,
			},
		},
		{
			scalabilityMode: "S3T3",
			want: ScalabilityMode{
				SpatialLayers:  3,
				TemporalLayers: 3,
				Ksvc:           false,
			},
		},
		{
			scalabilityMode: "L11T3",
			want: ScalabilityMode{
				SpatialLayers:  11,
				TemporalLayers: 3,
				Ksvc:           false,
			},
		},
		{
			scalabilityMode: "invalid",
			want: ScalabilityMode{
				SpatialLayers:  1,
				TemporalLayers: 1,
				Ksvc:           false,
			},
		},
	}

	log.Println()
	for _, testCase := range testCases {
		mode := ParseScalabilityMode(testCase.scalabilityMode)
		assert.EqualValues(t, testCase.want, mode)
	}
}
