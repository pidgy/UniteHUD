// Exported.

#ifdef __cplusplus
extern "C" {
#endif

#include <stdlib.h>
typedef const int DeviceType;

const DeviceType DeviceTypeAudioCapture = 0x01;
const DeviceType DeviceTypeVideoCapture = 0x02;

typedef struct _Device {
  char *Name, *Description, *Path;
  long WaveInID;
} Device;

void DeviceFree(Device *device);
int DeviceInit(Device *device, int index, DeviceType t);
char *DeviceName(int index, DeviceType t);
char *DevicePath(int index, DeviceType t);

#ifdef __cplusplus
}
#endif

// Unexported.

#ifdef __cplusplus

#include <codecvt>
#include <comdef.h>
#include <comutil.h>
#include <dshow.h>
#include <iostream>
#include <locale>
#include <sstream>
#include <stdio.h>
#include <stdlib.h>
#include <string>
#include <windows.h>

#pragma comment(lib, "strmiids");

#define _free(x)                                                               \
  free(x);                                                                     \
  x = NULL

typedef struct _props {
private:
  IPropertyBag *propertyBag = NULL;
  IEnumMoniker *enumMoniker = NULL;
  IMoniker *moniker = NULL;
  ICreateDevEnum *devEnum = NULL;
  HRESULT hresult = -1;

  template <class T> void _release(T **ppT...) {
    if (*ppT) {
      (*ppT)->Release();
      *ppT = NULL;
      return;
    }
  };

  void _failed() {
    if (SUCCEEDED(hresult)) {
      hresult = -1;
    }
  }

public:
  operator bool() const {
    return this && propertyBag && enumMoniker && moniker && SUCCEEDED(hresult);
  }

  ~_props() {
    _release(&moniker);
    _release(&enumMoniker);
    _release(&propertyBag);
    _release(&devEnum);

    CoUninitialize();
  }

  _props(int index, DeviceType type) {
    ULONG num;
    GUID guid;

    switch (type) {
    case DeviceTypeAudioCapture:
      guid = CLSID_AudioInputDeviceCategory;
      break;
    case DeviceTypeVideoCapture:
      guid = CLSID_VideoInputDeviceCategory;
      break;
    default:
      goto failed;
    }

    CoInitializeEx(NULL, COINIT_SPEED_OVER_MEMORY);

    hresult = CoCreateInstance(CLSID_SystemDeviceEnum, NULL,
                               CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&devEnum));
    if (FAILED(hresult)) {
      goto failed;
    }

    hresult = devEnum->CreateClassEnumerator(guid, &enumMoniker, 0);
    if (hresult != S_OK) {
      goto failed;
    }

    for (int i = 0; i <= index; i++) {
      hresult = enumMoniker->Next(1, &moniker, &num);
      if (FAILED(hresult)) {
        goto failed;
      }
    }
    if (num != 1) {
      goto failed;
    }

    hresult = moniker->BindToStorage(0, 0, IID_PPV_ARGS(&propertyBag));
    if (FAILED(hresult)) {
      goto failed;
    }

    return;

  failed:
    _failed();
  }

  HRESULT result() { return hresult; }

  LONG int32(LPCOLESTR name) {
    LONG l;
    VARIANT v;

    VariantInit(&v);

    if (SUCCEEDED(propertyBag->Read(name, &v, NULL))) {
      l = v.lVal;
    }

    VariantClear(&v);

    return l;
  }

  LPSTR string(LPCOLESTR name) {
    LPSTR s = NULL;
    VARIANT v;

    VariantInit(&v);

    if (SUCCEEDED(propertyBag->Read(name, &v, NULL))) {
      UINT l = SysStringByteLen(v.bstrVal);
      s = (LPSTR)calloc(l, sizeof(char));
      snprintf(s, l, "%S", v.bstrVal);
    }

    VariantClear(&v);

    return s;
  }

} props;

static int cDeviceInit(Device *device, int index, DeviceType type) {
  props p(index, type);
  if (!p) {
    return -1;
  }

  device->Name = p.string(L"FriendlyName");
  device->Path = p.string(L"DevicePath");
  device->WaveInID = p.int32(L"WaveInID");
  device->Description = p.string(L"Description");

  return p.result();
}

static char *cDeviceProp(int index, DeviceType type, LPCOLESTR prop) {
  props props(index, type);
  if (!props) {
    return NULL;
  }

  return props.string(prop);
}

#endif
