package mediasoup

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseScalabilityMode(t *testing.T) {
	testCases := []struct {
		scalabilityMode string
		want            ScalabilityMode
	}{
		{
			scalabilityMode: "L1T3",
			want:            ScalabilityMode{SpatialLayers: 1, TemporalLayers: 3, Ksvc: false},
		},
		{
			scalabilityMode: "L3T2_KEY",
			want:            ScalabilityMode{SpatialLayers: 3, TemporalLayers: 2, Ksvc: true},
		},
		{
			scalabilityMode: "S2T3",
			want:            ScalabilityMode{SpatialLayers: 2, TemporalLayers: 3, Ksvc: false},
		},
		{
			scalabilityMode: "foo",
			want:            ScalabilityMode{SpatialLayers: 1, TemporalLayers: 1, Ksvc: false},
		},
		{
			scalabilityMode: "",
			want:            ScalabilityMode{SpatialLayers: 1, TemporalLayers: 1, Ksvc: false},
		},
		{
			scalabilityMode: "S0T3",
			want:            ScalabilityMode{SpatialLayers: 1, TemporalLayers: 1, Ksvc: false},
		},
		{
			scalabilityMode: "S1T0",
			want:            ScalabilityMode{SpatialLayers: 1, TemporalLayers: 1, Ksvc: false},
		},
		{
			scalabilityMode: "L20T3",
			want:            ScalabilityMode{SpatialLayers: 20, TemporalLayers: 3, Ksvc: false},
		},
		{
			scalabilityMode: "S200T3",
			want:            ScalabilityMode{SpatialLayers: 1, TemporalLayers: 1, Ksvc: false},
		},
		{
			scalabilityMode: "L4T7_KEY_SHIFT",
			want:            ScalabilityMode{SpatialLayers: 4, TemporalLayers: 7, Ksvc: true},
		},
	}

	for _, testCase := range testCases {
		mode := ParseScalabilityMode(testCase.scalabilityMode)
		assert.EqualValues(t, testCase.want, mode)
	}
}
