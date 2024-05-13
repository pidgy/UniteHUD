#ifdef __cplusplus
extern "C" {
#endif

#include <stdlib.h>

typedef struct _AudioDevice {
    const char *name;
    const char *id;
    const char *guid;
    const char *format;
    const char *association;
    const char *jacksubtype;
    const char *description;
} AudioDevice;

int NewAudioCaptureDevice(AudioDevice *device, int index);
int NewAudioCaptureRenderDevice(AudioDevice *device, int index);
int NewAudioRenderDevice(AudioDevice *device, int index);

#ifdef __cplusplus
}
#endif
