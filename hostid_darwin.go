// +build darwin

package xxid

import "syscall"

func readPlatformMachineID() (string, error) {
	return syscall.Sysctl("kern.uuid")
}
