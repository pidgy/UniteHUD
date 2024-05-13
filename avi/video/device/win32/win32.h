#ifdef __cplusplus
extern "C" 
{
#endif

typedef struct _VideoCaptureDevice 
{
    char *name;
    char *path;
    char *description;

    int namelen;
    int pathlen;
    int descriptionlen;

    long waveinid;
} VideoCaptureDevice;

int NewVideoCaptureDevice(int index, VideoCaptureDevice *device);

#ifdef __cplusplus
}
#endif

#ifdef __cplusplus

template <class T> void release(T **ppT)
{
    if (!*ppT) 
    {
        return;
    }

    (*ppT)->Release();
    *ppT = NULL;
};

// https://learn.microsoft.com/en-us/windows/win32/directshow/selecting-a-capture-device
HRESULT _enumerateDevices(REFGUID category, IEnumMoniker **ppEnum)
{
    VIDEOINFO      *pvi      = NULL;
    ICreateDevEnum *pDevEnum = NULL;
    HRESULT         hr;

    hr = CoCreateInstance(CLSID_SystemDeviceEnum, NULL, CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&pDevEnum));
    if (FAILED(hr))
    {
        goto release;
    }

    hr = pDevEnum->CreateClassEnumerator(category, ppEnum, 0);
    if (FAILED(hr))
    {
        goto release;
    }

release:
    release(&pDevEnum);

    return hr;
}

#endif
