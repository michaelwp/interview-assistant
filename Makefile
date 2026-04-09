BIN_DIR := bin
BINARY  := $(BIN_DIR)/interview-assistant
MODEL_DIR := models
MODEL_BIN := ggml-base.bin

.PHONY: build clean deps setup-local

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BINARY) .

clean:
	rm -rf $(BIN_DIR)

deps:
	go mod tidy

# Download the base model (required for -local mode).
# Prerequisite: brew install whisper-cpp
setup-local:
	mkdir -p $(MODEL_DIR)
	curl -L --progress-bar -o $(MODEL_DIR)/$(MODELS_BIN) https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin
