#include "win32.h"

#include <windows.h>
#include <stdio.h>
#include <stdlib.h>

#include <iostream>
#include <sstream>
#include <codecvt> 
#include <locale>
#include <comutil.h>
#include <comdef.h>
#include <string>

#define INITGUID  // For PKEY_AudioEndpoint_GUID
#include <mfidl.h>
#include <mfapi.h>
#include <mmdeviceapi.h>
#include <mmsystem.h>
#include <functiondiscoverykeys_devpkey.h>
#include <uuids.h>

#pragma comment(lib, "strmiids");

template <class T> void release(T **ppT);

static int getStringProp(IPropertyStore *pProps, PROPERTYKEY key, const char **out);

int NewAudioCaptureDevice(AudioCaptureDevice *device, int index) {
    IMMDeviceEnumerator *pEnum       = NULL; // Audio device enumerator.
    IMMDeviceCollection *pDevices    = NULL; // Audio device collection.
    IMMDevice           *pDevice     = NULL; // An audio device.
    IMFAttributes       *pAttributes = NULL; // Attribute store.
    IMFMediaSink        *pSink       = NULL; // Streaming audio renderer (SAR)
    IPropertyStore      *pProps      = NULL;
    LPWSTR               wstrID      = NULL; // Device ID.

    HRESULT hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
    if (FAILED(hr))
    {
        goto release;
    }

    // Create the device enumerator.
    hr = CoCreateInstance(__uuidof(MMDeviceEnumerator), NULL, CLSCTX_ALL, __uuidof(IMMDeviceEnumerator), (void**)&pEnum);
    if (FAILED(hr))
    {
        goto release;
    }

    hr = pEnum->EnumAudioEndpoints(eRender, DEVICE_STATE_ACTIVE, &pDevices);
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

    hr = getStringProp(pProps, PKEY_DeviceInterface_FriendlyName, &device->name);
    if (FAILED(hr)) {
        goto release;
    }

    hr = getStringProp(pProps, PKEY_AudioEndpoint_GUID, &device->guid);
    if (FAILED(hr)) {
        goto release;
    }

    hr = getStringProp(pProps, PKEY_AudioEndpoint_Association, &device->association);
    if (FAILED(hr)) {
        goto release;
    }

    hr = getStringProp(pProps, PKEY_AudioEndpoint_JackSubType, &device->jacksubtype);
    if (FAILED(hr)) {
        goto release;
    }

    hr = getStringProp(pProps, PKEY_Device_DeviceDesc, &device->description);
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

static int getStringProp(IPropertyStore *pProps, PROPERTYKEY key, const char **out) {
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
    if (*ppT)
    {
        (*ppT)->Release();
        *ppT = NULL;
    }
}
