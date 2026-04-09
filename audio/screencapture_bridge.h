#pragma once
#include <stdint.h>

// sckDeliverSamples is implemented in Go (//export sckDeliverSamples).
// The Objective-C delegate calls it for every audio buffer received from SCStream.
extern void sckDeliverSamples(int16_t *samples, int count);

// sck_start begins ScreenCaptureKit system-audio capture.
// Returns an opaque handle on success, NULL on failure.
// sampleRate must be 16000; channels must be 1.
void *sck_start(int sampleRate, int channels);

// sck_stop stops the capture session and frees the handle.
void sck_stop(void *handle);
