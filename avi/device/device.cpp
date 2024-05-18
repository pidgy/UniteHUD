#include "device.h"

void
DeviceFree(Device* device)
{
  _free(device->Name);
  _free(device->Path);
  _free(device->Description);
}

int
DeviceInit(Device* device, int index, DeviceType type)
{
  return _deviceInit(device, index, type);
}

char*
DeviceName(int index, DeviceType type)
{
  return _deviceProp(index, type, L"FriendlyName");
}

char*
DevicePath(int index, DeviceType type)
{
  return _deviceProp(index, type, L"DevicePath");
}