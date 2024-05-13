#include "win32.h"

int newAudioDevice(AudioDevice *device, int index, EDataFlow eDataFlow);

int NewAudioCaptureDevice(AudioDevice *device, int index) {
    return newAudioDevice(device, index, EDataFlow::eCapture);
}

int NewAudioCaptureRenderDevice(AudioDevice *device, int index) {
    return newAudioDevice(device, index, EDataFlow::eAll);
}

int NewAudioRenderDevice(AudioDevice *device, int index) {
    return newAudioDevice(device, index, EDataFlow::eRender);
}

int newAudioDevice(AudioDevice *device, int index, EDataFlow eDataFlow) {
    IMMDeviceEnumerator *pEnum       = NULL; // Audio device enumerator.
    IMMDeviceCollection *pDevices    = NULL; // Audio device collection.
    IMMDevice           *pDevice     = NULL; // An audio device.
    IMFAttributes       *pAttributes = NULL; // Attribute store.
    IMFMediaSink        *pSink       = NULL; // Streaming audio renderer (SAR)
    IPropertyStore      *pProps      = NULL;
    LPWSTR               wstrID      = NULL; // Device ID.
    const IID            _mmdeID      = __uuidof(MMDeviceEnumerator);
    const IID            _immdeID     = __uuidof(IMMDeviceEnumerator);

    HRESULT hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
    if (FAILED(hr))
    {
        goto release;
    }

    hr = CoCreateInstance(_mmdeID, NULL, CLSCTX_ALL, _immdeID, (void**)&pEnum);
    if (FAILED(hr))
    {
        goto release;
    }

    hr = pEnum->EnumAudioEndpoints(eDataFlow, DEVICE_STATE_ACTIVE, &pDevices);
    if (FAILED(hr))
    {
        goto release;
    }

    hr = pDevices->Item(index, &pDevice);
    if (FAILED(hr))
    {
        goto release;
    }

    hr = pDevice->GetId(&wstrID);
    if (FAILED(hr))
    {
        goto release;
    }

    hr = MFCreateAttributes(&pAttributes, 2);
    if (FAILED(hr))
    {
        goto release;
    }

    device->id = (const char *)calloc(sizeof(char), 1024);
    hr = snprintf((char *)device->id, 1024, "%S", wstrID);
    if (FAILED(hr)) {
        goto release;
    }

    hr = pDevice->OpenPropertyStore(STGM_READ, &pProps);
    if (FAILED(hr)) {
        goto release;
    }

    hr = _pstring(pProps, PKEY_DeviceInterface_FriendlyName, &device->name);
    if (FAILED(hr)) {
        goto release;
    }

    hr =  _pstring(pProps, PKEY_AudioEndpoint_GUID, &device->guid);
    if (FAILED(hr)) {
        goto release;
    }

    hr =  _pstring(pProps, PKEY_AudioEndpoint_Association, &device->association);
    if (FAILED(hr)) {
        goto release;
    }

    hr =  _pstring(pProps, PKEY_AudioEndpoint_JackSubType, &device->jacksubtype);
    if (FAILED(hr)) {
        goto release;
    }

    hr =  _pstring(pProps, PKEY_Device_DeviceDesc, &device->description);
    if (FAILED(hr)) {
        goto release;
    }

    // Create the audio renderer.
    // hr = pAttributes->SetString(MF_AUDIO_RENDERER_ATTRIBUTE_ENDPOINT_ID, wstrID);
    // if (FAILED(hr))
    // {
    //     goto release;
    // }

    // hr = MFCreateAudioRenderer(pAttributes, &pSink);
    // if (FAILED(hr)) {
    //     goto release;
    // }

    release:
        release(&pEnum);
        release(&pDevices);
        release(&pDevice);
        release(&pAttributes);
        release(&pSink);
        release(&pProps);

    return hr;
}

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

template <class T> void release(T **ppT)
{
    if (*ppT) {
        (*ppT)->Release();
        *ppT = NULL;
    }
};