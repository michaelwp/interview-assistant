package audio

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
)

// Capturer manages mic capture (PortAudio) and system audio capture
// (PortAudio or ScreenCaptureKit), emitting VAD-gated Chunks.
type Capturer struct {
	micStream *portaudio.Stream
	sysStream *portaudio.Stream // non-nil when using PortAudio for system audio
	micBuf    []int16
	sysBuf    []int16

	useSCK bool

	chunks chan Chunk
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// New creates a Capturer from the given config.
func New(cfg CaptureConfig) (*Capturer, error) {
	c := &Capturer{
		micBuf: make([]int16, FramesPerBuffer),
		sysBuf: make([]int16, FramesPerBuffer),
		chunks: make(chan Chunk, 32),
		stopCh: make(chan struct{}),
		useSCK: cfg.UseScreenCaptureKit,
	}

	var err error
	c.micStream, err = openInputStream(cfg.MicDevIdx, c.micBuf)
	if err != nil {
		return nil, fmt.Errorf("open mic stream: %w", err)
	}

	if !cfg.UseScreenCaptureKit && cfg.SysDevIdx >= 0 {
		c.sysStream, err = openInputStream(cfg.SysDevIdx, c.sysBuf)
		if err != nil {
			c.micStream.Close()
			return nil, fmt.Errorf("open system audio stream: %w", err)
		}
	}

	return c, nil
}

// Start begins audio capture goroutines.
func (c *Capturer) Start() error {
	if err := c.micStream.Start(); err != nil {
		return fmt.Errorf("start mic: %w", err)
	}
	c.wg.Add(1)
	go c.captureDevice(c.micStream, c.micBuf, SpeakerMic)

	if c.sysStream != nil {
		if err := c.sysStream.Start(); err != nil {
			return fmt.Errorf("start system audio: %w", err)
		}
		c.wg.Add(1)
		go c.captureDevice(c.sysStream, c.sysBuf, SpeakerSystem)
	} else if c.useSCK {
		if err := startScreenCapture(); err != nil {
			return fmt.Errorf("ScreenCaptureKit: %w", err)
		}
		c.wg.Add(1)
		go c.captureFromSCK(SpeakerSystem)
	}

	return nil
}

// Chunks returns the read-only channel of captured speech chunks.
func (c *Capturer) Chunks() <-chan Chunk {
	return c.chunks
}

// Stop signals all goroutines to exit and waits for cleanup.
func (c *Capturer) Stop() {
	close(c.stopCh)
	c.wg.Wait()
	c.micStream.Stop()
	c.micStream.Close()
	if c.sysStream != nil {
		c.sysStream.Stop()
		c.sysStream.Close()
	}
	if c.useSCK {
		stopScreenCapture()
	}
}

// captureDevice reads FramesPerBuffer-sized frames from a PortAudio stream,
// runs them through VAD, and emits complete utterance Chunks.
func (c *Capturer) captureDevice(stream *portaudio.Stream, buf []int16, speaker Speaker) {
	defer c.wg.Done()
	vad := newVADState()
	for {
		select {
		case <-c.stopCh:
			return
		default:
		}
		if err := stream.Read(); err != nil {
			time.Sleep(5 * time.Millisecond)
			continue
		}
		frame := make([]int16, FramesPerBuffer)
		copy(frame, buf)
		if chunk, ok := vad.process(frame, speaker); ok {
			select {
			case c.chunks <- chunk:
			case <-c.stopCh:
				return
			}
		}
	}
}

// captureFromSCK reads variable-length sample batches from the ScreenCaptureKit
// callback, reframes them to FramesPerBuffer, and runs the same VAD pipeline.
func (c *Capturer) captureFromSCK(speaker Speaker) {
	defer c.wg.Done()
	vad := newVADState()
	rawCh := sckSamples()
	var remainder []int16

	for {
		select {
		case <-c.stopCh:
			return
		case batch, ok := <-rawCh:
			if !ok {
				return
			}
			all := append(remainder, batch...)
			for len(all) >= FramesPerBuffer {
				frame := make([]int16, FramesPerBuffer)
				copy(frame, all[:FramesPerBuffer])
				if chunk, ok := vad.process(frame, speaker); ok {
					select {
					case c.chunks <- chunk:
					case <-c.stopCh:
						return
					}
				}
				all = all[FramesPerBuffer:]
			}
			remainder = make([]int16, len(all))
			copy(remainder, all)
		}
	}
}

func rms(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, s := range samples {
		f := float64(s) / 32768.0
		sum += f * f
	}
	return math.Sqrt(sum / float64(len(samples)))
}
