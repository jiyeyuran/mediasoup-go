package mediasoup

import "testing"

func TestParseScalabilityMode(t *testing.T) {
	tests := []struct {
		input    string
		expected ScalabilityMode
	}{
		{
			input: "L1T3",
			expected: ScalabilityMode{
				SpatialLayers:  1,
				TemporalLayers: 3,
				Ksvc:           false,
			},
		},
		{
			input: "L3T2_KEY",
			expected: ScalabilityMode{
				SpatialLayers:  3,
				TemporalLayers: 2,
				Ksvc:           true,
			},
		},
		{
			input: "S2T3",
			expected: ScalabilityMode{
				SpatialLayers:  2,
				TemporalLayers: 3,
				Ksvc:           false,
			},
		},
		{
			input: "foo",
			expected: ScalabilityMode{
				SpatialLayers:  1,
				TemporalLayers: 1,
				Ksvc:           false,
			},
		},
		{
			input: "",
			expected: ScalabilityMode{
				SpatialLayers:  1,
				TemporalLayers: 1,
				Ksvc:           false,
			},
		},
		{
			input: "S0T3",
			expected: ScalabilityMode{
				SpatialLayers:  1,
				TemporalLayers: 1,
				Ksvc:           false,
			},
		},
		{
			input: "S1T0",
			expected: ScalabilityMode{
				SpatialLayers:  1,
				TemporalLayers: 1,
				Ksvc:           false,
			},
		},
		{
			input: "L20T3",
			expected: ScalabilityMode{
				SpatialLayers:  20,
				TemporalLayers: 3,
				Ksvc:           false,
			},
		},
		{
			input: "S200T3",
			expected: ScalabilityMode{
				SpatialLayers:  1,
				TemporalLayers: 1,
				Ksvc:           false,
			},
		},
		{
			input: "L4T7_KEY_SHIFT",
			expected: ScalabilityMode{
				SpatialLayers:  4,
				TemporalLayers: 7,
				Ksvc:           true,
			},
		},
	}

	for _, tt := range tests {
		result := parseScalabilityMode(tt.input)
		if result != tt.expected {
			t.Errorf("parseScalabilityMode(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
