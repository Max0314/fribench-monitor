package metrics

import "testing"

func TestCalculateDiskUsageMatchesDFSemantics(t *testing.T) {
	total, used, available, usagePercent := calculateDiskUsage(100, 30, 25, 1024)

	if total != 100*1024 || used != 70*1024 || available != 25*1024 {
		t.Fatalf("disk values = %d, %d, %d", total, used, available)
	}
	expected := float64(70) / float64(95) * 100
	if usagePercent != expected {
		t.Fatalf("usage percent = %f, want %f", usagePercent, expected)
	}
}
