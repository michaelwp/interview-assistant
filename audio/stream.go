package audio

import (
	"fmt"

	"github.com/gordonklaus/portaudio"
)

// openInputStream opens a PortAudio input stream for the given device index.
// devIdx < 0 uses the system default input device.
func openInputStream(devIdx int, buf []int16) (*portaudio.Stream, error) {
	var dev *portaudio.DeviceInfo
	var err error

	if devIdx < 0 {
		dev, err = portaudio.DefaultInputDevice()
		if err != nil {
			return nil, err
		}
	} else {
		devices, err := portaudio.Devices()
		if err != nil {
			return nil, err
		}
		if devIdx >= len(devices) {
			return nil, fmt.Errorf("device index %d out of range (%d devices found)", devIdx, len(devices))
		}
		dev = devices[devIdx]
		if dev.MaxInputChannels < 1 {
			return nil, fmt.Errorf("device %q has no input channels", dev.Name)
		}
	}

	params := portaudio.StreamParameters{
		Input: portaudio.StreamDeviceParameters{
			Device:   dev,
			Channels: Channels,
			Latency:  dev.DefaultLowInputLatency,
		},
		SampleRate:      SampleRate,
		FramesPerBuffer: FramesPerBuffer,
	}
	return portaudio.OpenStream(params, buf)
}
