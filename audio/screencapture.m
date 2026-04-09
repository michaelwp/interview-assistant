// ScreenCaptureKit system-audio bridge for macOS 13.0+
// Captures all system audio without requiring a virtual audio driver (BlackHole).

#import <ScreenCaptureKit/ScreenCaptureKit.h>
#import <CoreMedia/CoreMedia.h>
#import <Foundation/Foundation.h>
#include <math.h>
#include <stdlib.h>
#include "screencapture_bridge.h"

API_AVAILABLE(macos(13.0))
@interface SCKDelegate : NSObject <SCStreamOutput, SCStreamDelegate>
@end

API_AVAILABLE(macos(13.0))
@implementation SCKDelegate

- (void)stream:(SCStream *)stream
    didOutputSampleBuffer:(CMSampleBufferRef)sampleBuffer
    ofType:(SCStreamOutputType)type
{
    if (type != SCStreamOutputTypeAudio) return;

    AudioBufferList audioBufferList;
    CMBlockBufferRef blockBuffer = NULL;

    OSStatus status = CMSampleBufferGetAudioBufferListWithRetainedBlockBuffer(
        sampleBuffer,
        NULL,
        &audioBufferList,
        sizeof(audioBufferList),
        kCFAllocatorDefault,
        kCFAllocatorDefault,
        kCMSampleBufferFlag_AudioBufferList_Assure16ByteAlignment,
        &blockBuffer
    );

    if (status != noErr || blockBuffer == NULL) return;

    for (UInt32 i = 0; i < audioBufferList.mNumberBuffers; i++) {
        const AudioBuffer *buf = &audioBufferList.mBuffers[i];
        if (buf->mData == NULL || buf->mDataByteSize == 0) continue;

        const float *floatSamples = (const float *)buf->mData;
        UInt32 numSamples = buf->mDataByteSize / sizeof(float);

        int16_t *pcm = (int16_t *)malloc(numSamples * sizeof(int16_t));
        if (!pcm) continue;

        for (UInt32 j = 0; j < numSamples; j++) {
            float s = floatSamples[j];
            if (s >  1.0f) s =  1.0f;
            if (s < -1.0f) s = -1.0f;
            pcm[j] = (int16_t)(s * 32767.0f);
        }

        sckDeliverSamples(pcm, (int)numSamples);
        free(pcm);
    }

    CFRelease(blockBuffer);
}

- (void)stream:(SCStream *)stream didStopWithError:(NSError *)error {
    NSLog(@"SCK stream stopped: %@", error.localizedDescription);
}

@end

// ── Capture handle ────────────────────────────────────────────────────────────

API_AVAILABLE(macos(13.0))
@interface SCKCapture : NSObject
@property (strong, nonatomic) SCStream    *stream;
@property (strong, nonatomic) SCKDelegate *delegate;
@end

@implementation SCKCapture
@end

// ── Public C API ──────────────────────────────────────────────────────────────

void *sck_start(int sampleRate, int channels) {
    if (@available(macOS 13.0, *)) {
        __block SCKCapture *result = nil;
        dispatch_semaphore_t sem = dispatch_semaphore_create(0);

        [SCShareableContent getShareableContentWithCompletionHandler:^(
            SCShareableContent *content, NSError *error)
        {
            if (error || content.displays.count == 0) {
                NSLog(@"SCK: getShareableContent failed: %@", error.localizedDescription);
                dispatch_semaphore_signal(sem);
                return;
            }

            SCStreamConfiguration *cfg = [[SCStreamConfiguration alloc] init];
            cfg.capturesAudio             = YES;
            cfg.excludesCurrentProcessAudio = NO;
            cfg.sampleRate                = sampleRate;
            cfg.channelCount              = channels;
            // Minimise video overhead — we only want audio.
            cfg.width                     = 2;
            cfg.height                    = 2;
            cfg.minimumFrameInterval      = CMTimeMake(1, 1); // 1 fps

            SCContentFilter *filter = [[SCContentFilter alloc]
                initWithDisplay:content.displays.firstObject
                excludingWindows:@[]];

            SCKDelegate *delegate = [[SCKDelegate alloc] init];
            SCStream *stream = [[SCStream alloc]
                initWithFilter:filter
                 configuration:cfg
                       delegate:delegate];

            NSError *addErr = nil;
            BOOL ok = [stream
                addStreamOutput:delegate
                           type:SCStreamOutputTypeAudio
             sampleHandlerQueue:dispatch_get_global_queue(QOS_CLASS_USER_INTERACTIVE, 0)
                          error:&addErr];
            if (!ok) {
                NSLog(@"SCK: addStreamOutput failed: %@", addErr.localizedDescription);
                dispatch_semaphore_signal(sem);
                return;
            }

            [stream startCaptureWithCompletionHandler:^(NSError *startErr) {
                if (startErr) {
                    NSLog(@"SCK: startCapture failed: %@", startErr.localizedDescription);
                } else {
                    SCKCapture *cap = [[SCKCapture alloc] init];
                    cap.stream   = stream;
                    cap.delegate = delegate;
                    result = cap;
                }
                dispatch_semaphore_signal(sem);
            }];
        }];

        // Wait up to 10 seconds for the async SCK setup.
        dispatch_time_t timeout = dispatch_time(DISPATCH_TIME_NOW, 10 * NSEC_PER_SEC);
        dispatch_semaphore_wait(sem, timeout);

        if (!result) return NULL;
        // Transfer ownership to C caller via CFBridgingRetain.
        return (void *)CFBridgingRetain(result);
    } else {
        NSLog(@"SCK: requires macOS 13.0 or later");
        return NULL;
    }
}

void sck_stop(void *handle) {
    if (!handle) return;
    if (@available(macOS 13.0, *)) {
        // Transfer ownership back to ARC.
        SCKCapture *cap = (SCKCapture *)CFBridgingRelease(handle);
        [cap.stream stopCaptureWithCompletionHandler:^(NSError *err) {
            if (err) NSLog(@"SCK: stopCapture error: %@", err.localizedDescription);
        }];
    }
}
