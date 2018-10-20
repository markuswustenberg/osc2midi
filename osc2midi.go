package osc2midi

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/hypebeast/go-osc/osc"
	"github.com/pkg/errors"
	"github.com/rakyll/portmidi"
	"gopkg.in/yaml.v2"
)

const (
	DefaultOSCPort = 8000
)

type Arguments struct {
	ConfigFilename string
	Port           *int
}

func Start(args Arguments) error {
	configFile, err := ioutil.ReadFile(args.ConfigFilename)
	if err != nil {
		return errors.Wrap(err, "could not read config file")
	}
	config := &Config{}
	if err := yaml.Unmarshal(configFile, config); err != nil {
		return errors.Wrap(err, "could not parse config file")
	}

	portmidi.Initialize()
	defer portmidi.Terminate()
	id := portmidi.DefaultOutputDeviceID()
	fmt.Fprintf(os.Stderr, "MIDI output to %v\n", portmidi.Info(id).Name)
	out, err := portmidi.NewOutputStream(id, 1024, 0)
	if err != nil {
		return errors.Wrap(err, "could not create midi output")
	}
	defer out.Close()

	ip, err := getLocalIPAddress()
	if err != nil {
		return errors.Wrap(err, "could not get local ip address")
	}
	addr := fmt.Sprintf("%v:%v", ip, *args.Port)
	fmt.Fprintln(os.Stderr, "Listening for OSC on UDP", addr)

	oscServer := &osc.Server{Addr: addr}
	for _, endpoint := range config.Endpoints {
		fmt.Fprintln(os.Stderr, "Handling", endpoint.Address)
		oscServer.Handle(endpoint.Address, func(msg *osc.Message) {
			if len(msg.Arguments) == 0 {
				return
			}
			for i, arg := range msg.Arguments {
				switch arg2 := arg.(type) {
				case float32:
					if endpoint.CC != nil {
						out.WriteShort(int64(0xb0+endpoint.CC.Channel-1), int64(endpoint.CC.Number+i), int64(arg2*127))
					}
				}

			}
		})
	}

	if err := oscServer.ListenAndServe(); err != nil {
		return errors.Wrap(err, "could not start osc server")
	}
	return nil
}

func getLocalIPAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		errors.Wrap(err, "could not get network interfaces")
	}
	for _, iface := range interfaces {
		if iface.Name != "en0" {
			continue
		}
		addresses, err := iface.Addrs()
		if err != nil {
			errors.Wrapf(err, "could not get network addresses for interface %v", iface.Name)
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
