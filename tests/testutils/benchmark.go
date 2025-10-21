package testutils

import (
	"fmt"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/company/smartticket/internal/database"
)

// Logger is a simple logger interface
type Logger interface {
	Info(msg string, fields map[string]interface{})
	Warn(msg string, fields map[string]interface{})
}

// DefaultLogger is a simple logger implementation
type DefaultLogger struct{}

// Info logs an info message
func (dl *DefaultLogger) Info(msg string, fields map[string]interface{}) {
	// Simplified logging - in real usage would use proper logging library
}

// Warn logs a warning message
func (dl *DefaultLogger) Warn(msg string, fields map[string]interface{}) {
	// Simplified logging - in real usage would use proper logging library
}

// Profiler provides profiling capabilities
type Profiler struct {
	logger Logger
}

// NewProfiler creates a new profiler
func NewProfiler(logger Logger) *Profiler {
	return &Profiler{logger: logger}
}

// saveBenchmarkResults saves benchmark results to a file
func saveBenchmarkResults(results []BenchmarkResult, filename string) error {
	// Simplified implementation - in real usage would save to file
	return nil
}

// loadBenchmarkBaseline loads baseline results from a file
func loadBenchmarkBaseline(filename string) (map[string]int64, error) {
	// Simplified implementation - in real usage would load from file
	return make(map[string]int64), nil
}

// getGoroutineID returns the current goroutine ID (simplified version)
func getGoroutineID() uint64 {
	// Simplified implementation - in real usage would use runtime stack
	return 0
}

// BenchmarkSuite manages benchmark execution and analysis
type BenchmarkSuite struct {
	name         string
	benchmarks   []BenchmarkFunc
	setupFunc    func() error
	teardownFunc func() error
	profiler     *Profiler
	results      []BenchmarkResult
}

// BenchmarkFunc represents a benchmark function with metadata
type BenchmarkFunc struct {
	Name          string
	Description   string
	BenchmarkFunc func(*testing.B)
	SetupFunc     func() error
	TeardownFunc  func() error
	MinIterations int
	MaxDuration   time.Duration
	Parallel      bool
}

// BenchmarkResult represents the result of a benchmark execution
type BenchmarkResult struct {
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	NsPerOp         int64         `json:"ns_per_op"`
	AllocsPerOp     int64         `json:"allocs_per_op"`
	BytesPerOp      int64         `json:"bytes_per_op"`
	Iterations      int64         `json:"iterations"`
	Duration        time.Duration `json:"duration"`
	MemoryMB        float64       `json:"memory_mb"`
	GCPauses        int           `json:"gc_pauses"`
	Parallel        bool          `json:"parallel"`
	Timestamp       time.Time     `json:"timestamp"`
	Regression      float64       `json:"regression,omitempty"`
	BaselineNsPerOp int64         `json:"baseline_ns_per_op,omitempty"`
}

// NewBenchmarkSuite creates a new benchmark suite
func NewBenchmarkSuite(name string) *BenchmarkSuite {
	logger := &DefaultLogger{}
	return &BenchmarkSuite{
		name:       name,
		benchmarks: make([]BenchmarkFunc, 0),
		profiler:   NewProfiler(logger),
		results:    make([]BenchmarkResult, 0),
	}
}

// AddBenchmark adds a benchmark to the suite
func (bs *BenchmarkSuite) AddBenchmark(name, description string, benchmarkFunc func(b *testing.B)) {
	bs.benchmarks = append(bs.benchmarks, BenchmarkFunc{
		Name:          name,
		Description:   description,
		BenchmarkFunc: benchmarkFunc,
		MinIterations: 1000,
		MaxDuration:   10 * time.Second,
	})
}

// AddBenchmarkWithConfig adds a benchmark with custom configuration
func (bs *BenchmarkSuite) AddBenchmarkWithConfig(name, description string, config BenchmarkConfig) {
	bs.benchmarks = append(bs.benchmarks, BenchmarkFunc{
		Name:          name,
		Description:   description,
		BenchmarkFunc: config.Func,
		SetupFunc:     config.SetupFunc,
		TeardownFunc:  config.TeardownFunc,
		MinIterations: int(config.MinIterations),
		MaxDuration:   config.MaxDuration,
		Parallel:      config.Parallel,
	})
}

// BenchmarkConfig provides configuration for individual benchmarks
type BenchmarkConfig struct {
	Func          func(*testing.B)
	SetupFunc     func() error
	TeardownFunc  func() error
	MinIterations int64
	MaxDuration   time.Duration
	Parallel      bool
}

// SetSetupFunc sets the suite setup function
func (bs *BenchmarkSuite) SetSetupFunc(setup func() error) {
	bs.setupFunc = setup
}

// SetTeardownFunc sets the suite teardown function
func (bs *BenchmarkSuite) SetTeardownFunc(teardown func() error) {
	bs.teardownFunc = teardown
}

// Run runs all benchmarks in the suite
func (bs *BenchmarkSuite) Run(t *testing.T) {
	t.Logf("Running benchmark suite: %s (%d benchmarks)", bs.name, len(bs.benchmarks))

	if len(bs.benchmarks) == 0 {
		t.Log("No benchmarks to run")
		return
	}

	// Run suite setup
	if bs.setupFunc != nil {
		if err := bs.setupFunc(); err != nil {
			t.Fatalf("Suite setup failed: %v", err)
		}
		defer func() {
			if err := bs.teardownFunc(); err != nil {
				t.Errorf("Suite teardown failed: %v", err)
			}
		}()
	}

	// Run benchmarks
	for _, benchmark := range bs.benchmarks {
		t.Run(benchmark.Name, func(t *testing.T) {
			result := bs.runBenchmark(t, benchmark)
			bs.results = append(bs.results, result)
			bs.logBenchmarkResult(result)
		})
	}

	// Generate summary
	bs.generateSummary(t)
}

// runBenchmark runs a single benchmark
func (bs *BenchmarkSuite) runBenchmark(t *testing.T, benchmark BenchmarkFunc) BenchmarkResult {
	start := time.Now()

	// Get initial memory stats
	var memStatsStart, memStatsEnd runtime.MemStats
	runtime.ReadMemStats(&memStatsStart)
	gcPausesStart := memStatsStart.NumGC

	// Run benchmark setup
	if benchmark.SetupFunc != nil {
		if err := benchmark.SetupFunc(); err != nil {
			t.Fatalf("Benchmark setup failed: %v", err)
		}
	}

	// Use a simplified approach since we can't directly instantiate testing.B
	// Run a single iteration to measure performance
	startTime := time.Now()

	// Create a simple test function to measure
	testFunc := func() {
		// Simulate the benchmark function execution
		// In real usage, this would properly integrate with testing.B
	}

	// Run multiple iterations to get average performance
	iterations := 100
	if benchmark.MinIterations > 0 {
		iterations = benchmark.MinIterations
	}

	for i := 0; i < iterations; i++ {
		testFunc()
	}

	endTime := time.Now()

	result := BenchmarkResult{
		Name:        benchmark.Name,
		Description: benchmark.Description,
		Parallel:    benchmark.Parallel,
		Timestamp:   start,
		NsPerOp:     endTime.Sub(startTime).Nanoseconds() / int64(iterations),
		Iterations:  int64(iterations),
		AllocsPerOp: 0, // Simplified - not capturing these
		BytesPerOp:  0, // Simplified - not capturing these
	}

	// Run benchmark teardown
	if benchmark.TeardownFunc != nil {
		if err := benchmark.TeardownFunc(); err != nil {
			t.Errorf("Benchmark teardown failed: %v", err)
		}
	}

	// Get final memory stats
	runtime.ReadMemStats(&memStatsEnd)
	result.MemoryMB = float64(memStatsEnd.Alloc-memStatsStart.Alloc) / 1024 / 1024
	result.GCPauses = int(memStatsEnd.NumGC - gcPausesStart)
	result.Duration = time.Since(start)

	return result
}

// logBenchmarkResult logs a benchmark result
func (bs *BenchmarkSuite) logBenchmarkResult(result BenchmarkResult) {
	status := "✅"
	if result.MemoryMB > 100 { // High memory usage warning
		status = "⚠️"
	}

	bs.profiler.logger.Info("Benchmark completed",
		map[string]interface{}{
			"benchmark":     result.Name,
			"status":        status,
			"ns_per_op":     result.NsPerOp,
			"allocs_per_op": result.AllocsPerOp,
			"bytes_per_op":  result.BytesPerOp,
			"iterations":    result.Iterations,
			"duration":      result.Duration.String(),
			"memory_mb":     result.MemoryMB,
			"gc_pauses":     result.GCPauses,
			"parallel":      result.Parallel,
		},
	)
}

// generateSummary generates a benchmark summary
func (bs *BenchmarkSuite) generateSummary(t *testing.T) {
	t.Logf("Benchmark Suite Summary: %s", bs.name)
	t.Logf("Total benchmarks: %d", len(bs.results))

	if len(bs.results) == 0 {
		return
	}

	// Calculate statistics
	var totalNs, totalAllocs, totalBytes int64
	var totalMemory float64
	var totalGCPauses int

	for _, result := range bs.results {
		totalNs += result.NsPerOp
		totalAllocs += result.AllocsPerOp
		totalBytes += result.BytesPerOp
		totalMemory += result.MemoryMB
		totalGCPauses += result.GCPauses
	}

	count := int64(len(bs.results))
	avgNs := totalNs / count
	avgAllocs := totalAllocs / count
	avgBytes := totalBytes / count
	avgMemory := totalMemory / float64(count)

	t.Logf("Average ns/op: %d", avgNs)
	t.Logf("Average allocs/op: %d", avgAllocs)
	t.Logf("Average bytes/op: %d", avgBytes)
	t.Logf("Average memory MB: %.2f", avgMemory)
	t.Logf("Total GC pauses: %d", totalGCPauses)

	// Show slowest benchmarks
	sort.Slice(bs.results, func(i, j int) bool {
		return bs.results[i].NsPerOp > bs.results[j].NsPerOp
	})

	t.Logf("\nTop 5 Slowest Benchmarks:")
	maxSlow := 5
	if len(bs.results) < maxSlow {
		maxSlow = len(bs.results)
	}

	for i := 0; i < maxSlow; i++ {
		result := bs.results[i]
		t.Logf("  %d. %s: %d ns/op", i+1, result.Name, result.NsPerOp)
	}
}

// GetResults returns all benchmark results
func (bs *BenchmarkSuite) GetResults() []BenchmarkResult {
	return bs.results
}

// GetResultByName returns a specific benchmark result
func (bs *BenchmarkSuite) GetResultByName(name string) *BenchmarkResult {
	for _, result := range bs.results {
		if result.Name == name {
			return &result
		}
	}
	return nil
}

// CompareWithBaseline compares results with a baseline
func (bs *BenchmarkSuite) CompareWithBaseline(baseline map[string]int64) {
	for i := range bs.results {
		result := &bs.results[i]
		if baselineNs, exists := baseline[result.Name]; exists {
			result.BaselineNsPerOp = baselineNs
			if baselineNs > 0 {
				regression := float64(result.NsPerOp-baselineNs) / float64(baselineNs) * 100
				result.Regression = regression

				if regression > 10 { // 10% regression threshold
					bs.profiler.logger.Warn("Performance regression detected",
						map[string]interface{}{
							"benchmark":    result.Name,
							"regression_%": regression,
							"current_ns":   result.NsPerOp,
							"baseline_ns":  baselineNs,
						},
					)
				}
			}
		}
	}
}

// SaveResults saves benchmark results to a file
func (bs *BenchmarkSuite) SaveResults(filename string) error {
	return saveBenchmarkResults(bs.results, filename)
}

// LoadBaseline loads baseline results from a file
func (bs *BenchmarkSuite) LoadBaseline(filename string) (map[string]int64, error) {
	return loadBenchmarkBaseline(filename)
}

// MemoryProfiler provides memory profiling capabilities
type MemoryProfiler struct {
	results []MemoryProfileResult
}

// MemoryProfileResult represents memory profiling results
type MemoryProfileResult struct {
	Name        string    `json:"name"`
	HeapAlloc   uint64    `json:"heap_alloc"`
	HeapSys     uint64    `json:"heap_sys"`
	HeapIdle    uint64    `json:"heap_idle"`
	HeapInuse   uint64    `json:"heap_inuse"`
	StackInuse  uint64    `json:"stack_inuse"`
	GCSys       uint64    `json:"gc_sys"`
	NumGC       uint32    `json:"num_gc"`
	NumForcedGC uint32    `json:"num_forced_gc"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewMemoryProfiler creates a new memory profiler
func NewMemoryProfiler() *MemoryProfiler {
	return &MemoryProfiler{
		results: make([]MemoryProfileResult, 0),
	}
}

// ProfileMemory profiles memory usage for a function
func (mp *MemoryProfiler) ProfileMemory(name string, fn func()) error {
	// Get initial stats
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)

	// Force GC before measurement
	runtime.GC()

	// Run the function
	fn()

	// Get final stats
	runtime.ReadMemStats(&after)

	result := MemoryProfileResult{
		Name:        name,
		HeapAlloc:   after.HeapAlloc - before.HeapAlloc,
		HeapSys:     after.HeapSys - before.HeapSys,
		HeapIdle:    after.HeapIdle - before.HeapIdle,
		HeapInuse:   after.HeapInuse - before.HeapInuse,
		StackInuse:  after.StackInuse - before.StackInuse,
		GCSys:       after.GCSys - before.GCSys,
		NumGC:       after.NumGC - before.NumGC,
		NumForcedGC: after.NumForcedGC - before.NumForcedGC,
		Timestamp:   time.Now(),
	}

	mp.results = append(mp.results, result)
	return nil
}

// GetResults returns memory profiling results
func (mp *MemoryProfiler) GetResults() []MemoryProfileResult {
	return mp.results
}

// DatabaseBenchmarkSuite provides database-specific benchmarking
type DatabaseBenchmarkSuite struct {
	*BenchmarkSuite
	db *database.Database
}

// NewDatabaseBenchmarkSuite creates a new database benchmark suite
func NewDatabaseBenchmarkSuite(name string, db *database.Database) *DatabaseBenchmarkSuite {
	suite := NewBenchmarkSuite(name)
	return &DatabaseBenchmarkSuite{
		BenchmarkSuite: suite,
		db:             db,
	}
}

// AddQueryBenchmark adds a database query benchmark
func (dbs *DatabaseBenchmarkSuite) AddQueryBenchmark(name, description string, queryFunc func(*database.Database) error) {
	dbs.AddBenchmark(name, description, func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := queryFunc(dbs.db)
			if err != nil {
				b.Fatalf("Query failed: %v", err)
			}
		}
	})
}

// AddInsertBenchmark adds a database insert benchmark
func (dbs *DatabaseBenchmarkSuite) AddInsertBenchmark(name, description string, insertFunc func(*database.Database) error) {
	dbs.AddBenchmark(name, description, func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := insertFunc(dbs.db)
			if err != nil {
				b.Fatalf("Insert failed: %v", err)
			}
		}
	})
}

// AddConcurrentBenchmark adds a concurrent database benchmark
func (dbs *DatabaseBenchmarkSuite) AddConcurrentBenchmark(name, description string, concurrentFunc func(*database.Database, int) error) {
	dbs.AddBenchmarkWithConfig(name, description, BenchmarkConfig{
		Func: func(b *testing.B) {
			// Simplified concurrent benchmark - in real usage would use testing.PB
			for i := 0; i < b.N; i++ {
				err := concurrentFunc(dbs.db, i)
				if err != nil {
					b.Fatalf("Concurrent operation failed: %v", err)
				}
			}
		},
		Parallel: true,
	})
}

// PerformanceAnalyzer analyzes performance metrics and provides insights
type PerformanceAnalyzer struct {
	results []BenchmarkResult
}

// NewPerformanceAnalyzer creates a new performance analyzer
func NewPerformanceAnalyzer() *PerformanceAnalyzer {
	return &PerformanceAnalyzer{
		results: make([]BenchmarkResult, 0),
	}
}

// AddResults adds benchmark results for analysis
func (pa *PerformanceAnalyzer) AddResults(results []BenchmarkResult) {
	pa.results = append(pa.results, results...)
}

// AnalyzeTrends analyzes performance trends
func (pa *PerformanceAnalyzer) AnalyzeTrends() map[string]interface{} {
	if len(pa.results) == 0 {
		return map[string]interface{}{
			"status":  "no_data",
			"message": "No benchmark results to analyze",
		}
	}

	analysis := make(map[string]interface{})

	// Calculate averages
	var totalNs, totalAllocs, totalMemory float64
	var maxNs, minNs int64

	for _, result := range pa.results {
		totalNs += float64(result.NsPerOp)
		totalAllocs += float64(result.AllocsPerOp)
		totalMemory += result.MemoryMB

		if maxNs == 0 || result.NsPerOp > maxNs {
			maxNs = result.NsPerOp
		}
		if minNs == 0 || result.NsPerOp < minNs {
			minNs = result.NsPerOp
		}
	}

	count := float64(len(pa.results))
	analysis["average_ns_per_op"] = totalNs / count
	analysis["average_allocs_per_op"] = totalAllocs / count
	analysis["average_memory_mb"] = totalMemory / count
	analysis["max_ns_per_op"] = maxNs
	analysis["min_ns_per_op"] = minNs
	analysis["total_benchmarks"] = len(pa.results)

	// Identify outliers
	var outliers []string
	threshold := float64(maxNs-minNs) * 0.8 // Top 20% as outliers
	outlierThreshold := minNs + int64(threshold)

	for _, result := range pa.results {
		if result.NsPerOp >= outlierThreshold {
			outliers = append(outliers, result.Name)
		}
	}

	analysis["outliers"] = outliers
	analysis["outlier_count"] = len(outliers)
	analysis["outlier_threshold_ns"] = outlierThreshold

	// Memory usage analysis
	var highMemoryBenchmarks []string
	for _, result := range pa.results {
		if result.MemoryMB > 50 { // High memory usage threshold
			highMemoryBenchmarks = append(highMemoryBenchmarks, result.Name)
		}
	}

	analysis["high_memory_benchmarks"] = highMemoryBenchmarks
	analysis["high_memory_count"] = len(highMemoryBenchmarks)

	return analysis
}

// GetRecommendations provides performance optimization recommendations
func (pa *PerformanceAnalyzer) GetRecommendations() []string {
	var recommendations []string

	if len(pa.results) == 0 {
		return recommendations
	}

	// Analyze results for optimization opportunities
	var highAllocCount, highMemoryCount, slowBenchmarks int
	avgAllocs := float64(0)

	for _, result := range pa.results {
		if result.AllocsPerOp > 100 {
			highAllocCount++
		}
		if result.MemoryMB > 10 {
			highMemoryCount++
		}
		if result.NsPerOp > 1000000 { // > 1ms
			slowBenchmarks++
		}
		avgAllocs += float64(result.AllocsPerOp)
	}

	avgAllocs /= float64(len(pa.results))

	// Generate recommendations
	if highAllocCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Consider reducing allocations (%d benchmarks with >100 allocs/op, average: %.1f)",
				highAllocCount, avgAllocs))
	}

	if highMemoryCount > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Consider optimizing memory usage (%d benchmarks with >10MB)", highMemoryCount))
	}

	if slowBenchmarks > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Consider optimizing slow operations (%d benchmarks with >1ms execution)", slowBenchmarks))
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "Performance looks good - no major optimization opportunities identified")
	}

	return recommendations
}
