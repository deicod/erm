package metrics

import "time"

// Collector captures lightweight instrumentation for ORM/GraphQL operations.
type Collector interface {
	RecordDataloaderBatch(name string, size int, duration time.Duration)
}

// NoopCollector discards all metrics.
type NoopCollector struct{}

// RecordDataloaderBatch implements Collector.
func (NoopCollector) RecordDataloaderBatch(string, int, time.Duration) {}

// MultiCollector fan-outs events to multiple collectors.
type MultiCollector []Collector

// RecordDataloaderBatch implements Collector.
func (mc MultiCollector) RecordDataloaderBatch(name string, size int, duration time.Duration) {
	for _, c := range mc {
		if c == nil {
			continue
		}
		c.RecordDataloaderBatch(name, size, duration)
	}
}

// WithCollector returns a collector that fans out to all provided collectors.
func WithCollector(primary Collector, others ...Collector) Collector {
	collectors := make([]Collector, 0, 1+len(others))
	if primary != nil {
		collectors = append(collectors, primary)
	}
	for _, c := range others {
		if c != nil {
			collectors = append(collectors, c)
		}
	}
	switch len(collectors) {
	case 0:
		return NoopCollector{}
	case 1:
		return collectors[0]
	default:
		return MultiCollector(collectors)
	}
}
