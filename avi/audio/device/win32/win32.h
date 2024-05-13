#ifdef __cplusplus
extern "C" {
#endif

#define INITGUID  // For PKEY_AudioEndpoint_GUID.

#include <windows.h>
#include <stdio.h>
#include <stdlib.h>

#include <mfidl.h>
#include <mfapi.h>
#include <mmdeviceapi.h>
#include <mmsystem.h>
#include <functiondiscoverykeys_devpkey.h>

typedef struct _AudioDevice {
    const char *name, 
               *id, 
               *guid,
               *format, 
               *association, 
               *jacksubtype, 
               *description;
} AudioDevice;

int NewAudioCaptureDevice(AudioDevice *device, int index);
int NewAudioCaptureRenderDevice(AudioDevice *device, int index);
int NewAudioRenderDevice(AudioDevice *device, int index);

#ifdef __cplusplus
}
#endif

#ifdef __cplusplus
template <class T> void release(T **ppT);
static int _pstring(IPropertyStore *pProps, PROPERTYKEY key, const char **out);
#endif