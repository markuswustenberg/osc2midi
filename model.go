package osc2midi

// Config is the main config document.
type Config struct {
	Endpoints []Endpoint
}

// Endpoint is for OSC address and MIDI mapping definitions.
type Endpoint struct {
	Address string
	CC      []MidiCC
}

// MidiCC defines a channel and CC number to send on.
type MidiCC struct {
	Channel int
	Number  int
}
