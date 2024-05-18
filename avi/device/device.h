// Exported.

#ifdef __cplusplus
extern "C"
{
#endif

#include <stdlib.h>
  typedef const int DeviceType;

  const DeviceType DeviceTypeAudioCapture = 0x01;
  const DeviceType DeviceTypeVideoCapture = 0x02;

  typedef struct _Device
  {
    char *Name, *Description, *Path;
    long WaveInID;
  } Device;

  void DeviceFree(Device* device);
  int DeviceInit(Device* device, int index, DeviceType t);
  char* DeviceName(int index, DeviceType t);
  char* DevicePath(int index, DeviceType t);

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

typedef struct __props
{
private:
  IPropertyBag* _bag = NULL;
  IEnumMoniker* _enum = NULL;
  IMoniker* _moniker = NULL;
  ICreateDevEnum* _dev = NULL;
  HRESULT _result = -1;

  template<class T>
  void _release(T** ppT...)
  {
    if (*ppT) {
      (*ppT)->Release();
      *ppT = NULL;
      return;
    }
  };

  void _failed()
  {
    if (SUCCEEDED(_result)) {
      _result = -1;
    }
  }

public:
  operator bool() const
  {
    return this && _bag && _enum && _moniker && SUCCEEDED(_result);
  }

  ~__props()
  {
    _release(&_moniker);
    _release(&_enum);
    _release(&_bag);
    _release(&_dev);

    CoUninitialize();
  }

  __props(int index, DeviceType type)
  {
    ULONG n;
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

    if (IS_ERROR(CoInitializeEx(NULL, COINIT_MULTITHREADED))) {
      goto failed;
    }

    _result = CoCreateInstance(
      CLSID_SystemDeviceEnum, NULL, CLSCTX_INPROC_SERVER, IID_PPV_ARGS(&_dev));
    if (FAILED(_result)) {
      goto failed;
    }

    _result = _dev->CreateClassEnumerator(guid, &_enum, 0);
    if (FAILED(_result)) {
      goto failed;
    }
    if (_result != S_OK) {
      goto failed;
    }

    for (int i = 0; i <= index; i++) {
      _result = _enum->Next(1, &_moniker, &n);
      if (FAILED(_result)) {
        goto failed;
      }
    }
    if (n != 1) {
      goto failed;
    }

    _result = _moniker->BindToStorage(0, 0, IID_PPV_ARGS(&_bag));
    if (FAILED(_result)) {
      goto failed;
    }

    return;
  failed:
    _failed();
  }

  HRESULT result() { return _result; }

  LONG int32(LPCOLESTR name)
  {
    LONG l;
    VARIANT v;

    VariantInit(&v);

    if (SUCCEEDED(_bag->Read(name, &v, NULL))) {
      l = v.lVal;
    }

    VariantClear(&v);

    return l;
  }

  LPSTR string(LPCOLESTR name)
  {
    LPSTR s = NULL;
    VARIANT v;

    VariantInit(&v);

    if (SUCCEEDED(_bag->Read(name, &v, NULL))) {
      UINT l = SysStringByteLen(v.bstrVal);
      s = (LPSTR)calloc(l, sizeof(char));
      snprintf(s, l, "%S", v.bstrVal);
    }

    VariantClear(&v);

    return s;
  }

} _props;

static int
_deviceInit(Device* device, int index, DeviceType type)
{
  _props props(index, type);
  if (!props) {
    return props.result();
  }

  device->Name = props.string(L"FriendlyName");
  device->Path = props.string(L"DevicePath");
  device->WaveInID = props.int32(L"WaveInID");
  device->Description = props.string(L"Description");

  return props.result();
}

static char*
_deviceProp(int index, DeviceType type, LPCOLESTR prop)
{
  _props props(index, type);
  if (!props) {
    return NULL;
  }

  return props.string(prop);
}

#endif
