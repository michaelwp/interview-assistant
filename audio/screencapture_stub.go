//go:build !darwin

package audio

import "errors"

func sckSamples() <-chan []int16        { return nil }
func startScreenCapture() error         { return errors.New("ScreenCaptureKit is only available on macOS") }
func stopScreenCapture()                {}
