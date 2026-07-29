//go:build linux

package metrics

import "syscall"

func diskUsage(path string) (Disk, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return Disk{}, err
	}

	total := stats.Blocks * uint64(stats.Bsize)
	free := stats.Bavail * uint64(stats.Bsize)
	used := total - free
	inodesTotal := stats.Files
	inodesFree := stats.Ffree
	inodesUsed := inodesTotal - inodesFree

	return Disk{
		TotalBytes:    total,
		UsedBytes:     used,
		FreeBytes:     free,
		UsagePercent:  percent(used, total),
		InodesTotal:   inodesTotal,
		InodesUsed:    inodesUsed,
		InodesPercent: percent(inodesUsed, inodesTotal),
	}, nil
}
