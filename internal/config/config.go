package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr             string
	ProcRoot         string
	CgroupRoot       string
	RootFSProbe      string
	Timezone         string
	DockerEnabled    bool
	DockerSocket     string
	SampleInterval   time.Duration
	Retention        time.Duration
	Demo             bool
	AuthUsername     string
	AuthPasswordFile string
}

func Load() (Config, error) {
	sampleInterval, err := durationEnv("MONITOR_SAMPLE_INTERVAL", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	retention, err := durationEnv("MONITOR_RETENTION", 24*time.Hour)
	if err != nil {
		return Config{}, err
	}

	dockerEnabled, err := boolEnv("MONITOR_DOCKER_ENABLED", true)
	if err != nil {
		return Config{}, err
	}

	demo, err := boolEnv("MONITOR_DEMO", false)
	if err != nil {
		return Config{}, err
	}

	if sampleInterval < time.Second {
		return Config{}, fmt.Errorf("MONITOR_SAMPLE_INTERVAL must be at least 1s")
	}
	if retention < sampleInterval {
		return Config{}, fmt.Errorf("MONITOR_RETENTION must be greater than the sample interval")
	}

	return Config{
		Addr:             stringEnv("MONITOR_ADDR", "127.0.0.1:8080"),
		ProcRoot:         stringEnv("MONITOR_PROC_ROOT", "/proc"),
		CgroupRoot:       stringEnv("MONITOR_CGROUP_ROOT", "/sys/fs/cgroup"),
		RootFSProbe:      stringEnv("MONITOR_ROOTFS_PROBE", "/"),
		Timezone:         stringEnv("MONITOR_TIMEZONE", "Asia/Shanghai"),
		DockerEnabled:    dockerEnabled,
		DockerSocket:     stringEnv("MONITOR_DOCKER_SOCKET", "/var/run/docker.sock"),
		SampleInterval:   sampleInterval,
		Retention:        retention,
		Demo:             demo,
		AuthUsername:     strings.TrimSpace(os.Getenv("MONITOR_AUTH_USERNAME")),
		AuthPasswordFile: strings.TrimSpace(os.Getenv("MONITOR_AUTH_PASSWORD_FILE")),
	}, nil
}

func stringEnv(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func durationEnv(name string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return parsed, nil
}

func boolEnv(name string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("%s: %w", name, err)
	}
	return parsed, nil
}
