package metrics

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestLinuxCollectorReadsProcFixtures(t *testing.T) {
	procRoot := t.TempDir()
	cgroupRoot := t.TempDir()

	writeFixture(t, procRoot, "sys/kernel/hostname", "fribench\n")
	writeFixture(t, procRoot, "sys/kernel/osrelease", "6.8.0-test\n")
	writeFixture(t, procRoot, "uptime", "543210.25 0.00\n")
	writeFixture(t, procRoot, "loadavg", "0.11 0.17 0.12 1/100 42\n")
	writeFixture(t, procRoot, "cpuinfo", "processor : 0\nprocessor : 1\nprocessor : 2\nprocessor : 3\n")
	writeFixture(t, procRoot, "stat", "cpu 100 0 100 800 0 0 0 0 0 0\n")
	writeFixture(t, procRoot, "pressure/cpu", "some avg10=0.08 avg60=0.20 avg300=0.20 total=1\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=0\n")
	writeFixture(t, procRoot, "pressure/memory", "some avg10=0.00 avg60=0.00 avg300=0.00 total=1\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=0\n")
	writeFixture(t, procRoot, "pressure/io", "some avg10=0.02 avg60=0.00 avg300=0.00 total=1\nfull avg10=0.00 avg60=0.00 avg300=0.00 total=0\n")
	writeFixture(t, procRoot, "meminfo", fmt.Sprintf(
		"MemTotal:       %d kB\nMemAvailable:   %d kB\nSwapTotal:      %d kB\nSwapFree:       %d kB\n",
		7_864_320,
		6_606_028,
		2_000_000,
		1_999_732,
	))
	writeFixture(t, procRoot, "1/net/route", "Iface Destination Gateway Flags RefCnt Use Metric Mask\neth0 00000000 00000000 0003 0 0 100 00000000\n")
	writeFixture(t, procRoot, "1/net/dev", "Inter-| Receive | Transmit\n face |bytes packets errs drop fifo frame compressed multicast|bytes packets errs drop fifo colls carrier compressed\n  lo: 100 0 0 0 0 0 0 0 100 0 0 0 0 0 0 0\neth0: 1000 0 0 0 0 0 0 0 2000 0 0 0 0 0 0 0\nveth0: 9000 0 0 0 0 0 0 0 9000 0 0 0 0 0 0 0\n")

	collector := NewLinuxCollector(CollectorOptions{
		ProcRoot:      procRoot,
		CgroupRoot:    cgroupRoot,
		RootFSProbe:   procRoot,
		Timezone:      "Asia/Shanghai",
		DockerEnabled: false,
	})

	first, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("first collection failed: %v", err)
	}
	if first.Hostname != "fribench" {
		t.Fatalf("hostname = %q", first.Hostname)
	}
	if first.CPU.LogicalCores != 4 {
		t.Fatalf("logical cores = %d", first.CPU.LogicalCores)
	}
	if first.Memory.TotalBytes != 7_864_320*1024 {
		t.Fatalf("memory total = %d", first.Memory.TotalBytes)
	}
	if first.Memory.SwapUsedBytes != 268*1024 {
		t.Fatalf("swap used = %d", first.Memory.SwapUsedBytes)
	}
	if first.CPU.Pressure.SomeAvg10 != 0.08 {
		t.Fatalf("CPU pressure = %f", first.CPU.Pressure.SomeAvg10)
	}

	writeFixture(t, procRoot, "stat", "cpu 150 0 150 900 0 0 0 0 0 0\n")
	writeFixture(t, procRoot, "1/net/dev", "Inter-| Receive | Transmit\n face |bytes packets errs drop fifo frame compressed multicast|bytes packets errs drop fifo colls carrier compressed\neth0: 2000 0 0 0 0 0 0 0 2600 0 0 0 0 0 0 0\nveth0: 12000 0 0 0 0 0 0 0 13000 0 0 0 0 0 0 0\n")
	second, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("second collection failed: %v", err)
	}
	if second.CPU.UsagePercent != 50 {
		t.Fatalf("CPU usage = %f, want 50", second.CPU.UsagePercent)
	}
	if second.Network.ReceivedBytes != 2000 || second.Network.SentBytes != 2600 {
		t.Fatalf("network totals = %+v", second.Network)
	}
}

func TestLinuxCollectorUsesHostnameOverride(t *testing.T) {
	procRoot := t.TempDir()
	cgroupRoot := t.TempDir()

	writeFixture(t, procRoot, "sys/kernel/hostname", "container-id\n")
	writeFixture(t, procRoot, "sys/kernel/osrelease", "6.8.0-test\n")
	writeFixture(t, procRoot, "uptime", "1 0\n")
	writeFixture(t, procRoot, "loadavg", "0 0 0 1/1 1\n")
	writeFixture(t, procRoot, "cpuinfo", "processor : 0\n")
	writeFixture(t, procRoot, "stat", "cpu 1 0 0 1 0 0 0 0 0 0\n")
	writeFixture(t, procRoot, "meminfo", "MemTotal: 1024 kB\nMemAvailable: 512 kB\n")

	collector := NewLinuxCollector(CollectorOptions{
		Hostname:      "fribench",
		ProcRoot:      procRoot,
		CgroupRoot:    cgroupRoot,
		RootFSProbe:   procRoot,
		DockerEnabled: false,
	})

	snapshot, err := collector.Collect(context.Background())
	if err != nil {
		t.Fatalf("collection failed: %v", err)
	}
	if snapshot.Hostname != "fribench" {
		t.Fatalf("hostname = %q, want override", snapshot.Hostname)
	}
}

func TestPercentHandlesZeroAndBounds(t *testing.T) {
	if value := percent(1, 0); value != 0 {
		t.Fatalf("zero total returned %f", value)
	}
	if value := percent(150, 100); value != 100 {
		t.Fatalf("over-capacity value returned %f", value)
	}
}

func writeFixture(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
