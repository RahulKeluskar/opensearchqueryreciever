package opensearchqueryreceiver

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
	"go.uber.org/zap"
)

// scraper queries OpenSearch and converts results to OTel metrics
type scraper struct {
	config   *Config
	client   *OpenSearchClient
	logger   *zap.Logger
	settings receiver.Settings
}

// newScraperInstance creates a new scraper instance
func newScraperInstance(cfg *Config, settings receiver.Settings) (*scraper, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Create OpenSearch client
	client, err := NewOpenSearchClient(cfg, settings.Logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	return &scraper{
		config:   cfg,
		client:   client,
		logger:   settings.Logger,
		settings: settings,
	}, nil
}

// start is called when the receiver starts
func (s *scraper) start(ctx context.Context, host component.Host) error {
	s.logger.Info("Starting OpenSearch Query Receiver",
		zap.String("mode", s.config.Mode),
		zap.String("endpoint", s.config.GetEndpoint()),
		zap.String("index_pattern", s.config.IndexPattern),
		zap.Int("num_queries", len(s.config.Queries)),
	)

	// Ping OpenSearch to verify connectivity
	if err := s.client.Ping(ctx); err != nil {
		s.logger.Warn("Failed to ping OpenSearch on startup", zap.Error(err))
		// Don't fail startup, as OpenSearch might be temporarily unavailable
	}

	return nil
}

// shutdown is called when the receiver stops
func (s *scraper) shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down OpenSearch Query Receiver")
	return nil
}

// scrape executes all configured queries and returns metrics
func (s *scraper) scrape(ctx context.Context) (pmetric.Metrics, error) {
	now := pcommon.NewTimestampFromTime(time.Now())
	metrics := pmetric.NewMetrics()

	// Create resource metrics
	rm := metrics.ResourceMetrics().AppendEmpty()
	resourceAttrs := rm.Resource().Attributes()
	resourceAttrs.PutStr("receiver", typeStr.String())
	resourceAttrs.PutStr("opensearch.endpoint", s.config.GetEndpoint())
	resourceAttrs.PutStr("opensearch.index_pattern", s.config.IndexPattern)
	resourceAttrs.PutStr("opensearch.mode", s.config.Mode)

	// Create scope metrics
	sm := rm.ScopeMetrics().AppendEmpty()
	sm.Scope().SetName(typeStr.String())

	// Execute each configured query
	for _, queryConfig := range s.config.Queries {
		if err := s.executeAndRecordQuery(ctx, queryConfig, sm, now); err != nil {
			s.logger.Error("Failed to execute query",
				zap.String("query_name", queryConfig.Name),
				zap.Error(err),
			)
			// Continue with other queries even if one fails
			continue
		}
	}

	return metrics, nil
}

// executeAndRecordQuery executes a single query and records the results as metrics
func (s *scraper) executeAndRecordQuery(
	ctx context.Context,
	queryConfig QueryConfig,
	sm pmetric.ScopeMetrics,
	timestamp pcommon.Timestamp,
) error {
	// Execute query
	resp, err := s.client.ExecuteQuery(ctx, queryConfig)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}

	// Determine metric name
	metricName := queryConfig.MetricName
	if metricName == "" {
		metricName = fmt.Sprintf("opensearch.query.%s", queryConfig.Name)
	}

	// Record hit count as a gauge metric
	s.recordGaugeMetric(
		sm,
		fmt.Sprintf("%s.count", metricName),
		"Number of documents matching the query",
		"hits",
		float64(resp.Hits.Total.Value),
		queryConfig,
		timestamp,
	)

	// Record query execution time
	s.recordGaugeMetric(
		sm,
		fmt.Sprintf("%s.took_ms", metricName),
		"Query execution time in milliseconds",
		"ms",
		float64(resp.Took),
		queryConfig,
		timestamp,
	)

	// Record shard statistics
	s.recordGaugeMetric(
		sm,
		fmt.Sprintf("%s.shards.total", metricName),
		"Total number of shards queried",
		"shards",
		float64(resp.Shards.Total),
		queryConfig,
		timestamp,
	)

	s.recordGaugeMetric(
		sm,
		fmt.Sprintf("%s.shards.successful", metricName),
		"Number of successful shards",
		"shards",
		float64(resp.Shards.Successful),
		queryConfig,
		timestamp,
	)

	s.recordGaugeMetric(
		sm,
		fmt.Sprintf("%s.shards.failed", metricName),
		"Number of failed shards",
		"shards",
		float64(resp.Shards.Failed),
		queryConfig,
		timestamp,
	)

	// Process aggregations if present
	if resp.Aggregations != nil && len(resp.Aggregations) > 0 {
		s.processAggregations(sm, metricName, resp.Aggregations, queryConfig, timestamp)
	}

	s.logger.Debug("Query metrics recorded",
		zap.String("query_name", queryConfig.Name),
		zap.Int64("hit_count", resp.Hits.Total.Value),
		zap.Int("took_ms", resp.Took),
	)

	return nil
}

// recordGaugeMetric creates and records a gauge metric
func (s *scraper) recordGaugeMetric(
	sm pmetric.ScopeMetrics,
	name string,
	description string,
	unit string,
	value float64,
	queryConfig QueryConfig,
	timestamp pcommon.Timestamp,
) {
	metric := sm.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetDescription(description)
	metric.SetUnit(unit)

	gauge := metric.SetEmptyGauge()
	dp := gauge.DataPoints().AppendEmpty()
	dp.SetTimestamp(timestamp)
	dp.SetDoubleValue(value)

	// Add query-specific attributes
	attrs := dp.Attributes()
	attrs.PutStr("query.name", queryConfig.Name)
	if queryConfig.Description != "" {
		attrs.PutStr("query.description", queryConfig.Description)
	}

	// Add custom labels
	for k, v := range queryConfig.Labels {
		attrs.PutStr(k, v)
	}
}

// processAggregations processes OpenSearch aggregation results
func (s *scraper) processAggregations(
	sm pmetric.ScopeMetrics,
	baseMetricName string,
	aggregations map[string]interface{},
	queryConfig QueryConfig,
	timestamp pcommon.Timestamp,
) {
	for aggName, aggValue := range aggregations {
		s.processAggregation(sm, baseMetricName, aggName, aggValue, queryConfig, timestamp)
	}
}

// processAggregation processes a single aggregation
func (s *scraper) processAggregation(
	sm pmetric.ScopeMetrics,
	baseMetricName string,
	aggName string,
	aggValue interface{},
	queryConfig QueryConfig,
	timestamp pcommon.Timestamp,
) {
	aggMap, ok := aggValue.(map[string]interface{})
	if !ok {
		return
	}

	// Handle value aggregations (avg, sum, min, max, etc.)
	if value, ok := aggMap["value"].(float64); ok {
		metricName := fmt.Sprintf("%s.agg.%s", baseMetricName, aggName)
		s.recordGaugeMetric(
			sm,
			metricName,
			fmt.Sprintf("Aggregation result for %s", aggName),
			"",
			value,
			queryConfig,
			timestamp,
		)
	}

	// Handle doc_count (for terms aggregations, etc.)
	if docCount, ok := aggMap["doc_count"].(float64); ok {
		metricName := fmt.Sprintf("%s.agg.%s.doc_count", baseMetricName, aggName)
		s.recordGaugeMetric(
			sm,
			metricName,
			fmt.Sprintf("Document count for %s", aggName),
			"documents",
			docCount,
			queryConfig,
			timestamp,
		)
	}

	// Handle buckets (for terms, date_histogram aggregations)
	if buckets, ok := aggMap["buckets"].([]interface{}); ok {
		s.processBuckets(sm, baseMetricName, aggName, buckets, queryConfig, timestamp)
	}
}

// processBuckets processes aggregation buckets
func (s *scraper) processBuckets(
	sm pmetric.ScopeMetrics,
	baseMetricName string,
	aggName string,
	buckets []interface{},
	queryConfig QueryConfig,
	timestamp pcommon.Timestamp,
) {
	for i, bucket := range buckets {
		bucketMap, ok := bucket.(map[string]interface{})
		if !ok {
			continue
		}

		// Get bucket key
		key := ""
		if keyVal, ok := bucketMap["key"].(string); ok {
			key = keyVal
		} else if keyVal, ok := bucketMap["key"].(float64); ok {
			key = fmt.Sprintf("%.0f", keyVal)
		} else {
			key = fmt.Sprintf("bucket_%d", i)
		}

		// Record bucket doc_count
		if docCount, ok := bucketMap["doc_count"].(float64); ok {
			metricName := fmt.Sprintf("%s.agg.%s.bucket.doc_count", baseMetricName, aggName)
			metric := sm.Metrics().AppendEmpty()
			metric.SetName(metricName)
			metric.SetDescription(fmt.Sprintf("Bucket document count for %s", aggName))
			metric.SetUnit("documents")

			gauge := metric.SetEmptyGauge()
			dp := gauge.DataPoints().AppendEmpty()
			dp.SetTimestamp(timestamp)
			dp.SetDoubleValue(docCount)

			// Add attributes
			attrs := dp.Attributes()
			attrs.PutStr("query.name", queryConfig.Name)
			attrs.PutStr("bucket.key", key)
			attrs.PutStr("aggregation", aggName)

			// Add custom labels
			for k, v := range queryConfig.Labels {
				attrs.PutStr(k, v)
			}
		}

		// Process nested aggregations in bucket
		for nestedAggName, nestedAggValue := range bucketMap {
			if nestedAggName != "key" && nestedAggName != "doc_count" && nestedAggName != "key_as_string" {
				nestedBaseMetric := fmt.Sprintf("%s.%s", baseMetricName, key)
				s.processAggregation(sm, nestedBaseMetric, nestedAggName, nestedAggValue, queryConfig, timestamp)
			}
		}
	}
}
