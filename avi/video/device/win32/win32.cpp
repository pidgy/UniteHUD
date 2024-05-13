#include <windows.h>
#include <iostream>
#include <sstream>
#include <codecvt> 
#include <locale>
#include <comutil.h>
#include <comdef.h>
#include <string>

#include <stdio.h>
#include <stdlib.h>
#include <dshow.h>

#include "win32.h"

#pragma comment(lib, "strmiids");

int GetVideoCaptureDevice(int index, VideoCaptureDevice *device) 
{
    VARIANT       var;
    IEnumMoniker *pEnum    = NULL;
    IMoniker     *pMoniker = NULL;
    IPropertyBag *pPropBag = NULL;

    HRESULT hr = CoInitializeEx(NULL, COINIT_MULTITHREADED);
    if (FAILED(hr))
    {
        goto release;
    }

    hr = _enumerateDevices(CLSID_VideoInputDeviceCategory, &pEnum);
    if (FAILED(hr))
    {
        goto release;
    }

    for (int i = index; i >= 0; i--) hr = pEnum->Next(1, &pMoniker, NULL);
    if (FAILED(hr)) 
    {
        goto release;
    }

    hr = pMoniker->BindToStorage(0, 0, IID_PPV_ARGS(&pPropBag));
    if (FAILED(hr))
    {
        goto release;
    } 

    VariantInit(&var);
    
    hr = pPropBag->Read(L"FriendlyName", &var, NULL);
    if (FAILED(hr))
    {
        goto release;
    }

    device->namelen = (int)SysStringByteLen(var.bstrVal);
    device->name = (char *)calloc(device->namelen, sizeof(char));
    memcpy(device->name, (const char *)var.bstrVal, device->namelen);
    
    VariantClear(&var); 

    hr = pPropBag->Read(L"DevicePath", &var, NULL);
    if (FAILED(hr))
    {
        goto release;
    }

    device->pathlen = (int)SysStringByteLen(var.bstrVal);
    device->path = (char *)calloc(device->pathlen, sizeof(char));
    memcpy(device->path, (const char *)var.bstrVal, device->pathlen);
    VariantClear(&var); 

    if (SUCCEEDED(pPropBag->Read(L"WaveInID", &var, NULL))) 
    {
        device->waveinid = var.lVal;
        VariantClear(&var); 
    }

release:
    release(&pPropBag);
    release(&pMoniker);
    release(&pEnum);
    CoUninitialize();

    return hr;
}