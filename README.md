# Interview Assistant

A macOS CLI tool that listens to interview audio in real time, transcribes speech, and suggests answers — fully offline by default.

## How it works

1. Captures audio from your mic and/or system audio (interviewer's voice via ScreenCaptureKit)
2. Transcribes speech locally using [whisper.cpp](https://github.com/ggerganov/whisper.cpp)
3. Detects questions and generates suggested answers via a local Ollama model (or OpenAI)
4. Displays everything in a live terminal UI

## Prerequisites

| Dependency | Install |
|---|---|
| Go 1.24+ | `brew install go` |
| PortAudio | `brew install portaudio` |
| whisper-cpp | auto-installed on first run |
| Ollama | [ollama.com](https://ollama.com) — auto-started on first run |

> **Screen Recording permission** is required for dual mode (system audio capture). macOS will prompt on first use.

## Installation

```bash
git clone <repo>
cd macInterviewCracking
make build
```

The binary is written to `bin/interview-assistant`.

## Usage

```bash
# Default: local transcription + local Ollama answers
./bin/interview-assistant

# Dual mode: hear both mic and system audio (interviewer + you)
./bin/interview-assistant --mode dual

# Load your resume for profile-aware answers
./bin/interview-assistant --mode dual --profile ~/resume.pdf

# Load resume + company/job description for fully contextual answers
./bin/interview-assistant --mode dual --profile ~/resume.pdf --company ~/stripe_jd.pdf

# Use OpenAI for answers instead of local Ollama
./bin/interview-assistant --online-answers --api-key $OPENAI_API_KEY

# Use OpenAI Whisper for transcription as well
./bin/interview-assistant --online --online-answers --api-key $OPENAI_API_KEY
```

## All options

| Flag | Default | Description |
|---|---|---|
| `--mode` | `single` | `single` (mic only) or `dual` (mic + system audio) |
| `--profile` | — | Path to your resume/profile (`.txt`, `.pdf`, `.docx`) |
| `--company` | — | Path to company or job description file (`.txt`, `.pdf`, `.docx`) |
| `--model` | `llama3:latest` / `gpt-4o` | Answer model name |
| `--local-answers` | `true` | Use local Ollama model for answers |
| `--online-answers` | `false` | Use OpenAI for answers |
| `--answer-url` | `http://localhost:11434/v1` | Ollama base URL |
| `--api-key` | `$OPENAI_API_KEY` | OpenAI API key (required for `--online` / `--online-answers`) |
| `--online` | `false` | Use OpenAI Whisper API for transcription |
| `--local` | `false` | Use local whisper-cpp for transcription (default) |
| `--model-path` | `~/models/ggml-base.en.bin` | Whisper model file (auto-downloaded if absent) |
| `--whisper-bin` | _(PATH)_ | Path to `whisper-cli` binary (auto-installed if absent) |
| `--lang` | `en` | Language hint for Whisper |
| `--mic` | `-1` | Microphone device index (`--list-devices` to see options) |
| `--sys` | `-1` | System audio device index |
| `--chunk` | `4.0` | Audio chunk duration in seconds |
| `--list-devices` | — | Print available audio devices and exit |

## Keybindings

| Key | Action |
|---|---|
| `q` / `ctrl+c` | Quit |
| `c` | Clear transcript |
| `↑` / `↓` | Scroll transcript |

## First run

On first run the app will automatically:

1. Install `whisper-cpp` via Homebrew if not found
2. Download the Whisper model (`ggml-base.en.bin`, ~148 MB) if not found
3. Start the Ollama server if not running
4. Pull the Ollama model (`llama3:latest`) if not pulled

Subsequent runs skip any steps that are already satisfied.
