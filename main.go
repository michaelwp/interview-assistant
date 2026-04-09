package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gordonklaus/portaudio"

	"macInterviewCracking/assistant"
	"macInterviewCracking/resume"
	"macInterviewCracking/transcription"
	"macInterviewCracking/ui"
)

const defaultModelPath = "~/models/ggml-base.en.bin"

func main() {
	var (
		apiKey      = flag.String("api-key", os.Getenv("OPENAI_API_KEY"), "OpenAI API key (required for -online and for GPT answers)")
		listDevices = flag.Bool("list-devices", false, "List available audio input devices and exit")
		mode        = flag.String("mode", "single", `Mode of operation:
  single  – mic only; answers any question heard (default)
  dual    – auto-detects mic + system audio; answers only the interviewer's questions`)
		micDevice  = flag.Int("mic", -1, "Microphone device index (-1 = auto-detect default mic)")
		sysDevice  = flag.Int("sys", -1, "System audio device index (-1 = auto-detect in dual mode)")
		chunkSec   = flag.Float64("chunk", 4.0, "Audio chunk duration in seconds (used as safety cap)")
		model        = flag.String("model", "llama3:latest", "Model used for suggested answers (local model name, or OpenAI model name when --online-answers is set)")
		localAnswers = flag.Bool("local-answers", true, "Use a local Ollama model for answer suggestions (default)")
		onlineAnswers = flag.Bool("online-answers", false, "Use OpenAI for answer suggestions instead of local Ollama")
		answerURL    = flag.String("answer-url", "", "Base URL for the local answer model (default: http://localhost:11434/v1)")
		lang       = flag.String("lang", "en", "Language hint for Whisper (e.g. en, es, fr)")
		profile    = flag.String("profile", "", "Path to interviewee resume/profile (.txt, .pdf, .docx)")
		useOnline  = flag.Bool("online", false, "Use OpenAI Whisper API for transcription (requires --api-key)")
		useLocal   = flag.Bool("local", false, "Use local offline transcription via whisper-cpp (default)")
		modelPath  = flag.String("model-path", defaultModelPath, "Path to whisper.cpp model file (auto-downloaded if absent)")
		whisperBin = flag.String("whisper-bin", "", "Path to whisper-cpp binary (default: searches PATH)")
	)
	flag.Parse()

	if err := portaudio.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "PortAudio init failed: %v\n", err)
		fmt.Fprintln(os.Stderr, "Install portaudio first: brew install portaudio")
		os.Exit(1)
	}
	defer portaudio.Terminate()

	if *listDevices {
		printDevices()
		return
	}

	transcriber := buildTranscriber(*useOnline, *useLocal, *apiKey, expandPath(*modelPath), *lang, *whisperBin)

	if *mode != "single" && *mode != "dual" {
		fmt.Fprintf(os.Stderr, "Invalid --mode %q. Use 'single' or 'dual'.\n", *mode)
		os.Exit(1)
	}

	if *mode == "single" {
		*sysDevice = -1
	}

	useSCK := false
	if *mode == "dual" {
		if *sysDevice >= 0 {
			// User explicitly chose a PortAudio device — honour it (BlackHole, etc.).
			micIdx, micName, sysIdx, sysName, err := autoDetectDevices(*micDevice, *sysDevice)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Device detection failed:", err)
				os.Exit(1)
			}
			if sysIdx < 0 {
				fmt.Fprintln(os.Stderr, "dual mode: device index not found (see --list-devices)")
				os.Exit(1)
			}
			fmt.Printf("dual mode — mic: [%d] %s  |  system audio: [%d] %s  (PortAudio)\n",
				micIdx, micName, sysIdx, sysName)
			*micDevice = micIdx
			*sysDevice = sysIdx
		} else {
			// Default: use ScreenCaptureKit — no virtual audio driver needed.
			micIdx, micName, _, _, err := autoDetectDevices(*micDevice, -1)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Device detection failed:", err)
				os.Exit(1)
			}
			fmt.Printf("dual mode — mic: [%d] %s  |  system audio: ScreenCaptureKit\n",
				micIdx, micName)
			fmt.Println("  → Grant Screen Recording permission if prompted.")
			*micDevice = micIdx
			*sysDevice = -1
			useSCK = true
		}
	}

	var profileText, profileName string
	if *profile != "" {
		text, err := resume.Load(*profile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to load profile:", err)
			os.Exit(1)
		}
		profileText = text
		profileName = filepath.Base(*profile)
		fmt.Printf("Profile loaded: %s (%d chars)\n", profileName, len(profileText))
	}

	answerer, answerModelName := buildAnswerer(*localAnswers && !*onlineAnswers, *model, *answerURL, *apiKey, profileText)

	cfg := ui.Config{
		APIKey:              *apiKey,
		Mode:                *mode,
		MicDeviceIdx:        *micDevice,
		SysDeviceIdx:        *sysDevice,
		UseScreenCaptureKit: useSCK,
		ChunkSeconds:        *chunkSec,
		Language:            *lang,
		ProfileText:         profileText,
		ProfileName:         profileName,
		Transcriber:         transcriber,
		Answerer:            answerer,
		AnswerModelName:     answerModelName,
	}

	p := tea.NewProgram(ui.New(cfg), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

// buildTranscriber validates flags and constructs the appropriate transcription backend.
// Default (no flag): local transcription with auto-downloaded ggml-base.en.bin.
func buildTranscriber(useOnline, useLocal bool, apiKey, modelPath, lang, whisperBin string) transcription.Transcriber {
	if useOnline && useLocal {
		fmt.Fprintln(os.Stderr, "Error: -online and -local are mutually exclusive.")
		os.Exit(1)
	}

	if useOnline {
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, "Error: -online requires an OpenAI API key.")
			fmt.Fprintln(os.Stderr, "Set OPENAI_API_KEY or pass --api-key <key>")
			os.Exit(1)
		}
		return transcription.NewOnline(apiKey, lang)
	}

	// Default to local when neither flag is set.
	resolvedBin, err := transcription.EnsureWhisperBin(whisperBin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	if err := transcription.EnsureModel(modelPath); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	return transcription.NewLocal(modelPath, lang, resolvedBin)
}

// buildAnswerer constructs the answer backend.
// When localAnswers is true, it uses a local Ollama model; otherwise OpenAI.
func buildAnswerer(localAnswers bool, model, answerURL, apiKey, profileText string) (assistant.Answerer, string) {
	if localAnswers {
		if err := assistant.EnsureOllamaModel(model); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		label := model + "@local"
		fmt.Printf("Answer backend: local (%s)\n", label)
		return assistant.NewLocal(model, profileText, answerURL), label
	}

	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: OpenAI API key is required for answer suggestions.")
		fmt.Fprintln(os.Stderr, "Set OPENAI_API_KEY or pass --api-key <key>, or use --local-answers for offline mode.")
		os.Exit(1)
	}
	// If the user didn't explicitly set --model, use a sensible OpenAI default.
	if model == "llama3:latest" {
		model = "gpt-4o"
	}
	fmt.Printf("Answer backend: OpenAI (%s)\n", model)
	return assistant.New(apiKey, model, profileText), model
}

// expandPath replaces a leading ~ with the user's home directory.
func expandPath(path string) string {
	if !strings.HasPrefix(path, "~/") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
