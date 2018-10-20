package osc2midi

type Config struct {
	Endpoints []Endpoint
}

type Endpoint struct {
	Address string
	CC *MidiCC
}

type MidiCC struct {
	Channel int
	Number int
}
