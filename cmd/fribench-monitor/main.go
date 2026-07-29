package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Max0314/fribench-monitor/internal/config"
	"github.com/Max0314/fribench-monitor/internal/metrics"
	"github.com/Max0314/fribench-monitor/internal/monitor"
	monitorserver "github.com/Max0314/fribench-monitor/internal/server"
	"github.com/Max0314/fribench-monitor/internal/webui"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		if err := runHealthcheck(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if err := run(); err != nil {
		slog.Error("fribench monitor stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	auth, err := loadAuth(cfg)
	if err != nil {
		return err
	}

	state := monitor.NewState(cfg.SampleInterval, cfg.Retention)

	var source metrics.Source
	if cfg.Demo {
		source = metrics.NewDemoCollector()
		for _, point := range metrics.DemoPoints(time.Now(), cfg.Retention, cfg.SampleInterval) {
			state.AddHistory(point)
		}
	} else {
		source = metrics.NewLinuxCollector(metrics.CollectorOptions{
			Hostname:      cfg.Hostname,
			ProcRoot:      cfg.ProcRoot,
			CgroupRoot:    cfg.CgroupRoot,
			RootFSProbe:   cfg.RootFSProbe,
			Timezone:      cfg.Timezone,
			DockerEnabled: cfg.DockerEnabled,
			DockerSocket:  cfg.DockerSocket,
		})
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := collectOnce(ctx, source, state); err != nil {
		slog.Warn("initial metrics are partial", "error", err)
	}
	if !cfg.Demo {
		timer := time.NewTimer(300 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			if err := collectOnce(ctx, source, state); err != nil {
				slog.Warn("initial CPU sample is partial", "error", err)
			}
		}
	}

	handler, err := monitorserver.NewHandler(state, webui.Assets, auth)
	if err != nil {
		return err
	}

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go sampleLoop(ctx, source, state, cfg.SampleInterval)

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info(
			"fribench monitor started",
			"addr",
			cfg.Addr,
			"sample_interval",
			cfg.SampleInterval,
			"retention",
			cfg.Retention,
			"demo",
			cfg.Demo,
		)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownContext)
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func sampleLoop(ctx context.Context, source metrics.Source, state *monitor.State, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := collectOnce(ctx, source, state); err != nil {
				slog.Warn("metrics are partial", "error", err)
			}
		}
	}
}

func collectOnce(ctx context.Context, source metrics.Source, state *monitor.State) error {
	snapshot, err := source.Collect(ctx)
	state.Update(snapshot)
	return err
}

func loadAuth(cfg config.Config) (monitorserver.Auth, error) {
	if cfg.AuthUsername == "" && cfg.AuthPasswordFile == "" {
		return monitorserver.Auth{}, nil
	}
	if cfg.AuthUsername == "" || cfg.AuthPasswordFile == "" {
		return monitorserver.Auth{}, errors.New("MONITOR_AUTH_USERNAME and MONITOR_AUTH_PASSWORD_FILE must be set together")
	}

	file, err := os.Open(cfg.AuthPasswordFile)
	if err != nil {
		return monitorserver.Auth{}, fmt.Errorf("open auth password file: %w", err)
	}
	defer file.Close()

	passwordBytes, err := io.ReadAll(io.LimitReader(file, 4096))
	if err != nil {
		return monitorserver.Auth{}, fmt.Errorf("read auth password file: %w", err)
	}
	password := strings.TrimSpace(string(passwordBytes))
	if password == "" {
		return monitorserver.Auth{}, errors.New("auth password file is empty")
	}

	return monitorserver.Auth{Username: cfg.AuthUsername, Password: password}, nil
}

func runHealthcheck() error {
	url := strings.TrimSpace(os.Getenv("MONITOR_HEALTH_URL"))
	if url == "" {
		url = "http://127.0.0.1:8080/healthz"
	}

	client := http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck returned %s", response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 16))
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(body)) != "ok" {
		return errors.New("healthcheck response is invalid")
	}
	return nil
}
