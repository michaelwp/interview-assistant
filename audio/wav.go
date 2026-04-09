package audio

import (
	"bytes"
	"encoding/binary"
)

// EncodeWAV wraps int16 PCM samples in a WAV container.
// Whisper accepts WAV (PCM16, mono, 16 kHz).
func EncodeWAV(samples []int16, sampleRate, channels int) []byte {
	dataBytes := len(samples) * 2 // 2 bytes per int16 sample
	buf := new(bytes.Buffer)

	// RIFF header
	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(36+dataBytes)) // chunk size
	buf.WriteString("WAVE")

	// fmt sub-chunk
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))                          // sub-chunk size
	binary.Write(buf, binary.LittleEndian, uint16(1))                           // PCM format
	binary.Write(buf, binary.LittleEndian, uint16(channels))                    // num channels
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate))                  // sample rate
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate*channels*2))       // byte rate
	binary.Write(buf, binary.LittleEndian, uint16(channels*2))                  // block align
	binary.Write(buf, binary.LittleEndian, uint16(16))                          // bits per sample

	// data sub-chunk
	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, uint32(dataBytes))
	binary.Write(buf, binary.LittleEndian, samples)

	return buf.Bytes()
}
