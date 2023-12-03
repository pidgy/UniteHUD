// windev.h
#ifdef __cplusplus
extern "C" {
#endif

const char * GetVideoCaptureDeviceName(int device, int *len);
const char * GetVideoCaptureDevicePath(int device, int *len);
const char * GetVideoCaptureDeviceDescription(int device, int *len);
const char * GetVideoCaptureDeviceWaveInID(int device, int *len);

#ifdef __cplusplus
}
#endif
