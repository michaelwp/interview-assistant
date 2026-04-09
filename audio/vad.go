package audio

import "time"

// vadState tracks voice activity detection state for one audio stream.
// Feed it FramesPerBuffer-sized frames via process(); it returns a complete
// Chunk whenever an utterance ends.
type vadState struct {
	preBuf        [][]int16
	preIdx        int
	acc           []int16
	speaking      bool
	speechFrames  int
	silenceFrames int
}

func newVADState() *vadState {
	preBuf := make([][]int16, preSpeechFrames)
	for i := range preBuf {
		preBuf[i] = make([]int16, FramesPerBuffer)
	}
	return &vadState{
		preBuf: preBuf,
		acc:    make([]int16, 0, maxChunkFrames*FramesPerBuffer),
	}
}

// process handles one FramesPerBuffer-sized frame.
// Returns (chunk, true) when an utterance ends, otherwise (zero, false).
func (v *vadState) process(frame []int16, speaker Speaker) (Chunk, bool) {
	isSpeech := rms(frame) > SilenceThreshold

	if !v.speaking {
		copy(v.preBuf[v.preIdx], frame)
		v.preIdx = (v.preIdx + 1) % preSpeechFrames

		if isSpeech {
			v.speaking = true
			v.speechFrames = 1
			v.silenceFrames = 0
			for i := 0; i < preSpeechFrames; i++ {
				v.acc = append(v.acc, v.preBuf[(v.preIdx+i)%preSpeechFrames]...)
			}
			v.acc = append(v.acc, frame...)
		}
		return Chunk{}, false
	}

	v.acc = append(v.acc, frame...)
	if isSpeech {
		v.speechFrames++
		v.silenceFrames = 0
	} else {
		v.silenceFrames++
		if v.silenceFrames >= silenceGapFrames {
			return v.flush(speaker)
		}
	}
	if len(v.acc) >= maxChunkFrames*FramesPerBuffer {
		return v.flush(speaker)
	}
	return Chunk{}, false
}

func (v *vadState) flush(speaker Speaker) (Chunk, bool) {
	defer func() {
		v.acc = v.acc[:0]
		v.speaking = false
		v.speechFrames = 0
		v.silenceFrames = 0
	}()

	if v.speechFrames < minSpeechFrames || len(v.acc) == 0 {
		return Chunk{}, false
	}
	samples := make([]int16, len(v.acc))
	copy(samples, v.acc)
	return Chunk{
		Samples:   samples,
		Speaker:   speaker,
		Timestamp: time.Now(),
		HasSpeech: true,
	}, true
}
