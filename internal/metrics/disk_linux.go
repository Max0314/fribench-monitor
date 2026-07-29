//go:build linux

package metrics

import "syscall"

func diskUsage(path string) (Disk, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return Disk{}, err
	}

	total, used, available, usagePercent := calculateDiskUsage(
		stats.Blocks,
		stats.Bfree,
		stats.Bavail,
		uint64(stats.Bsize),
	)
	inodesTotal := stats.Files
	inodesFree := stats.Ffree
	inodesUsed := inodesTotal - inodesFree

	return Disk{
		TotalBytes:    total,
		UsedBytes:     used,
		FreeBytes:     available,
		UsagePercent:  usagePercent,
		InodesTotal:   inodesTotal,
		InodesUsed:    inodesUsed,
		InodesPercent: percent(inodesUsed, inodesTotal),
	}, nil
}
