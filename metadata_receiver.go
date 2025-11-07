package opensearchqueryreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

// metricsReceiver implements receiver.Metrics
type metricsReceiver struct {
	config   *Config
	consumer consumer.Metrics
	scraper  *scraper
	cancel   context.CancelFunc
	logger   *zap.Logger
	settings receiver.Settings
}

// newMetricsReceiver creates a new metrics receiver
func newMetricsReceiver(
	cfg *Config,
	consumer consumer.Metrics,
	settings receiver.Settings,
) (*metricsReceiver, error) {
	// Create scraper
	scraper, err := newScraperInstance(cfg, settings)
	if err != nil {
		return nil, err
	}

	return &metricsReceiver{
		config:   cfg,
		consumer: consumer,
		scraper:  scraper,
		logger:   settings.Logger,
		settings: settings,
	}, nil
}

// Start begins the metrics collection process
func (r *metricsReceiver) Start(ctx context.Context, host component.Host) error {
	r.logger.Info("Starting OpenSearch Query Receiver")

	// Start scraper
	if err := r.scraper.start(ctx, host); err != nil {
		return err
	}

	// Create cancellable context
	ctx, cancel := context.WithCancel(ctx)
	r.cancel = cancel

	// Start collection loop
	go r.collectionLoop(ctx)

	return nil
}

// Shutdown stops the metrics collection
func (r *metricsReceiver) Shutdown(ctx context.Context) error {
	r.logger.Info("Shutting down OpenSearch Query Receiver")

	if r.cancel != nil {
		r.cancel()
	}

	return r.scraper.shutdown(ctx)
}

// collectionLoop periodically collects metrics
func (r *metricsReceiver) collectionLoop(ctx context.Context) {
	// Initial delay
	if r.config.InitialDelay > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(r.config.InitialDelay):
		}
	}

	// Collection ticker
	ticker := time.NewTicker(r.config.CollectionInterval)
	defer ticker.Stop()

	// Collect immediately on start
	r.collect(ctx)

	// Periodic collection
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.collect(ctx)
		}
	}
}

// collect executes queries and sends metrics
func (r *metricsReceiver) collect(ctx context.Context) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	// Scrape metrics
	metrics, err := r.scraper.scrape(timeoutCtx)
	if err != nil {
		r.logger.Error("Failed to scrape metrics", zap.Error(err))
		return
	}

	// Send metrics to consumer
	if err := r.consumer.ConsumeMetrics(timeoutCtx, metrics); err != nil {
		r.logger.Error("Failed to consume metrics", zap.Error(err))
	}
}
