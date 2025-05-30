package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/fortxun/idop/pkg/alerting"
	"github.com/fortxun/idop/pkg/analysis"
	"github.com/fortxun/idop/pkg/metrics"
	"github.com/fortxun/idop/pkg/pmm"
	"github.com/fortxun/idop/pkg/processor"
	"github.com/fortxun/idop/pkg/reporting"
	"github.com/fortxun/idop/pkg/types/config"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	configPath = flag.String("config", "config/config.yaml", "Path to configuration file")
	logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
)

func main() {
	flag.Parse()

	logger := initLogger(*logLevel)
	defer logger.Sync()

	logger.Info("Starting IDOP server")

	cfg, err := loadConfig(*configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration",
			zap.Error(err))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pmmClient, err := pmm.NewClient(&cfg.PMM, logger)
	if err != nil {
		logger.Fatal("Failed to create PMM client",
			zap.Error(err))
	}

	collector, err := metrics.NewCollector(pmmClient, cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create metrics collector",
			zap.Error(err))
	}

	detector := analysis.NewAnomalyDetector(&cfg.Analysis, logger)

	reporter := reporting.NewReportGenerator(logger)

	notifier := alerting.NewNotifier(&cfg.Alerting, logger)

	proc, err := processor.NewIDOPProcessor(
		cfg,
		collector,
		detector,
		reporter,
		notifier,
		logger,
	)
	if err != nil {
		logger.Fatal("Failed to create IDOP processor",
			zap.Error(err))
	}

	if err := proc.Start(ctx); err != nil {
		logger.Fatal("Failed to start IDOP processor",
			zap.Error(err))
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	logger.Info("Received shutdown signal")

	_, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := proc.Stop(); err != nil {
		logger.Error("Failed to stop IDOP processor",
			zap.Error(err))
	}

	logger.Info("IDOP server stopped")
}

func loadConfig(path string) (*config.Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.metrics_interval", "1m")
	v.SetDefault("analysis.enabled", true)
	v.SetDefault("analysis.anomaly_threshold", 2.0)
	v.SetDefault("analysis.sensitivity_level", "medium")
	v.SetDefault("analysis.min_data_points", 10)
	v.SetDefault("analysis.adaptive_baseline", true)
	v.SetDefault("analysis.adaptive_rate", 0.1)
	v.SetDefault("alerting.enabled", true)

	v.SetEnvPrefix("IDOP")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create config directory: %w", err)
			}

			if err := v.SafeWriteConfig(); err != nil {
				return nil, fmt.Errorf("failed to write default config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
	}

	var cfg config.Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func initLogger(level string) *zap.Logger {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapLevel)
	config.OutputPaths = []string{"stdout"}
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	logger, err := config.Build()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	return logger
}
