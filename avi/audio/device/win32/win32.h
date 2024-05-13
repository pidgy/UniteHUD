#ifdef __cplusplus
extern "C" 
{
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

typedef struct _AudioDevice 
{
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

const IID            _mmdeID     = __uuidof(MMDeviceEnumerator);
const IID            _immdeID    = __uuidof(IMMDeviceEnumerator);

int newAudioDevice(AudioDevice *device, int index, EDataFlow eDataFlow);

template <class T> void release(T **ppT)
{
    if (!*ppT) 
    {
        return;
    }

    (*ppT)->Release();
    *ppT = NULL;
};

static int _pstring(IPropertyStore *pProps, PROPERTYKEY key, const char **out) 
{
    HRESULT hr;
    PROPVARIANT var;

    PropVariantInit(&var);
    
    hr = pProps->GetValue(key, &var);
    if (FAILED(hr))
    {
        goto release;
    }

    *out = (const char *)calloc(sizeof(char), 1024);
    snprintf((char *)*out, 1024, "%S", var.bstrVal);

    PropVariantClear(&var); 

release:

    return hr;
}

#endif