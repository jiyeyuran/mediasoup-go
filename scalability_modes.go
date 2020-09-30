package mediasoup

import (
	"regexp"
	"strconv"
)

var scalabilityModeRegex = regexp.MustCompile(`^[LS]([1-9]\d{0,1})T([1-9]\d{0,1})(_KEY)?`)

type ScalabilityMode struct {
	SpatialLayers  int  `json:"spatialLayers,omitempty"`
	TemporalLayers int  `json:"temporalLayers,omitempty"`
	Ksvc           bool `json:"ksvc,omitempty"`
}

func ParseScalabilityMode(scalabilityMode string) ScalabilityMode {
	match := scalabilityModeRegex.FindStringSubmatch(scalabilityMode)

	if len(match) == 4 {
		spatialLayers, _ := strconv.Atoi(match[1])
		temporalLayers, _ := strconv.Atoi(match[2])

		return ScalabilityMode{
			SpatialLayers:  spatialLayers,
			TemporalLayers: temporalLayers,
			Ksvc:           len(match[3]) > 0,
		}
	} else {
		return ScalabilityMode{
			SpatialLayers:  1,
			TemporalLayers: 1,
			Ksvc:           false,
		}
	}
}
