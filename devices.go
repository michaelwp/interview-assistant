package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/gordonklaus/portaudio"
)

// virtualAudioKeywords matches known virtual audio loopback drivers.
var virtualAudioKeywords = []string{
	"blackhole", "soundflower", "loopback", "virtual", "wavtap",
}

// autoDetectDevices resolves mic and system audio device indices.
// If micHint >= 0 or sysHint >= 0 the hint is used as-is; otherwise auto-detection runs.
// Returns (micIdx, micName, sysIdx, sysName, error).
// sysIdx == -1 means no virtual audio device was found.
func autoDetectDevices(micHint, sysHint int) (int, string, int, string, error) {
	devices, err := portaudio.Devices()
	if err != nil {
		return -1, "", -1, "", err
	}

	defInput, _ := portaudio.DefaultInputDevice()

	micIdx, micName := -1, "system default"
	sysIdx, sysName := -1, ""

	for i, d := range devices {
		if d.MaxInputChannels < 1 {
			continue
		}

		lower := strings.ToLower(d.Name)
		isVirtual := false
		for _, kw := range virtualAudioKeywords {
			if strings.Contains(lower, kw) {
				isVirtual = true
				break
			}
		}

		if isVirtual {
			if sysHint >= 0 {
				if i == sysHint {
					sysIdx, sysName = i, d.Name
				}
			} else if sysIdx < 0 {
				sysIdx, sysName = i, d.Name
			}
			continue
		}

		// Real input device (non-virtual).
		if micHint >= 0 {
			if i == micHint {
				micIdx, micName = i, d.Name
			}
		} else if micIdx < 0 {
			if defInput != nil && d.Name == defInput.Name {
				micIdx, micName = i, d.Name
			} else if micIdx < 0 {
				micIdx, micName = i, d.Name
			}
		}
	}

	return micIdx, micName, sysIdx, sysName, nil
}

func printDevices() {
	devices, err := portaudio.Devices()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Available audio INPUT devices:")
	fmt.Println("─────────────────────────────────────────────────")
	found := false
	for i, d := range devices {
		if d.MaxInputChannels > 0 {
			fmt.Printf("  [%2d]  %s\n        channels: %d  sample rate: %.0f Hz\n\n",
				i, d.Name, d.MaxInputChannels, d.DefaultSampleRate)
			found = true
		}
	}
	if !found {
		fmt.Println("  No input devices found.")
	}
	fmt.Println("─────────────────────────────────────────────────")
	fmt.Println("Modes:")
	fmt.Println("  single (default)  Mic only. Answers any question it hears.")
	fmt.Println("  dual              Auto-detects mic + BlackHole. Answers only interviewer questions.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  interview-assistant --mode single")
	fmt.Println("  interview-assistant --mode dual")
	fmt.Println("  interview-assistant --mode dual --mic 5 --sys 2  # override auto-detect")
	fmt.Println()
	fmt.Println("Install BlackHole for dual mode:")
	fmt.Println("  brew install blackhole-2ch")
	fmt.Println("  Then set BlackHole as output in System Settings > Sound")
}
