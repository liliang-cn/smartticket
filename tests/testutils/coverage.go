package testutils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// CoverageReport represents a detailed coverage report.
type CoverageReport struct {
	OverallCoverage float64                 `json:"overall_coverage"`
	Modules         []ModuleCoverage        `json:"modules"`
	CoverageByFile  map[string]FileCoverage `json:"coverage_by_file"`
	GeneratedAt     time.Time               `json:"generated_at"`
	TestRunID       string                  `json:"test_run_id"`
}

// ModuleCoverage represents coverage for a module.
type ModuleCoverage struct {
	ModuleName       string  `json:"module_name"`
	PackagePath      string  `json:"package_path"`
	LineCoverage     float64 `json:"line_coverage"`
	BranchCoverage   float64 `json:"branch_coverage"`
	FunctionCoverage float64 `json:"function_coverage"`
	TotalLines       int     `json:"total_lines"`
	CoveredLines     int     `json:"covered_lines"`
	UncoveredLines   []int   `json:"uncovered_lines"`
}

// FileCoverage represents coverage for a single file.
type FileCoverage struct {
	FilePath       string             `json:"file_path"`
	LineCoverage   float64            `json:"line_coverage"`
	TotalLines     int                `json:"total_lines"`
	CoveredLines   int                `json:"covered_lines"`
	UncoveredLines []int              `json:"uncovered_lines"`
	Functions      []FunctionCoverage `json:"functions"`
}

// FunctionCoverage represents coverage for a function.
type FunctionCoverage struct {
	FunctionName   string  `json:"function_name"`
	StartLine      int     `json:"start_line"`
	EndLine        int     `json:"end_line"`
	Coverage       float64 `json:"coverage"`
	CoveredLines   []int   `json:"covered_lines"`
	UncoveredLines []int   `json:"uncovered_lines"`
}

// CoverageConfig represents coverage collection configuration.
type CoverageConfig struct {
	OutputDir       string            `json:"output_dir"`
	Threshold       float64           `json:"threshold"`
	ExcludePatterns []string          `json:"exclude_patterns"`
	IncludePatterns []string          `json:"include_patterns"`
	Modules         map[string]string `json:"modules"` // module name -> package path
}

// NewCoverageConfig creates a new coverage configuration.
func NewCoverageConfig() *CoverageConfig {
	return &CoverageConfig{
		OutputDir: "qa/coverage",
		Threshold: 100.0, // Constitution requirement
		ExcludePatterns: []string{
			"*/mocks/*",
			"*/testutils/*",
			"*/fixtures/*",
			"*/generated/*",
			"cmd/server/main.go",
		},
		IncludePatterns: []string{
			"./internal/...",
			"./pkg/...",
		},
		Modules: map[string]string{
			"Models":          "internal/models",
			"Services":        "internal/services",
			"Repositories":    "internal/repositories",
			"API Handlers":    "internal/api/handlers",
			"API Middleware":  "internal/api/middleware",
			"Database":        "internal/database",
			"Config":          "internal/config",
			"Auth":            "internal/auth",
			"Utils":           "internal/utils",
			"Tenant":          "internal/tenant",
			"Logger":          "internal/logger",
			"Public Packages": "pkg",
		},
	}
}

// CollectCoverage collects test coverage information.
func CollectCoverage(t *testing.T, config *CoverageConfig) (*CoverageReport, error) {
	if config == nil {
		config = NewCoverageConfig()
	}

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create coverage output directory: %w", err)
	}

	// Generate test run ID
	testRunID := fmt.Sprintf("coverage-%s", time.Now().Format("20060102-150405"))

	// Run tests with coverage
	coverageFile := filepath.Join(config.OutputDir, fmt.Sprintf("coverage-%s.out", testRunID))

	args := []string{"test", "-coverprofile", coverageFile, "-v"}
	args = append(args, config.IncludePatterns...)
	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run tests with coverage: %w", err)
	}

	// Parse coverage file
	report, err := parseCoverageFile(t, coverageFile, config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse coverage file: %w", err)
	}

	// Set metadata
	report.GeneratedAt = time.Now()
	report.TestRunID = testRunID

	// Generate HTML report
	htmlFile := filepath.Join(config.OutputDir, fmt.Sprintf("coverage-%s.html", testRunID))
	if err := generateHTMLReport(coverageFile, htmlFile); err != nil {
		t.Logf("Warning: failed to generate HTML coverage report: %v", err)
	}

	// Save JSON report
	jsonFile := filepath.Join(config.OutputDir, fmt.Sprintf("coverage-%s.json", testRunID))
	if err := saveCoverageReport(report, jsonFile); err != nil {
		t.Logf("Warning: failed to save JSON coverage report: %v", err)
	}

	return report, nil
}

// parseCoverageFile parses a go coverage file and generates a detailed report.
func parseCoverageFile(t *testing.T, coverageFile string, config *CoverageConfig) (*CoverageReport, error) {
	file, err := os.Open(coverageFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open coverage file: %w", err)
	}
	defer file.Close()

	report := &CoverageReport{
		CoverageByFile: make(map[string]FileCoverage),
		Modules:        make([]ModuleCoverage, 0),
	}

	// Parse mode line
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return nil, fmt.Errorf("coverage file is empty")
	}
	modeLine := scanner.Text()
	if !strings.HasPrefix(modeLine, "mode: ") {
		return nil, fmt.Errorf("invalid coverage file format")
	}

	// Parse coverage data
	lineRe := regexp.MustCompile(`^([^:]+):(\d+)\.(\d+),(\d+)\.(\d+) (\d+) (\d+)$`)
	totalLines, coveredLines := 0, 0

	for scanner.Scan() {
		line := scanner.Text()
		matches := lineRe.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		filePath := matches[1]
		startLine, _ := strconv.Atoi(matches[2])
		_, _ = strconv.Atoi(matches[3]) // startCol - not used
		endLine, _ := strconv.Atoi(matches[4])
		_, _ = strconv.Atoi(matches[5]) // endCol - not used
		numStatements, _ := strconv.Atoi(matches[6])
		count, _ := strconv.Atoi(matches[7])

		// Skip excluded patterns
		if shouldExcludeFile(filePath, config.ExcludePatterns) {
			continue
		}

		// Update file coverage
		fileCoverage := report.CoverageByFile[filePath]
		fileCoverage.FilePath = filePath
		fileCoverage.TotalLines += endLine - startLine + 1
		if count > 0 {
			fileCoverage.CoveredLines += numStatements
			coveredLines += numStatements
		}
		totalLines += numStatements

		// Track uncovered lines
		if count == 0 {
			for i := startLine; i <= endLine; i++ {
				fileCoverage.UncoveredLines = append(fileCoverage.UncoveredLines, i)
			}
		}

		report.CoverageByFile[filePath] = fileCoverage
	}

	// Calculate overall coverage
	if totalLines > 0 {
		report.OverallCoverage = float64(coveredLines) / float64(totalLines) * 100
	}

	// Group by modules
	for filePath, fileCoverage := range report.CoverageByFile {
		// Calculate file coverage percentage
		if fileCoverage.TotalLines > 0 {
			fileCoverage.LineCoverage = float64(fileCoverage.CoveredLines) / float64(fileCoverage.TotalLines) * 100
		}

		// Determine module
		moduleName, packagePath := determineModule(filePath, config.Modules)
		if moduleName == "" {
			continue
		}

		// Find or create module coverage
		var moduleCoverage *ModuleCoverage
		for i, mc := range report.Modules {
			if mc.ModuleName == moduleName {
				moduleCoverage = &report.Modules[i]
				break
			}
		}
		if moduleCoverage == nil {
			moduleCoverage = &ModuleCoverage{
				ModuleName:  moduleName,
				PackagePath: packagePath,
			}
			report.Modules = append(report.Modules, *moduleCoverage)
			moduleCoverage = &report.Modules[len(report.Modules)-1]
		}

		// Aggregate module coverage
		moduleCoverage.TotalLines += fileCoverage.TotalLines
		moduleCoverage.CoveredLines += fileCoverage.CoveredLines
		moduleCoverage.UncoveredLines = append(moduleCoverage.UncoveredLines, fileCoverage.UncoveredLines...)
	}

	// Calculate module coverage percentages
	for i := range report.Modules {
		if report.Modules[i].TotalLines > 0 {
			report.Modules[i].LineCoverage = float64(report.Modules[i].CoveredLines) / float64(report.Modules[i].TotalLines) * 100
		}
	}

	return report, nil
}

// shouldExcludeFile checks if a file should be excluded from coverage analysis.
func shouldExcludeFile(filePath string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
	}
	return false
}

// determineModule determines which module a file belongs to.
func determineModule(filePath string, modules map[string]string) (string, string) {
	for moduleName, packagePath := range modules {
		if strings.Contains(filePath, packagePath) {
			return moduleName, packagePath
		}
	}
	return "", ""
}

// generateHTMLReport generates an HTML coverage report.
func generateHTMLReport(coverageFile, htmlFile string) error {
	cmd := exec.Command("go", "tool", "cover", "-html", coverageFile, "-o", htmlFile)
	return cmd.Run()
}

// saveCoverageReport saves the coverage report as JSON.
func saveCoverageReport(report *CoverageReport, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

// AssertCoverageThreshold asserts that coverage meets the required threshold.
func AssertCoverageThreshold(t *testing.T, report *CoverageReport, threshold float64) {
	if report.OverallCoverage < threshold {
		t.Errorf("Coverage %.2f%% is below required threshold %.2f%%", report.OverallCoverage, threshold)

		// Show uncovered modules
		for _, module := range report.Modules {
			if module.LineCoverage < threshold {
				t.Errorf("Module %s coverage %.2f%% is below threshold", module.ModuleName, module.LineCoverage)
			}
		}
	} else {
		t.Logf("Coverage %.2f%% meets threshold %.2f%%", report.OverallCoverage, threshold)
	}
}

// GetUncoveredLines returns uncovered lines for a specific file.
func GetUncoveredLines(t *testing.T, report *CoverageReport, filePath string) []int {
	fileCoverage, exists := report.CoverageByFile[filePath]
	if !exists {
		return nil
	}
	return fileCoverage.UncoveredLines
}

// GetModuleCoverage returns coverage for a specific module.
func GetModuleCoverage(t *testing.T, report *CoverageReport, moduleName string) *ModuleCoverage {
	for _, module := range report.Modules {
		if module.ModuleName == moduleName {
			return &module
		}
	}
	return nil
}

// GenerateCoverageSummary generates a human-readable coverage summary.
func GenerateCoverageSummary(report *CoverageReport) string {
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Coverage Report - %s\n", report.GeneratedAt.Format("2006-01-02 15:04:05")))
	summary.WriteString(fmt.Sprintf("Test Run ID: %s\n", report.TestRunID))
	summary.WriteString(fmt.Sprintf("Overall Coverage: %.2f%%\n\n", report.OverallCoverage))

	summary.WriteString("Module Coverage:\n")
	for _, module := range report.Modules {
		status := "✅"
		if module.LineCoverage < 100.0 {
			status = "❌"
		}
		summary.WriteString(fmt.Sprintf("  %s %s: %.2f%% (%d/%d lines)\n",
			status, module.ModuleName, module.LineCoverage,
			module.CoveredLines, module.TotalLines))
	}

	// Show modules below threshold
	summary.WriteString("\nModules Needing Attention:\n")
	for _, module := range report.Modules {
		if module.LineCoverage < 100.0 {
			summary.WriteString(fmt.Sprintf("  📊 %s: %.2f%% - %d uncovered lines\n",
				module.ModuleName, module.LineCoverage, len(module.UncoveredLines)))
		}
	}

	return summary.String()
}

// WithCoverageAssertion is a helper that runs a test with coverage collection and threshold assertion.
func WithCoverageAssertion(t *testing.T, threshold float64, testFunc func(*testing.T)) {
	config := NewCoverageConfig()
	config.Threshold = threshold

	// Run the test function first
	testFunc(t)

	// Collect coverage
	report, err := CollectCoverage(t, config)
	if err != nil {
		t.Fatalf("Failed to collect coverage: %v", err)
	}

	// Print summary
	summary := GenerateCoverageSummary(report)
	t.Logf("Coverage Summary:\n%s", summary)

	// Assert threshold
	AssertCoverageThreshold(t, report, threshold)
}

// MergeCoverageReports merges multiple coverage reports.
func MergeCoverageReports(reports []*CoverageReport) *CoverageReport {
	if len(reports) == 0 {
		return nil
	}

	merged := &CoverageReport{
		CoverageByFile: make(map[string]FileCoverage),
		Modules:        make([]ModuleCoverage, 0),
		GeneratedAt:    time.Now(),
		TestRunID:      fmt.Sprintf("merged-%s", time.Now().Format("20060102-150405")),
	}

	totalLines, coveredLines := 0, 0
	moduleAggregates := make(map[string]*ModuleCoverage)

	for _, report := range reports {
		// Merge file coverage
		for filePath, fileCoverage := range report.CoverageByFile {
			existing, exists := merged.CoverageByFile[filePath]
			if !exists {
				merged.CoverageByFile[filePath] = fileCoverage
				existing = fileCoverage
			} else {
				// Merge line numbers (union of uncovered lines)
				uncoveredMap := make(map[int]bool)
				for _, line := range existing.UncoveredLines {
					uncoveredMap[line] = true
				}
				for _, line := range fileCoverage.UncoveredLines {
					if !uncoveredMap[line] {
						existing.UncoveredLines = append(existing.UncoveredLines, line)
					}
				}

				// Take max of totals and covered lines
				if fileCoverage.TotalLines > existing.TotalLines {
					existing.TotalLines = fileCoverage.TotalLines
				}
				if fileCoverage.CoveredLines > existing.CoveredLines {
					existing.CoveredLines = fileCoverage.CoveredLines
				}
			}

			totalLines += existing.TotalLines
			coveredLines += existing.CoveredLines
		}

		// Aggregate by modules
		for _, module := range report.Modules {
			aggregate, exists := moduleAggregates[module.ModuleName]
			if !exists {
				aggregate = &ModuleCoverage{
					ModuleName:  module.ModuleName,
					PackagePath: module.PackagePath,
				}
				moduleAggregates[module.ModuleName] = aggregate
			}

			aggregate.TotalLines += module.TotalLines
			aggregate.CoveredLines += module.CoveredLines
			aggregate.UncoveredLines = append(aggregate.UncoveredLines, module.UncoveredLines...)
		}
	}

	// Calculate overall coverage
	if totalLines > 0 {
		merged.OverallCoverage = float64(coveredLines) / float64(totalLines) * 100
	}

	// Convert module aggregates to slice
	for _, aggregate := range moduleAggregates {
		if aggregate.TotalLines > 0 {
			aggregate.LineCoverage = float64(aggregate.CoveredLines) / float64(aggregate.TotalLines) * 100
		}
		merged.Modules = append(merged.Modules, *aggregate)
	}

	return merged
}
