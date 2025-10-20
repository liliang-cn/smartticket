package testutils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// TestResult represents the result of running tests
type TestResult struct {
	Passed   int
	Failed   int
	Skipped  int
	Coverage float64
	Duration string
	Output   string
	Error    error
}

// TestRunner provides functionality to run Go tests
type TestRunner struct {
	RootDir  string
	TestDir  string
	Coverage bool
	Verbose  bool
	Race     bool
}

// NewTestRunner creates a new test runner
func NewTestRunner(rootDir string) *TestRunner {
	return &TestRunner{
		RootDir:  rootDir,
		TestDir:  filepath.Join(rootDir, "..."),
		Coverage: true,
		Verbose:  true,
		Race:     false, // Disabled by default for speed
	}
}

// RunAllTests runs all tests in the project
func (tr *TestRunner) RunAllTests() *TestResult {
	args := []string{
		"test",
		"-v",
		tr.TestDir,
	}

	if tr.Coverage {
		args = append(args, "-coverprofile=coverage.out")
	}

	if tr.Race {
		args = append(args, "-race")
	}

	return tr.runGoCommand(args)
}

// RunUnitTests runs only unit tests (excluding integration and e2e tests)
func (tr *TestRunner) RunUnitTests() *TestResult {
	args := []string{
		"test",
		"-v",
		"-tags=unit",
		"./...",
	}

	if tr.Coverage {
		args = append(args, "-coverprofile=coverage.out")
	}

	if tr.Race {
		args = append(args, "-race")
	}

	return tr.runGoCommand(args)
}

// RunIntegrationTests runs integration tests
func (tr *TestRunner) RunIntegrationTests() *TestResult {
	args := []string{
		"test",
		"-v",
		"-tags=integration",
		"./tests/integration/...",
	}

	if tr.Coverage {
		args = append(args, "-coverprofile=coverage_integration.out")
	}

	if tr.Race {
		args = append(args, "-race")
	}

	return tr.runGoCommand(args)
}

// RunE2ETests runs end-to-end tests
func (tr *TestRunner) RunE2ETests() *TestResult {
	args := []string{
		"test",
		"-v",
		"-tags=e2e",
		"./tests/e2e/...",
	}

	if tr.Coverage {
		args = append(args, "-coverprofile=coverage_e2e.out")
	}

	if tr.Race {
		args = append(args, "-race")
	}

	return tr.runGoCommand(args)
}

// RunLinting runs golangci-lint on the codebase
func (tr *TestRunner) RunLinting() *TestResult {
	args := []string{
		"run",
		"--timeout=5m",
		"--verbose",
		"./...",
	}

	return tr.runCommand("golangci-lint", args)
}

// GenerateCoverageReport generates an HTML coverage report
func (tr *TestRunner) GenerateCoverageReport() error {
	if !tr.Coverage {
		return fmt.Errorf("coverage not enabled")
	}

	// Check if coverage.out exists
	if _, err := os.Stat("coverage.out"); os.IsNotExist(err) {
		return fmt.Errorf("coverage.out file not found")
	}

	// Generate HTML coverage report
	cmd := exec.Command("go", "tool", "cover", "-html=coverage.out", "-o=coverage.html")
	cmd.Dir = tr.RootDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to generate coverage report: %v\nOutput: %s", err, string(output))
	}

	fmt.Println("Coverage report generated: coverage.html")
	return nil
}

// ParseTestOutput parses the output of go test to extract statistics
func (tr *TestRunner) ParseTestOutput(output string) (int, int, int, string) {
	lines := strings.Split(output, "\n")
	var passed, failed, skipped int
	var duration string

	// Regular expressions to match test results
	passRegex := regexp.MustCompile(`^--- PASS: (\S+)`)
	failRegex := regexp.MustCompile(`^--- FAIL: (\S+)`)
	skipRegex := regexp.MustCompile(`^--- SKIP: (\S+)`)
	durationRegex := regexp.MustCompile(`^ok\s+[\w\./]+\s+([\d.]+s)`)

	for _, line := range lines {
		if passRegex.MatchString(line) {
			passed++
		} else if failRegex.MatchString(line) {
			failed++
		} else if skipRegex.MatchString(line) {
			skipped++
		} else if matches := durationRegex.FindStringSubmatch(line); len(matches) > 1 {
			duration = matches[1]
		}
	}

	return passed, failed, skipped, duration
}

// ParseCoverage parses coverage from output
func (tr *TestRunner) ParseCoverage(output string) float64 {
	// Look for coverage percentage in output
	coverageRegex := regexp.MustCompile(`coverage:\s*(\d+(?:\.\d+)?)%`)
	matches := coverageRegex.FindStringSubmatch(output)
	if len(matches) > 1 {
		if coverage, err := strconv.ParseFloat(matches[1], 64); err == nil {
			return coverage
		}
	}
	return 0.0
}

// runGoCommand executes a go command with given arguments
func (tr *TestRunner) runGoCommand(args []string) *TestResult {
	return tr.runCommand("go", args)
}

// runCommand executes any command with given arguments
func (tr *TestRunner) runCommand(command string, args []string) *TestResult {
	cmd := exec.Command(command, args...)
	cmd.Dir = tr.RootDir

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		return &TestResult{
			Output: outputStr,
			Error:  err,
		}
	}

	// Parse test results if it's a go test command
	if command == "go" && len(args) > 0 && args[0] == "test" {
		passed, failed, skipped, duration := tr.ParseTestOutput(outputStr)
		coverage := tr.ParseCoverage(outputStr)

		return &TestResult{
			Passed:   passed,
			Failed:   failed,
			Skipped:  skipped,
			Coverage: coverage,
			Duration: duration,
			Output:   outputStr,
		}
	}

	return &TestResult{
		Output: outputStr,
	}
}

// RunBenchmark runs benchmarks
func (tr *TestRunner) RunBenchmark() *TestResult {
	args := []string{
		"test",
		"-bench=.",
		"-benchmem",
		"./...",
	}

	return tr.runGoCommand(args)
}

// RunTestsForPackage runs tests for a specific package
func (tr *TestRunner) RunTestsForPackage(pkg string) *TestResult {
	args := []string{
		"test",
		"-v",
		pkg,
	}

	if tr.Coverage {
		args = append(args, "-coverprofile=coverage_"+strings.ReplaceAll(pkg, "/", "_")+".out")
	}

	return tr.runGoCommand(args)
}

// CheckTestDependencies checks if all required testing dependencies are available
func (tr *TestRunner) CheckTestDependencies() *TestResult {
	deps := []string{
		"github.com/stretchr/testify/assert",
		"github.com/stretchr/testify/require",
		"github.com/stretchr/testify/mock",
		"go.uber.org/mock/mockgen",
	}

	var missingDeps []string

	for _, dep := range deps {
		args := []string{"list", dep}
		result := tr.runGoCommand(args)
		if result.Error != nil {
			missingDeps = append(missingDeps, dep)
		}
	}

	if len(missingDeps) > 0 {
		return &TestResult{
			Error:  fmt.Errorf("missing test dependencies: %v", missingDeps),
			Output: strings.Join(missingDeps, "\n"),
		}
	}

	return &TestResult{
		Output: "All test dependencies are available",
	}
}

// CleanTestArtifacts cleans up test artifacts
func (tr *TestRunner) CleanTestArtifacts() error {
	artifacts := []string{
		"coverage.out",
		"coverage.html",
		"coverage_integration.out",
		"coverage_e2e.out",
		"*.test",
		"*.prof",
	}

	for _, artifact := range artifacts {
		files, err := filepath.Glob(artifact)
		if err != nil {
			continue
		}
		for _, file := range files {
			if err := os.Remove(file); err != nil {
				fmt.Printf("Warning: failed to remove %s: %v\n", file, err)
			}
		}
	}

	return nil
}
