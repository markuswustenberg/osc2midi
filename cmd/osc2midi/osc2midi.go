package main

import (
	"flag"
	"fmt"
	"os"
	"osc2midi"
)

func main() {
	port := flag.Int("port", osc2midi.DefaultOSCPort, "the udp port to listen on for OSC messages")
	debug := flag.Bool("debug", false, "debug mode")
	midiDevice := flag.String("midi-device", "", "the midi device to use (case-insensitive substring matching)")
	flag.Parse()

	if len(flag.Args()) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	args := osc2midi.Arguments{
		ConfigFilename: flag.Args()[0],
		Port:           *port,
		Debug:          *debug,
		MidiDevice:     *midiDevice,
	}

	if err := osc2midi.Start(args); err != nil {
		fmt.Fprintln(os.Stderr, "Error starting:", err)
		os.Exit(1)
	}
}
