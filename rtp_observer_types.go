package mediasoup

type ActiveSpeakerObserverOptions struct {
	Interval uint16

	// AppData is custom application data.
	AppData H
}

type ActiveSpeakerObserverOption func(*ActiveSpeakerObserverOptions)

// AudioLevelObserverOptions define options to create an AudioLevelObserver.
type AudioLevelObserverOptions struct {
	// MaxEntries is maximum int of entries in the 'volumes”' event. Default 1.
	MaxEntries uint16

	// Threshold is minimum average volume (in dBvo from -127 to 0) for entries in the
	// "volumes" event.	Default -80.
	Threshold int8

	// Interval in ms for checking audio volumes. Default 1000.
	Interval uint16

	// AppData is custom application data.
	AppData H
}

type AudioLevelObserverOption func(*AudioLevelObserverOptions)

type AudioLevelObserverDominantSpeaker struct {
	// ProducerId is the dominant audio producer instance.
	Producer *Producer
}

type AudioLevelObserverVolume struct {
	// ProducerId is the audio producer instance.
	Producer *Producer

	// Volume is the average volume (in dBvo from -127 to 0) of the audio producer in the
	// last interval.
	Volume int8
}
