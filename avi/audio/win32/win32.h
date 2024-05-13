#ifdef __cplusplus
extern "C" {
#endif

#include <stdlib.h>

typedef struct _AudioCaptureDevice {
    const char *name;
    const char *id;
    const char *guid;
    const char *format;
    const char *association;
    const char *jacksubtype;
    const char *description;
} AudioCaptureDevice;

int NewAudioCaptureDevice(AudioCaptureDevice *device, int index);

#ifdef __cplusplus
}
#endif
