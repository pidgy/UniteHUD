#include "device.h"

void
DeviceFree(Device* device)
{
  _free(device->Name);
  _free(device->Path);
  _free(device->Description);
}

int
DeviceInit(Device* device, int index, DeviceType t)
{
  return newDevice(device, index, _toGUID(t));
}

char*
DeviceName(int index, DeviceType t)
{
  _props props(index, _toGUID(t));
  if (!props) {
    return NULL;
  }

  return props.string(L"FriendlyName");
}