package osc2midi

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/hypebeast/go-osc/osc"
	"github.com/pkg/errors"
	"github.com/rakyll/portmidi"
	"gopkg.in/yaml.v2"
)

const (
	// DefaultOSCPort is the default UDP port to listen on.
	DefaultOSCPort = 8000
)

// Arguments for Start.
type Arguments struct {
	ConfigFilename string
	Port           int
	Debug          bool
	MidiDevice     string
}

var portMidiMutex *sync.Mutex

// Start the bridge.
func Start(args Arguments) error {
	config, err := parseConfig(args.ConfigFilename)
	if err != nil {
		return err
	}

	portMidiMutex = &sync.Mutex{}
	if err := portmidi.Initialize(); err != nil {
		return errors.Wrap(err, "could not initialize portmidi")
	}
	defer portmidi.Terminate()
	id := portmidi.DefaultOutputDeviceID()
	if args.Debug {
		for i := 0; i < portmidi.CountDevices(); i++ {
			log.Printf("%+v", portmidi.Info(portmidi.DeviceID(i)))
		}
	}

	if len(args.MidiDevice) > 0 {
		for i := 0; i < portmidi.CountDevices(); i++ {
			info := portmidi.Info(portmidi.DeviceID(i))
			if !info.IsOutputAvailable {
				continue
			}
			if strings.Contains(strings.ToLower(info.Name), strings.ToLower(args.MidiDevice)) {
				id = portmidi.DeviceID(i)
				break
			}
		}
	}
	log.Println("MIDI output to", portmidi.Info(id).Name)

	out, err := portmidi.NewOutputStream(id, 1024, 0)
	if err != nil {
		return errors.Wrap(err, "could not create midi output")
	}
	defer out.Close()

	ip, err := getLocalIPAddress()
	if err != nil {
		return errors.Wrap(err, "could not get local ip address")
	}
	addr := fmt.Sprintf("%v:%v", ip, args.Port)
	log.Println("Listening for OSC on UDP", addr)

	oscServer := &osc.Server{Addr: addr}
	for _, endpoint := range config.Endpoints {
		log.Println("Handling", endpoint.Address)
		oscServer.Handle(endpoint.Address, oscHandle(endpoint, out))
	}

	if args.Debug {
		oscServer.Handle("*", func(msg *osc.Message) {
			log.Println("Received at", msg.Address, "arguments", msg.Arguments)
		})
	}

	if err := oscServer.ListenAndServe(); err != nil {
		return errors.Wrap(err, "could not start osc server")
	}
	return nil
}

func oscHandle(endpoint Endpoint, out *portmidi.Stream) func(msg *osc.Message) {
	return func(msg *osc.Message) {
		if len(msg.Arguments) == 0 {
			return
		}
		for i, arg := range msg.Arguments {
			// Ignore further arguments
			if i == len(endpoint.CC) {
				break
			}

			cc := endpoint.CC[i]
			switch arg2 := arg.(type) {
			case float32:
				portMidiMutex.Lock()
				if err := out.WriteShort(int64(0xb0+cc.Channel-1), int64(cc.Number), int64(arg2*127)); err != nil {
					log.Println("Could not send MIDI CC for endpoint", endpoint.Address)
				}
				portMidiMutex.Unlock()
			}

		}
	}
}

func parseConfig(configFilename string) (*Config, error) {
	configFile, err := ioutil.ReadFile(configFilename)
	if err != nil {
		return nil, errors.Wrap(err, "could not read config file")
	}
	config := &Config{}
	if err := yaml.Unmarshal(configFile, config); err != nil {
		return nil, errors.Wrap(err, "could not parse config file")
	}
	return config, nil
}

func getLocalIPAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", errors.Wrap(err, "could not get network interfaces")
	}
	for _, iface := range interfaces {
		if iface.Name != "en0" {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			return "", errors.Wrapf(err, "could not get network addresses for interface %v", iface.Name)
		}
		for _, addr := range addresses {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if strings.Contains(ip.String(), ":") {
				continue
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("could not find any network interfaces")
}
