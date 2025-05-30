package analysis

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"

	"go.uber.org/zap"
)

type KnowledgeBase struct {
	entries     []KnowledgeEntry
	mutex       sync.RWMutex
	logger      *zap.Logger
	initialized bool
}

type KnowledgeEntry struct {
	ID string `json:"id"`
	Title string `json:"title"`
	Description string `json:"description"`
	MetricPatterns []string `json:"metric_patterns"`
	ValuePatterns []ValuePattern `json:"value_patterns"`
	CorrelationPatterns []CorrelationPattern `json:"correlation_patterns"`
	Recommendations []string `json:"recommendations"`
	Tags []string `json:"tags"`
	Severity string `json:"severity"`
	References []string `json:"references"`
}

type ValuePattern struct {
	Type string `json:"type"`
	Value float64 `json:"value"`
	Value2 float64 `json:"value2,omitempty"`
	Unit string `json:"unit,omitempty"`
}

type CorrelationPattern struct {
	SourceMetricPattern string `json:"source_metric_pattern"`
	TargetMetricPattern string `json:"target_metric_pattern"`
	CoefficientMin float64 `json:"coefficient_min,omitempty"`
	CoefficientMax float64 `json:"coefficient_max,omitempty"`
}

func NewKnowledgeBase(filePath string, logger *zap.Logger) (*KnowledgeBase, error) {
	kb := &KnowledgeBase{
		entries:     make([]KnowledgeEntry, 0),
		logger:      logger,
		initialized: false,
	}

	if filePath != "" {
		if err := kb.LoadFromFile(filePath); err != nil {
			return nil, err
		}
	} else {
		kb.InitializeDefault()
	}

	return kb, nil
}

func (kb *KnowledgeBase) LoadFromFile(filePath string) error {
	kb.mutex.Lock()
	defer kb.mutex.Unlock()

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read knowledge base file: %w", err)
	}

	var entries []KnowledgeEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to parse knowledge base file: %w", err)
	}

	kb.entries = entries
	kb.initialized = true
	kb.logger.Info("Loaded knowledge base",
		zap.String("file", filePath),
		zap.Int("entries", len(entries)))

	return nil
}

func (kb *KnowledgeBase) SaveToFile(filePath string) error {
	kb.mutex.RLock()
	defer kb.mutex.RUnlock()

	data, err := json.MarshalIndent(kb.entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal knowledge base: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write knowledge base file: %w", err)
	}

	kb.logger.Info("Saved knowledge base",
		zap.String("file", filePath),
		zap.Int("entries", len(kb.entries)))

	return nil
}

func (kb *KnowledgeBase) InitializeDefault() {
	kb.mutex.Lock()
	defer kb.mutex.Unlock()

	kb.entries = []KnowledgeEntry{
		{
			ID:             "KB-001",
			Title:          "High MySQL Query Latency",
			Description:    "Abnormally high query latency often indicates query performance issues, excessive load, or resource constraints.",
			MetricPatterns: []string{"mysql_query_latency.*"},
			ValuePatterns: []ValuePattern{
				{
					Type:  "above",
					Value: 100,
					Unit:  "ms",
				},
			},
			CorrelationPatterns: []CorrelationPattern{
				{
					SourceMetricPattern: "mysql_query_latency.*",
					TargetMetricPattern: "mysql_buffer_pool_usage",
					CoefficientMin:      0.7,
				},
			},
			Recommendations: []string{
				"Optimize slow queries by adding appropriate indexes",
				"Increase InnoDB buffer pool size if memory allows",
				"Check for table locks or long-running transactions",
			},
			Tags:       []string{"mysql", "performance", "latency"},
			Severity:   "medium",
			References: []string{"https://dev.mysql.com/doc/refman/8.0/en/optimization.html"},
		},
		{
			ID:             "KB-002",
			Title:          "MySQL Connection Saturation",
			Description:    "Too many connections can exhaust server resources and lead to connection failures.",
			MetricPatterns: []string{"mysql_connections"},
			ValuePatterns: []ValuePattern{
				{
					Type:  "above",
					Value: 80,
					Unit:  "percent",
				},
			},
			CorrelationPatterns: []CorrelationPattern{
				{
					SourceMetricPattern: "mysql_connections",
					TargetMetricPattern: "mysql_aborted_connections",
					CoefficientMin:      0.5,
				},
			},
			Recommendations: []string{
				"Increase max_connections parameter if resources allow",
				"Implement connection pooling in application",
				"Optimize queries to reduce connection duration",
				"Check for connection leaks in application code",
			},
			Tags:       []string{"mysql", "connections", "saturation"},
			Severity:   "high",
			References: []string{"https://dev.mysql.com/doc/refman/8.0/en/too-many-connections.html"},
		},
		{
			ID:             "KB-003",
			Title:          "High MySQL Error Rate",
			Description:    "Elevated error rates may indicate application bugs, configuration issues, or resource constraints.",
			MetricPatterns: []string{"mysql_error_rate"},
			ValuePatterns: []ValuePattern{
				{
					Type:  "above",
					Value: 1,
					Unit:  "percent",
				},
			},
			Recommendations: []string{
				"Check MySQL error log for specific error messages",
				"Verify application SQL syntax and parameter binding",
				"Ensure proper error handling in application code",
			},
			Tags:       []string{"mysql", "errors", "reliability"},
			Severity:   "high",
			References: []string{"https://dev.mysql.com/doc/refman/8.0/en/error-log.html"},
		},
		{
			ID:             "KB-004",
			Title:          "MySQL Disk I/O Saturation",
			Description:    "High disk I/O utilization can cause performance degradation across all database operations.",
			MetricPatterns: []string{"mysql_disk_io_utilization"},
			ValuePatterns: []ValuePattern{
				{
					Type:  "above",
					Value: 80,
					Unit:  "percent",
				},
			},
			CorrelationPatterns: []CorrelationPattern{
				{
					SourceMetricPattern: "mysql_disk_io_utilization",
					TargetMetricPattern: "mysql_query_latency.*",
					CoefficientMin:      0.6,
				},
			},
			Recommendations: []string{
				"Increase InnoDB buffer pool size to reduce disk reads",
				"Consider using faster storage (SSD/NVMe)",
				"Optimize queries to reduce disk operations",
				"Check for unnecessary full table scans",
			},
			Tags:       []string{"mysql", "disk", "io", "performance"},
			Severity:   "medium",
			References: []string{"https://dev.mysql.com/doc/refman/8.0/en/optimizing-innodb-diskio.html"},
		},
		{
			ID:             "KB-005",
			Title:          "MySQL Memory Pressure",
			Description:    "High memory usage can lead to swapping and severe performance degradation.",
			MetricPatterns: []string{"mysql_memory_usage"},
			ValuePatterns: []ValuePattern{
				{
					Type:  "above",
					Value: 85,
					Unit:  "percent",
				},
			},
			Recommendations: []string{
				"Reduce InnoDB buffer pool size if causing system memory pressure",
				"Optimize memory-intensive queries",
				"Consider adding more RAM to the server",
				"Check for memory leaks in custom functions or plugins",
			},
			Tags:       []string{"mysql", "memory", "performance"},
			Severity:   "high",
			References: []string{"https://dev.mysql.com/doc/refman/8.0/en/memory-use.html"},
		},
	}

	kb.initialized = true
	kb.logger.Info("Initialized default knowledge base",
		zap.Int("entries", len(kb.entries)))
}

func (kb *KnowledgeBase) FindMatch(anomaly Anomaly, correlations []MetricCorrelation) (bool, string, []string) {
	kb.mutex.RLock()
	defer kb.mutex.RUnlock()

	if !kb.initialized {
		kb.logger.Warn("Knowledge base not initialized")
		return false, "", nil
	}

	for _, entry := range kb.entries {
		metricMatched := false
		for _, pattern := range entry.MetricPatterns {
			matched, err := regexp.MatchString(pattern, anomaly.MetricName)
			if err != nil {
				kb.logger.Warn("Error matching metric pattern",
					zap.String("pattern", pattern),
					zap.Error(err))
				continue
			}
			if matched {
				metricMatched = true
				break
			}
		}
		if !metricMatched && len(entry.MetricPatterns) > 0 {
			continue
		}

		valueMatched := false
		for _, pattern := range entry.ValuePatterns {
			switch strings.ToLower(pattern.Type) {
			case "above":
				if anomaly.Value > pattern.Value {
					valueMatched = true
				}
			case "below":
				if anomaly.Value < pattern.Value {
					valueMatched = true
				}
			case "between":
				if anomaly.Value >= pattern.Value && anomaly.Value <= pattern.Value2 {
					valueMatched = true
				}
			case "equal":
				if anomaly.Value == pattern.Value {
					valueMatched = true
				}
			}
			if valueMatched {
				break
			}
		}
		if !valueMatched && len(entry.ValuePatterns) > 0 {
			continue
		}

		correlationMatched := false
		for _, pattern := range entry.CorrelationPatterns {
			for _, correlation := range correlations {
				sourceMatched, _ := regexp.MatchString(pattern.SourceMetricPattern, correlation.SourceMetric)
				targetMatched, _ := regexp.MatchString(pattern.TargetMetricPattern, correlation.TargetMetric)
				
				if sourceMatched && targetMatched {
					coefficientMatched := true
					if pattern.CoefficientMin != 0 && correlation.Coefficient < pattern.CoefficientMin {
						coefficientMatched = false
					}
					if pattern.CoefficientMax != 0 && correlation.Coefficient > pattern.CoefficientMax {
						coefficientMatched = false
					}
					
					if coefficientMatched {
						correlationMatched = true
						break
					}
				}
			}
			if correlationMatched {
				break
			}
		}
		if !correlationMatched && len(entry.CorrelationPatterns) > 0 {
			continue
		}

		kb.logger.Info("Found knowledge base match",
			zap.String("anomaly", anomaly.MetricName),
			zap.String("kb_entry", entry.ID),
			zap.String("title", entry.Title))
		
		return true, entry.ID, entry.Recommendations
	}

	return false, "", nil
}

func (kb *KnowledgeBase) AddEntry(entry KnowledgeEntry) {
	kb.mutex.Lock()
	defer kb.mutex.Unlock()

	kb.entries = append(kb.entries, entry)
	kb.logger.Info("Added knowledge base entry",
		zap.String("id", entry.ID),
		zap.String("title", entry.Title))
}

func (kb *KnowledgeBase) GetEntry(id string) (KnowledgeEntry, bool) {
	kb.mutex.RLock()
	defer kb.mutex.RUnlock()

	for _, entry := range kb.entries {
		if entry.ID == id {
			return entry, true
		}
	}

	return KnowledgeEntry{}, false
}

func (kb *KnowledgeBase) GetAllEntries() []KnowledgeEntry {
	kb.mutex.RLock()
	defer kb.mutex.RUnlock()

	entries := make([]KnowledgeEntry, len(kb.entries))
	copy(entries, kb.entries)
	
	return entries
}
