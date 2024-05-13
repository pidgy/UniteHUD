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
template <class T> void release(T **ppT)
{
    if (*ppT) {
        (*ppT)->Release();
        *ppT = NULL;
    }
};

static int _pstring(IPropertyStore *pProps, PROPERTYKEY key, const char **out) {
    HRESULT hr = S_OK;
    
    PROPVARIANT var;

    PropVariantInit(&var);
    
    hr = pProps->GetValue(key, &var);
    if (FAILED(hr)) {
        goto exit;
    }

    *out = (const char *)calloc(sizeof(char), 1024);
    hr = snprintf((char *)*out, 1024, "%S", var.bstrVal);
    if (FAILED(hr)) {
        goto exit;
    }

exit:
    PropVariantClear(&var); 

    return hr;
}
#endif