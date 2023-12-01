// wrapper.cc
#include <iostream>

#include <stdio.h>
#include <stdlib.h>
#include <windows.h>
#include <dshow.h>

#include "win32.h"

#pragma comment(lib, "strmiids")

// https://learn.microsoft.com/en-us/windows/win32/directshow/selecting-a-capture-device
HRESULT EnumerateDevices(REFGUID category, IEnumMoniker **ppEnum)
{
    VIDEOINFO *pvi;
    // Create the System Device Enumerator.
    ICreateDevEnum *pDevEnum;
    HRESULT hr = CoCreateInstance(CLSID_SystemDeviceEnum, NULL, CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&pDevEnum));
    if (FAILED(hr))
    {
        return hr;
    }

    // Create an enumerator for the category.
    hr = pDevEnum->CreateClassEnumerator(category, ppEnum, 0);
    if (hr == S_FALSE)
    {
        hr = VFW_E_NOT_FOUND;  // The category is empty. Treat as an error.
    }
    pDevEnum->Release();

    return hr;
}

const char * GetVideoCaptureDeviceName(int device, int *len)
{
    HRESULT hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
    if (FAILED(hr))
    {
        return "";
    }

    IEnumMoniker *pEnum;

    hr = EnumerateDevices(CLSID_VideoInputDeviceCategory, &pEnum);
    if (FAILED(hr))
    {
        return "";
    }

    IMoniker *pMoniker = NULL;

    for (int i = 0; pEnum->Next(1, &pMoniker, NULL) == S_OK; i++)
    {
        IPropertyBag *pPropBag;
        HRESULT hr = pMoniker->BindToStorage(0, 0, IID_PPV_ARGS(&pPropBag));
        if (FAILED(hr))
        {
            pMoniker->Release();
            continue;  
        } 

        VARIANT var;
        VariantInit(&var);

        hr = pPropBag->Read(L"FriendlyName", &var, 0);
        if (FAILED(hr))
        {
            VariantClear(&var); 
            pPropBag->Release();
            pMoniker->Release();
            continue;
        }

        if (i == device) 
        {
            const char *name = (const char *)var.bstrVal;
            *len = (int)SysStringByteLen(var.bstrVal);

            VariantClear(&var); 
            pPropBag->Release();
            pMoniker->Release();
            pEnum->Release();
            CoUninitialize();

            return name;
        }

        VariantClear(&var); 
        pPropBag->Release();
        pMoniker->Release();
    }

    pEnum->Release();
    CoUninitialize();

    return "";
}


const char * GetVideoCaptureDevicePath(int device, int *len)
{
    HRESULT hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
    if (FAILED(hr))
    {
        return "";
    }

    IEnumMoniker *pEnum;

    hr = EnumerateDevices(CLSID_VideoInputDeviceCategory, &pEnum);
    if (FAILED(hr))
    {
        return "";
    }

    IMoniker *pMoniker = NULL;

    for (int i = 0; pEnum->Next(1, &pMoniker, NULL) == S_OK; i++)
    {
        IPropertyBag *pPropBag;
        HRESULT hr = pMoniker->BindToStorage(0, 0, IID_PPV_ARGS(&pPropBag));
        if (FAILED(hr))
        {
            pMoniker->Release();
            continue;  
        } 

        VARIANT var;
        VariantInit(&var);

        hr = pPropBag->Read(L"DevicePath", &var, 0);
        if (FAILED(hr))
        {
            VariantClear(&var); 
            pPropBag->Release();
            pMoniker->Release();
            continue;
        }

        if (i == device) 
        {
            const char *name = (const char *)var.bstrVal;
            *len = (int)SysStringByteLen(var.bstrVal);

            VariantClear(&var); 
            pPropBag->Release();
            pMoniker->Release();
            pEnum->Release();
            CoUninitialize();

            return name;
        }

        VariantClear(&var); 
        pPropBag->Release();
        pMoniker->Release();
    }

    pEnum->Release();
    CoUninitialize();

    return "";
}

const char * GetVideoCaptureDeviceDescription(int device, int *len)
{
    HRESULT hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
    if (FAILED(hr))
    {
        return "";
    }

    IEnumMoniker *pEnum;

    hr = EnumerateDevices(CLSID_VideoInputDeviceCategory, &pEnum);
    if (FAILED(hr))
    {
        return "";
    }

    IMoniker *pMoniker = NULL;

    for (int i = 0; pEnum->Next(1, &pMoniker, NULL) == S_OK; i++)
    {
        IPropertyBag *pPropBag;
        HRESULT hr = pMoniker->BindToStorage(0, 0, IID_PPV_ARGS(&pPropBag));
        if (FAILED(hr))
        {
            pMoniker->Release();
            continue;  
        } 

        VARIANT var;
        VariantInit(&var);

        hr = pPropBag->Read(L"Description", &var, 0);
        if (FAILED(hr))
        {
            VariantClear(&var); 
            pPropBag->Release();
            pMoniker->Release();
            continue;
        }

        if (i == device) 
        {
            const char *name = (const char *)var.bstrVal;
            *len = (int)SysStringByteLen(var.bstrVal);

            VariantClear(&var); 
            pPropBag->Release();
            pMoniker->Release();
            pEnum->Release();
            CoUninitialize();

            return name;
        }

        VariantClear(&var); 
        pPropBag->Release();
        pMoniker->Release();
    }

    pEnum->Release();
    CoUninitialize();

    return "";
}


const char * GetVideoCaptureDeviceWaveInID(int device, int *len)
{
    HRESULT hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
    if (FAILED(hr))
    {
        return "";
    }

    IEnumMoniker *pEnum;

    hr = EnumerateDevices(CLSID_VideoInputDeviceCategory, &pEnum);
    if (FAILED(hr))
    {
        return "";
    }

    IMoniker *pMoniker = NULL;

    for (int i = 0; pEnum->Next(1, &pMoniker, NULL) == S_OK; i++)
    {
        IPropertyBag *pPropBag;
        HRESULT hr = pMoniker->BindToStorage(0, 0, IID_PPV_ARGS(&pPropBag));
        if (FAILED(hr))
        {
            pMoniker->Release();
            continue;  
        } 

        VARIANT var;
        VariantInit(&var);

        hr = pPropBag->Read(L"WaveInID", &var, 0);
        if (FAILED(hr))
        {
            VariantClear(&var); 
            pPropBag->Release();
            pMoniker->Release();
            continue;
        }

        if (i == device) 
        {
            const char *name = (const char *)var.bstrVal;
            *len = (int)SysStringByteLen(var.bstrVal);

            VariantClear(&var); 
            pPropBag->Release();
            pMoniker->Release();
            pEnum->Release();
            CoUninitialize();

            return name;
        }

        VariantClear(&var); 
        pPropBag->Release();
        pMoniker->Release();
    }

    pEnum->Release();
    CoUninitialize();

    return "";
}

