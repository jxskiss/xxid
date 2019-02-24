// +build !darwin,!linux,!freebsd,!windows

package xxid

import "errors"

func readPlatformMachineID() (string, error) {
	return "", errors.New("not implemented")
}
