package ui

import (
	"macInterviewCracking/assistant"
	"macInterviewCracking/transcription"
)

// Config holds all runtime parameters passed into the UI.
type Config struct {
	APIKey              string
	Mode                string // "single" or "dual"
	MicDeviceIdx        int
	SysDeviceIdx        int
	UseScreenCaptureKit bool // use SCK instead of PortAudio for system audio
	ChunkSeconds        float64
	Language            string
	ProfileText         string // plain-text content of the interviewee's resume/profile
	ProfileName         string // filename shown in the header
	CompanyText         string // plain-text content of the company/role context
	CompanyName         string // filename shown in the header
	Transcriber         transcription.Transcriber
	Answerer            assistant.Answerer
	AnswerModelName     string // display only (e.g. "gpt-4o" or "llama3@local")
}
