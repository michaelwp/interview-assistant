//go:build darwin

package audio

/*
#cgo CFLAGS: -x objective-c -mmacosx-version-min=13.0
#cgo LDFLAGS: -framework ScreenCaptureKit -framework CoreMedia -framework CoreAudio -framework Foundation -framework CoreGraphics
#include "screencapture_bridge.h"
*/
import "C"
import (
	"errors"
	"sync"
	"unsafe"
)

var (
	sckMu      sync.Mutex
	sckRawCh   chan []int16    // raw float32→int16 samples from the ObjC delegate
	sckHandle  unsafe.Pointer // opaque SCKCapture* returned by sck_start
)

// SCKSamples returns the channel that receives raw PCM samples from SCK.
// Only valid after startScreenCapture() succeeds.
func sckSamples() <-chan []int16 {
	sckMu.Lock()
	defer sckMu.Unlock()
	return sckRawCh
}

// startScreenCapture initialises a ScreenCaptureKit audio session.
func startScreenCapture() error {
	sckMu.Lock()
	defer sckMu.Unlock()

	ch := make(chan []int16, 64)
	sckRawCh = ch

	handle := C.sck_start(C.int(SampleRate), C.int(Channels))
	if handle == nil {
		sckRawCh = nil
		return errors.New("ScreenCaptureKit: failed to start capture — " +
			"check Screen Recording permission in System Settings > Privacy & Security")
	}
	sckHandle = unsafe.Pointer(handle)
	return nil
}

// stopScreenCapture tears down the SCK session.
func stopScreenCapture() {
	sckMu.Lock()
	defer sckMu.Unlock()

	if sckHandle != nil {
		C.sck_stop(sckHandle)
		sckHandle = nil
	}
	sckRawCh = nil
}

// sckDeliverSamples is called from the Objective-C delegate on each audio buffer.
// It must be exported so CGO can find it.
//
//export sckDeliverSamples
func sckDeliverSamples(samples *C.int16_t, count C.int) {
	if count == 0 {
		return
	}
	sckMu.Lock()
	ch := sckRawCh
	sckMu.Unlock()
	if ch == nil {
		return
	}

	n := int(count)
	buf := make([]int16, n)
	copy(buf, (*[1 << 28]int16)(unsafe.Pointer(samples))[:n:n])

	select {
	case ch <- buf:
	default: // drop frame if consumer is behind
	}
}
