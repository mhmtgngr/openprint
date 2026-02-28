// Package testutil provides test metrics collection for testing.
package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// MetricsCollector collects test execution metrics.
type MetricsCollector struct {
	mu                sync.Mutex
	startTime         time.Time
	endTime           time.Time
	testCount         int
	passedTests       int
	failedTests       int
	skippedTests      int
	testDurations     map[string]time.Duration
	customMetrics     map[string]interface{}
	timers            map[string]time.Time
	counters          map[string]int64
	histograms        map[string][]float64
	labels            map[string]string
}

// NewMetricsCollector creates a new metrics collector.
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime:     time.Now(),
		testDurations: make(map[string]time.Duration),
		customMetrics: make(map[string]interface{}),
		timers:        make(map[string]time.Time),
		counters:      make(map[string]int64),
		histograms:    make(map[string][]float64),
		labels:        make(map[string]string),
	}
}

// StartTimer starts a named timer.
func (mc *MetricsCollector) StartTimer(name string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.timers[name] = time.Now()
}

// StopTimer stops a named timer and records the duration.
func (mc *MetricsCollector) StopTimer(name string) time.Duration {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if startTime, exists := mc.timers[name]; exists {
		duration := time.Since(startTime)
		delete(mc.timers, name)
		return duration
	}

	return 0
}

// RecordTimer records a duration for a named timer.
func (mc *MetricsCollector) RecordTimer(name string, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.histograms[name] = append(mc.histograms[name], float64(duration.Milliseconds()))
}

// RecordTest records a test result.
func (mc *MetricsCollector) RecordTest(name string, passed bool, duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.testCount++
	if passed {
		mc.passedTests++
	} else {
		mc.failedTests++
	}
	mc.testDurations[name] = duration
}

// RecordSkip records a skipped test.
func (mc *MetricsCollector) RecordSkip() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.skippedTests++
}

// Increment increments a counter by 1.
func (mc *MetricsCollector) Increment(name string) {
	mc.IncrementBy(name, 1)
}

// IncrementBy increments a counter by a specific amount.
func (mc *MetricsCollector) IncrementBy(name string, value int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.counters[name] += value
}

// SetGauge sets a gauge value.
func (mc *MetricsCollector) SetGauge(name string, value float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.customMetrics[name] = value
}

// SetLabel sets a label (metadata) for the metrics.
func (mc *MetricsCollector) SetLabel(key, value string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.labels[key] = value
}

// GetCounter returns the value of a counter.
func (mc *MetricsCollector) GetCounter(name string) int64 {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.counters[name]
}

// GetHistogram returns all values in a histogram.
func (mc *MetricsCollector) GetHistogram(name string) []float64 {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	return mc.histograms[name]
}

// GetHistogramStats returns statistics for a histogram.
func (mc *MetricsCollector) GetHistogramStats(name string) HistogramStats {
	values := mc.GetHistogram(name)
	if len(values) == 0 {
		return HistogramStats{}
	}

	stats := HistogramStats{
		Count:  len(values),
		Min:    values[0],
		Max:    values[0],
		Sum:    0,
		Values: values,
	}

	for _, v := range values {
		if v < stats.Min {
			stats.Min = v
		}
		if v > stats.Max {
			stats.Max = v
		}
		stats.Sum += v
	}

	// Calculate percentiles
	sorted := make([]float64, len(values))
	copy(sorted, values)
	// Simple sort - for production use a proper sorting algorithm
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	stats.P50 = percentile(sorted, 50)
	stats.P90 = percentile(sorted, 90)
	stats.P95 = percentile(sorted, 95)
	stats.P99 = percentile(sorted, 99)

	return stats
}

// HistogramStats holds histogram statistics.
type HistogramStats struct {
	Count  int
	Min    float64
	Max    float64
	Sum    float64
	Avg    float64
	P50    float64
	P90    float64
	P95    float64
	P99    float64
	Values []float64
}

// percentile calculates a percentile from a sorted slice.
// Uses the nearest rank method.
func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if len(sorted) == 1 {
		return sorted[0]
	}
	// Calculate index using nearest rank method: (p * (n-1)) / 100
	index := (p * (len(sorted) - 1)) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

// Finish marks the end of metric collection.
func (mc *MetricsCollector) Finish() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.endTime = time.Now()
}

// Duration returns the total duration of the test run.
func (mc *MetricsCollector) Duration() time.Duration {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.endTime.IsZero() {
		return mc.endTime.Sub(mc.startTime)
	}
	return time.Since(mc.startTime)
}

// Summary returns a summary of all metrics.
func (mc *MetricsCollector) Summary() map[string]interface{} {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	summary := make(map[string]interface{})

	// Test counts
	summary["test_count"] = mc.testCount
	summary["passed_tests"] = mc.passedTests
	summary["failed_tests"] = mc.failedTests
	summary["skipped_tests"] = mc.skippedTests

	// Duration - calculate without calling Duration() to avoid deadlock
	var duration time.Duration
	if !mc.endTime.IsZero() {
		duration = mc.endTime.Sub(mc.startTime)
	} else {
		duration = time.Since(mc.startTime)
	}
	summary["duration_ms"] = duration.Milliseconds()

	// Counters
	counters := make(map[string]int64)
	for k, v := range mc.counters {
		counters[k] = v
	}
	summary["counters"] = counters

	// Histograms
	histograms := make(map[string]map[string]interface{})
	for name := range mc.histograms {
		stats := mc.GetHistogramStats(name)
		histograms[name] = map[string]interface{}{
			"count": stats.Count,
			"min":   stats.Min,
			"max":   stats.Max,
			"avg":   stats.Avg,
			"p50":   stats.P50,
			"p90":   stats.P90,
			"p95":   stats.P95,
			"p99":   stats.P99,
		}
	}
	summary["histograms"] = histograms

	// Labels
	summary["labels"] = mc.labels

	return summary
}

// JSON returns the metrics as JSON.
func (mc *MetricsCollector) JSON() (string, error) {
	summary := mc.Summary()
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal metrics: %w", err)
	}
	return string(data), nil
}

// PrintSummary prints a human-readable summary.
func (mc *MetricsCollector) PrintSummary() {
	summary := mc.Summary()

	fmt.Println("\n=== Test Metrics Summary ===")
	fmt.Printf("Tests: %d passed, %d failed, %d skipped\n",
		summary["passed_tests"], summary["failed_tests"], summary["skipped_tests"])
	fmt.Printf("Duration: %v\n", mc.Duration())

	if counters, ok := summary["counters"].(map[string]int64); ok {
		fmt.Println("\nCounters:")
		for k, v := range counters {
			fmt.Printf("  %s: %d\n", k, v)
		}
	}

	if histograms, ok := summary["histograms"].(map[string]map[string]interface{}); ok {
		fmt.Println("\nHistograms:")
		for name, stats := range histograms {
			fmt.Printf("  %s:\n", name)
			fmt.Printf("    count: %v\n", stats["count"])
			fmt.Printf("    min: %.2fms\n", stats["min"])
			fmt.Printf("    max: %.2fms\n", stats["max"])
			if p50, ok := stats["p50"].(float64); ok {
				fmt.Printf("    p50: %.2fms\n", p50)
			}
			if p90, ok := stats["p90"].(float64); ok {
				fmt.Printf("    p90: %.2fms\n", p90)
			}
		}
	}

	fmt.Println("=========================")
}

// Reset resets all metrics.
func (mc *MetricsCollector) Reset() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.startTime = time.Now()
	mc.endTime = time.Time{}
	mc.testCount = 0
	mc.passedTests = 0
	mc.failedTests = 0
	mc.skippedTests = 0
	mc.testDurations = make(map[string]time.Duration)
	mc.customMetrics = make(map[string]interface{})
	mc.timers = make(map[string]time.Time)
	mc.counters = make(map[string]int64)
	mc.histograms = make(map[string][]float64)
	mc.labels = make(map[string]string)
}

// TestMetrics wraps testing.T to collect metrics automatically.
type TestMetrics struct {
	*testing.T
	collector *MetricsCollector
	testName  string
	startTime time.Time
}

// NewTestMetrics creates a new test metrics wrapper.
func NewTestMetrics(t *testing.T, collector *MetricsCollector) *TestMetrics {
	return &TestMetrics{
		T:         t,
		collector: collector,
		testName:  t.Name(),
		startTime: time.Now(),
	}
}

// Cleanup registers cleanup and records the test result.
func (tm *TestMetrics) Cleanup() {
	duration := time.Since(tm.startTime)
	tm.collector.RecordTest(tm.testName, !tm.T.Failed(), duration)
}

// Logf logs with the test name prefix.
func (tm *TestMetrics) Logf(format string, args ...interface{}) {
	tm.T.Logf("[%s] "+format, append([]interface{}{tm.testName}, args...)...)
}

// Errorf records an error and logs it.
func (tm *TestMetrics) Errorf(format string, args ...interface{}) {
	tm.T.Errorf("[%s] "+format, append([]interface{}{tm.testName}, args...)...)
}

// Fatalf records a fatal error and fails the test.
func (tm *TestMetrics) Fatalf(format string, args ...interface{}) {
	tm.T.Fatalf("[%s] "+format, append([]interface{}{tm.testName}, args...)...)
}

// Helper marks the test as a helper.
func (tm *TestMetrics) Helper() {
	tm.T.Helper()
}

// Parallel marks the test as parallel.
func (tm *TestMetrics) Parallel() {
	tm.T.Parallel()
}

// Run runs a subtest with metrics collection.
func (tm *TestMetrics) Run(name string, f func(tm *TestMetrics)) bool {
	return tm.T.Run(name, func(t *testing.T) {
		subMetrics := &TestMetrics{
			T:         t,
			collector: tm.collector,
			testName:  tm.testName + "/" + name,
			startTime: time.Now(),
		}
		defer subMetrics.Cleanup()
		f(subMetrics)
	})
}

// WithMetrics wraps a testing.T to automatically collect metrics.
// Usage:
//
//	func TestMain(m *testing.M) {
//	    collector := testutil.NewMetricsCollector()
//	    os.Exit(testutil.RunWithMetrics(m, collector))
//	}
func WithMetrics(t *testing.T, collector *MetricsCollector) *TestMetrics {
	tm := &TestMetrics{
		T:         t,
		collector: collector,
		testName:  t.Name(),
		startTime: time.Now(),
	}
	t.Cleanup(tm.Cleanup)
	return tm
}

// RunWithMetrics runs tests with metrics collection.
func RunWithMetrics(m *testing.M, collector *MetricsCollector) int {
	collector.startTime = time.Now()
	result := m.Run()
	collector.Finish()
	collector.PrintSummary()
	return result
}

// BenchmarkMetrics collects metrics for benchmark tests.
type BenchmarkMetrics struct {
	collector *MetricsCollector
	name      string
}

// NewBenchmarkMetrics creates benchmark metrics.
func NewBenchmarkMetrics(name string) *BenchmarkMetrics {
	return &BenchmarkMetrics{
		collector: NewMetricsCollector(),
		name:      name,
	}
}

// ReportBenchmark reports benchmark results to the metrics collector.
// Use this after running a benchmark with b.ReportAllocs() and storing results.
func (bm *BenchmarkMetrics) ReportBenchmark(allocsPerOp int64, nsPerOp float64) {
	bm.collector.SetGauge(bm.name+"_allocs", float64(allocsPerOp))
	bm.collector.SetGauge(bm.name+"_ns_per_op", nsPerOp)
}

// PerformanceTracker tracks performance metrics during tests.
type PerformanceTracker struct {
	mu              sync.Mutex
	memorySnapshots map[string]MemorySnapshot
	timingData      map[string][]time.Duration
}

// MemorySnapshot holds memory statistics.
type MemorySnapshot struct {
	Timestamp time.Time
	Allocs    uint64
	Bytes     uint64
}

// NewPerformanceTracker creates a new performance tracker.
func NewPerformanceTracker() *PerformanceTracker {
	return &PerformanceTracker{
		memorySnapshots: make(map[string]MemorySnapshot),
		timingData:      make(map[string][]time.Duration),
	}
}

// StartTiming starts timing an operation.
func (pt *PerformanceTracker) StartTiming(operation string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		pt.mu.Lock()
		defer pt.mu.Unlock()
		pt.timingData[operation] = append(pt.timingData[operation], duration)
	}
}

// GetTimings returns all timings for an operation.
func (pt *PerformanceTracker) GetTimings(operation string) []time.Duration {
	pt.mu.Lock()
	defer pt.mu.Unlock()
	return pt.timingData[operation]
}

// GetAverageTiming returns the average timing for an operation.
func (pt *PerformanceTracker) GetAverageTiming(operation string) time.Duration {
	timings := pt.GetTimings(operation)
	if len(timings) == 0 {
		return 0
	}

	var sum time.Duration
	for _, t := range timings {
		sum += t
	}
	return sum / time.Duration(len(timings))
}

// ResourceMonitor monitors resource usage during tests.
type ResourceMonitor struct {
	mu       sync.Mutex
	start    time.Time
	checks   []ResourceCheck
	ctx      context.Context
	cancel   context.CancelFunc
	interval time.Duration
}

// ResourceCheck holds a single resource check result.
type ResourceCheck struct {
	Timestamp   time.Time
	MemoryBytes uint64
	Goroutines  int
}

// NewResourceMonitor creates a new resource monitor.
func NewResourceMonitor(interval time.Duration) *ResourceMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	return &ResourceMonitor{
		start:    time.Now(),
		ctx:      ctx,
		cancel:   cancel,
		interval: interval,
		checks:   make([]ResourceCheck, 0),
	}
}

// Start begins monitoring resources.
func (rm *ResourceMonitor) Start() {
	ticker := time.NewTicker(rm.interval)
	go func() {
		defer ticker.Stop()

		for {
			select {
			case <-rm.ctx.Done():
				return
			case <-ticker.C:
				rm.recordCheck()
			}
		}
	}()
}

// Stop stops monitoring resources.
func (rm *ResourceMonitor) Stop() {
	rm.cancel()
}

// recordCheck records current resource usage.
func (rm *ResourceMonitor) recordCheck() {
	// Note: In a real implementation, you would use runtime.MemStats
	// and runtime.NumGoroutine to get actual metrics
	rm.mu.Lock()
	defer rm.mu.Unlock()

	check := ResourceCheck{
		Timestamp: time.Now(),
		// These would be actual values in production
		MemoryBytes: 0,
		Goroutines:  0,
	}

	rm.checks = append(rm.checks, check)
}

// GetChecks returns all resource checks.
func (rm *ResourceMonitor) GetChecks() []ResourceCheck {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	return rm.checks
}

// GetPeakMemory returns the peak memory usage.
func (rm *ResourceMonitor) GetPeakMemory() uint64 {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	var peak uint64
	for _, check := range rm.checks {
		if check.MemoryBytes > peak {
			peak = check.MemoryBytes
		}
	}
	return peak
}

// GetAverageGoroutines returns the average number of goroutines.
func (rm *ResourceMonitor) GetAverageGoroutines() float64 {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if len(rm.checks) == 0 {
		return 0
	}

	var sum int
	for _, check := range rm.checks {
		sum += check.Goroutines
	}
	return float64(sum) / float64(len(rm.checks))
}

// CoverageTracker tracks test coverage metrics.
type CoverageTracker struct {
	mu             sync.Mutex
	coveredFiles   map[string]int
	totalFiles     int
	coveredLines   map[string]int
	totalLines     int
	coveredFuncs   map[string]int
	totalFuncs     int
}

// NewCoverageTracker creates a new coverage tracker.
func NewCoverageTracker() *CoverageTracker {
	return &CoverageTracker{
		coveredFiles: make(map[string]int),
		coveredLines: make(map[string]int),
		coveredFuncs: make(map[string]int),
	}
}

// RecordCoverage records coverage data for a file.
func (ct *CoverageTracker) RecordCoverage(file string, linesCovered, totalLines int, funcsCovered, totalFuncs int) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if totalLines > 0 {
		ct.coveredLines[file] = linesCovered
		ct.totalLines += totalLines
	}

	if totalFuncs > 0 {
		ct.coveredFuncs[file] = funcsCovered
		ct.totalFuncs += totalFuncs
	}

	if linesCovered > 0 {
		ct.coveredFiles[file] = linesCovered
	}
}

// GetLineCoverage returns the line coverage percentage.
func (ct *CoverageTracker) GetLineCoverage() float64 {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if ct.totalLines == 0 {
		return 0
	}

	var covered int
	for _, c := range ct.coveredLines {
		covered += c
	}

	return float64(covered) / float64(ct.totalLines) * 100
}

// GetFunctionCoverage returns the function coverage percentage.
func (ct *CoverageTracker) GetFunctionCoverage() float64 {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	if ct.totalFuncs == 0 {
		return 0
	}

	var covered int
	for _, c := range ct.coveredFuncs {
		covered += c
	}

	return float64(covered) / float64(ct.totalFuncs) * 100
}

// TimingHelper helps measure execution time in tests.
type TimingHelper struct {
	mu     sync.Mutex
	timers map[string]*time.Timer
}

// NewTimingHelper creates a new timing helper.
func NewTimingHelper() *TimingHelper {
	return &TimingHelper{
		timers: make(map[string]*time.Timer),
	}
}

// Time measures the duration of a function.
func (th *TimingHelper) Time(name string, fn func()) time.Duration {
	start := time.Now()
	fn()
	duration := time.Since(start)

	th.mu.Lock()
	defer th.mu.Unlock()
	// Store duration (could use a different structure in production)

	return duration
}

// TimeAsync measures the duration of an async function.
func (th *TimingHelper) TimeAsync(name string, fn func() error) (time.Duration, error) {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	th.mu.Lock()
	defer th.mu.Unlock()
	// Store duration

	return duration, err
}

// Measure creates a deferred timer that records duration when returned.
func (th *TimingHelper) Measure(name string) func() time.Duration {
	start := time.Now()
	return func() time.Duration {
		duration := time.Since(start)
		th.mu.Lock()
		defer th.mu.Unlock()
		// Store duration
		return duration
	}
}
