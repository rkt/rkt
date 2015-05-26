// +build linux freebsd netbsd openbsd

package device

/*
#define _BSD_SOURCE
#define _DEFAULT_SOURCE
#include <sys/types.h>

unsigned int
my_major(dev_t dev)
{
  return major(dev);
}

unsigned int
my_minor(dev_t dev)
{
  return minor(dev);
}

dev_t
my_makedev(unsigned int maj, unsigned int min)
{
       return makedev(maj, min);
}
*/
import "C"

func Major(rdev uint64) uint {
	major := C.my_major(C.dev_t(rdev))
	return uint(major)
}

func Minor(rdev uint64) uint {
	minor := C.my_minor(C.dev_t(rdev))
	return uint(minor)
}

func Makedev(maj uint, min uint) uint64 {
	dev := C.my_makedev(C.uint(maj), C.uint(min))
	return uint64(dev)
}
