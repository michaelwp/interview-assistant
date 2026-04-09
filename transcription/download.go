package transcription

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

// modelURLs maps known ggml model filenames to their HuggingFace download URLs.
var modelURLs = map[string]string{
	"ggml-tiny.en.bin":   "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.en.bin",
	"ggml-tiny.bin":      "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin",
	"ggml-base.en.bin":   "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin",
	"ggml-base.bin":      "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin",
	"ggml-small.en.bin":  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.en.bin",
	"ggml-small.bin":     "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin",
	"ggml-medium.en.bin": "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.en.bin",
	"ggml-medium.bin":    "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.bin",
	"ggml-large-v3.bin":  "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3.bin",
}

// EnsureModel checks that the model file exists at path and downloads it if not.
// The filename must match one of the known ggml model names.
func EnsureModel(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // already present
	}

	name := filepath.Base(path)
	url, ok := modelURLs[name]
	if !ok {
		return fmt.Errorf("model file not found at %q and no known download URL for %q\n"+
			"Supported models: ggml-tiny.en.bin, ggml-base.en.bin, ggml-small.en.bin, ggml-medium.en.bin",
			path, name)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create model directory: %w", err)
	}

	fmt.Printf("Model not found. Downloading %s\n", name)

	tmp := path + ".tmp"
	if err := downloadFile(url, tmp); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("download %s: %w", name, err)
	}

	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("install model: %w", err)
	}

	fmt.Printf("Saved to %s\n", path)
	return nil
}

// EnsureWhisperBin checks that a whisper-cpp binary is available.
// If binaryPath is set, it verifies that path exists. Otherwise it searches
// PATH for "whisper-cli" (installed by `brew install whisper-cpp`) and falls
// back to the legacy name "whisper-cpp". If neither is found it installs the
// package via Homebrew and returns the resolved path.
func EnsureWhisperBin(binaryPath string) (string, error) {
	if binaryPath != "" {
		if _, err := os.Stat(binaryPath); err != nil {
			return "", fmt.Errorf("whisper binary not found at %q: %w", binaryPath, err)
		}
		return binaryPath, nil
	}

	// Prefer the current brew binary name; fall back to the old name.
	for _, name := range []string{"whisper-cli", "whisper-cpp"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	// Not in PATH — try installing via Homebrew.
	if _, err := exec.LookPath("brew"); err != nil {
		return "", fmt.Errorf("whisper-cli not found in PATH and Homebrew is not installed\n" +
			"Install whisper-cpp manually: https://github.com/ggerganov/whisper.cpp")
	}

	fmt.Println("whisper-cpp not found. Installing via Homebrew (this may take a minute)…")
	cmd := exec.Command("brew", "install", "whisper-cpp")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("brew install whisper-cpp failed: %w", err)
	}

	for _, name := range []string{"whisper-cli", "whisper-cpp"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("whisper-cli not found after brew install — try reopening your terminal")
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url) //nolint:noctx
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	total := resp.ContentLength
	var received int64
	buf := make([]byte, 32*1024)

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				return werr
			}
			received += int64(n)
			if total > 0 {
				fmt.Printf("\r  %.1f / %.1f MB  (%.0f%%)",
					float64(received)/1e6,
					float64(total)/1e6,
					float64(received)/float64(total)*100)
			} else {
				fmt.Printf("\r  %.1f MB", float64(received)/1e6)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	fmt.Println()
	return nil
}
