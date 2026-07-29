package metrics

func calculateDiskUsage(blocks, freeBlocks, availableBlocks, blockSize uint64) (
	total uint64,
	used uint64,
	available uint64,
	usagePercent float64,
) {
	total = blocks * blockSize
	if freeBlocks > blocks {
		freeBlocks = blocks
	}
	used = (blocks - freeBlocks) * blockSize
	available = availableBlocks * blockSize
	usagePercent = percent(used, used+available)
	return
}
