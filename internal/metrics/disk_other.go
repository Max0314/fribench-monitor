//go:build !linux

package metrics

import "errors"

func diskUsage(string) (Disk, error) {
	return Disk{}, errors.New("host filesystem metrics require Linux")
}
