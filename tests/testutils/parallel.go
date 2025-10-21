package testutils

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/company/smartticket/internal/database"
)

// TestDatabaseParallel wraps a database connection for parallel testing.
type TestDatabaseParallel struct {
	DB *database.Database
}

// ParallelTestRunner manages parallel test execution with resource isolation.
type ParallelTestRunner struct {
	maxWorkers    int
	timeout       time.Duration
	setupFunc     func() error
	teardownFunc  func() error
	resourceMutex map[string]*sync.Mutex
	nextWorkerID  int64
	workerIDMutex sync.Mutex
}

// NewParallelTestRunner creates a new parallel test runner.
func NewParallelTestRunner() *ParallelTestRunner {
	maxWorkers := runtime.NumCPU()
	if maxWorkers < 2 {
		maxWorkers = 2
	}

	return &ParallelTestRunner{
		maxWorkers:    maxWorkers,
		timeout:       30 * time.Second,
		resourceMutex: make(map[string]*sync.Mutex),
		nextWorkerID:  1,
	}
}

// SetMaxWorkers sets the maximum number of parallel workers.
func (ptr *ParallelTestRunner) SetMaxWorkers(workers int) {
	if workers < 1 {
		workers = 1
	}
	ptr.maxWorkers = workers
}

// SetTimeout sets the timeout for test execution.
func (ptr *ParallelTestRunner) SetTimeout(timeout time.Duration) {
	ptr.timeout = timeout
}

// SetSetupFunc sets the setup function to run before tests.
func (ptr *ParallelTestRunner) SetSetupFunc(setup func() error) {
	ptr.setupFunc = setup
}

// SetTeardownFunc sets the teardown function to run after tests.
func (ptr *ParallelTestRunner) SetTeardownFunc(teardown func() error) {
	ptr.teardownFunc = teardown
}

// GetResourceMutex returns a mutex for a specific resource name.
func (ptr *ParallelTestRunner) GetResourceMutex(resourceName string) *sync.Mutex {
	ptr.resourceMutex[resourceName] = &sync.Mutex{}
	return ptr.resourceMutex[resourceName]
}

// getNextWorkerID returns the next available worker ID.
func (ptr *ParallelTestRunner) getNextWorkerID() int {
	ptr.workerIDMutex.Lock()
	defer ptr.workerIDMutex.Unlock()

	id := int(ptr.nextWorkerID)
	ptr.nextWorkerID++
	return id
}

// RunParallel executes test functions in parallel.
func (ptr *ParallelTestRunner) RunParallel(t *testing.T, tests []TestFunc) {
	if len(tests) == 0 {
		return
	}

	// Run setup function
	if ptr.setupFunc != nil {
		if err := ptr.setupFunc(); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
		defer func() {
			if ptr.teardownFunc != nil {
				if err := ptr.teardownFunc(); err != nil {
					t.Errorf("Teardown failed: %v", err)
				}
			}
		}()
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), ptr.timeout)
	defer cancel()

	// Create work channel
	workChan := make(chan TestFunc, len(tests))
	for _, test := range tests {
		workChan <- test
	}
	close(workChan)

	// Create worker pool
	var wg sync.WaitGroup
	results := make(chan ParallelTestResult, len(tests))

	for i := 0; i < ptr.maxWorkers && i < len(tests); i++ {
		wg.Add(1)
		go ptr.worker(ctx, &wg, workChan, results)
	}

	// Wait for all workers to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var failedTests []ParallelTestResult
	for result := range results {
		if result.Error != nil {
			failedTests = append(failedTests, result)
		}
	}

	// Report failures
	if len(failedTests) > 0 {
		t.Errorf("%d out of %d tests failed:", len(failedTests), len(tests))
		for _, result := range failedTests {
			t.Errorf("  %s: %v", result.Name, result.Error)
		}
	}
}

// TestFunc represents a test function with metadata.
type TestFunc struct {
	Name        string
	TestFunc    func(*testing.T)
	Description string
	Timeout     time.Duration
	Resources   []string // Resources this test needs exclusive access to
}

// ParallelTestResult represents the result of a parallel test execution.
type ParallelTestResult struct {
	Name     string
	Error    error
	Duration time.Duration
	WorkerID int
}

// worker processes test functions from the work channel.
func (ptr *ParallelTestRunner) worker(ctx context.Context, wg *sync.WaitGroup, workChan <-chan TestFunc, results chan<- ParallelTestResult) {
	defer wg.Done()

	workerID := ptr.getNextWorkerID()

	for testFunc := range workChan {
		select {
		case <-ctx.Done():
			results <- ParallelTestResult{
				Name:     testFunc.Name,
				Error:    ctx.Err(),
				WorkerID: workerID,
			}
			return
		default:
		}

		// Acquire resource locks if needed
		acquiredLocks := make([]*sync.Mutex, 0, len(testFunc.Resources))
		for _, resource := range testFunc.Resources {
			mutex := ptr.GetResourceMutex(resource)
			mutex.Lock()
			acquiredLocks = append(acquiredLocks, mutex)
		}

		start := time.Now()
		result := ptr.runSingleTest(ctx, testFunc, workerID)
		result.Duration = time.Since(start)

		// Release locks
		for _, mutex := range acquiredLocks {
			mutex.Unlock()
		}

		results <- result
	}
}

// runSingleTest runs a single test function.
func (ptr *ParallelTestRunner) runSingleTest(ctx context.Context, testFunc TestFunc, workerID int) ParallelTestResult {
	// Create a test context with timeout
	testCtx, cancel := context.WithTimeout(ctx, testFunc.Timeout)
	defer cancel()

	// Create testing.T for this test
	helper := &testing.T{}
	done := make(chan struct{})
	var testErr error

	// Run test in goroutine
	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				testErr = fmt.Errorf("panic in test %s: %v", testFunc.Name, r)
			}
		}()

		// Create a subtest
		helper.Run(testFunc.Name, func(t *testing.T) {
			testFunc.TestFunc(t)
		})

		// Check if test failed
		if helper.Failed() {
			testErr = fmt.Errorf("test %s failed", testFunc.Name)
		}
	}()

	// Wait for completion or timeout
	select {
	case <-done:
		return ParallelTestResult{
			Name:     testFunc.Name,
			Error:    testErr,
			WorkerID: workerID,
		}
	case <-testCtx.Done():
		return ParallelTestResult{
			Name:     testFunc.Name,
			Error:    fmt.Errorf("test %s timed out", testFunc.Name),
			WorkerID: workerID,
		}
	}
}

// RunParallelWithBatches runs tests in parallel with batch processing.
func (ptr *ParallelTestRunner) RunParallelWithBatches(t *testing.T, tests []TestFunc, batchSize int) {
	if batchSize <= 0 {
		batchSize = len(tests)
	}

	for i := 0; i < len(tests); i += batchSize {
		end := i + batchSize
		if end > len(tests) {
			end = len(tests)
		}

		batch := tests[i:end]
		t.Logf("Running batch %d/%d (%d tests)", (i/batchSize)+1, (len(tests)+batchSize-1)/batchSize, len(batch))
		ptr.RunParallel(t, batch)
	}
}

// DatabaseParallelRunner manages parallel database tests with isolation.
type DatabaseParallelRunner struct {
	*ParallelTestRunner
	testDatabases chan *TestDatabaseParallel
	dbPool        *sync.Pool
}

// NewDatabaseParallelRunner creates a new database parallel runner.
func NewDatabaseParallelRunner(dbFactory func() *TestDatabaseParallel) *DatabaseParallelRunner {
	pr := NewParallelTestRunner()
	testDatabases := make(chan *TestDatabaseParallel, pr.maxWorkers)

	// Create database pool
	dbPool := &sync.Pool{
		New: func() interface{} {
			return dbFactory()
		},
	}

	return &DatabaseParallelRunner{
		ParallelTestRunner: pr,
		testDatabases:      testDatabases,
		dbPool:             dbPool,
	}
}

// GetTestDatabase gets a test database from the pool.
func (dpr *DatabaseParallelRunner) GetTestDatabase() *TestDatabaseParallel {
	db := dpr.dbPool.Get().(*TestDatabaseParallel)
	return db
}

// ReturnTestDatabase returns a test database to the pool.
func (dpr *DatabaseParallelRunner) ReturnTestDatabase(db *TestDatabaseParallel) {
	dpr.dbPool.Put(db)
}

// RunDatabaseTests runs database tests in parallel with isolated databases.
func (dpr *DatabaseParallelRunner) RunDatabaseTests(t *testing.T, tests []DatabaseTestFunc) {
	if len(tests) == 0 {
		return
	}

	// Convert to regular test functions
	regularTests := make([]TestFunc, len(tests))
	for i, test := range tests {
		regularTests[i] = TestFunc{
			Name:        test.Name,
			Description: test.Description,
			Timeout:     test.Timeout,
			TestFunc: func(t *testing.T) {
				// Get test database
				db := dpr.GetTestDatabase()
				defer dpr.ReturnTestDatabase(db)

				// Run database test
				test.TestFunc(t, db)
			},
			Resources: test.Resources,
		}
	}

	dpr.RunParallel(t, regularTests)
}

// DatabaseTestFunc represents a database test function.
type DatabaseTestFunc struct {
	Name        string
	Description string
	Timeout     time.Duration
	TestFunc    func(*testing.T, *TestDatabaseParallel)
	Resources   []string
}

// ConcurrentTestSuite manages a suite of concurrent tests.
type ConcurrentTestSuite struct {
	name           string
	tests          []TestFunc
	setupSuite     func() error
	teardownSuite  func() error
	setupTest      func(*testing.T) error
	teardownTest   func(*testing.T) error
	parallelRunner *ParallelTestRunner
}

// NewConcurrentTestSuite creates a new concurrent test suite.
func NewConcurrentTestSuite(name string) *ConcurrentTestSuite {
	return &ConcurrentTestSuite{
		name:           name,
		tests:          make([]TestFunc, 0),
		parallelRunner: NewParallelTestRunner(),
	}
}

// AddTest adds a test to the suite.
func (cts *ConcurrentTestSuite) AddTest(name, description string, testFunc func(*testing.T)) {
	cts.AddTestWithTimeout(name, description, testFunc, 30*time.Second)
}

// AddTestWithTimeout adds a test with custom timeout.
func (cts *ConcurrentTestSuite) AddTestWithTimeout(name, description string, testFunc func(*testing.T), timeout time.Duration) {
	cts.tests = append(cts.tests, TestFunc{
		Name:        name,
		Description: description,
		TestFunc:    testFunc,
		Timeout:     timeout,
	})
}

// AddTestWithResources adds a test that requires exclusive access to resources.
func (cts *ConcurrentTestSuite) AddTestWithResources(name, description string, testFunc func(*testing.T), resources []string) {
	cts.tests = append(cts.tests, TestFunc{
		Name:        name,
		Description: description,
		TestFunc:    testFunc,
		Timeout:     30 * time.Second,
		Resources:   resources,
	})
}

// SetSetupSuite sets the suite setup function.
func (cts *ConcurrentTestSuite) SetSetupSuite(setup func() error) {
	cts.setupSuite = setup
}

// SetTeardownSuite sets the suite teardown function.
func (cts *ConcurrentTestSuite) SetTeardownSuite(teardown func() error) {
	cts.teardownSuite = teardown
}

// SetSetupTest sets the per-test setup function.
func (cts *ConcurrentTestSuite) SetSetupTest(setup func(*testing.T) error) {
	cts.setupTest = setup
}

// SetTeardownTest sets the per-test teardown function.
func (cts *ConcurrentTestSuite) SetTeardownTest(teardown func(*testing.T) error) {
	cts.teardownTest = teardown
}

// Run runs the test suite.
func (cts *ConcurrentTestSuite) Run(t *testing.T) {
	t.Logf("Running concurrent test suite: %s (%d tests)", cts.name, len(cts.tests))

	if len(cts.tests) == 0 {
		t.Log("No tests to run")
		return
	}

	// Wrap tests with setup/teardown
	wrappedTests := make([]TestFunc, len(cts.tests))
	for i, test := range cts.tests {
		wrappedTests[i] = TestFunc{
			Name:        test.Name,
			Description: test.Description,
			Timeout:     test.Timeout,
			Resources:   test.Resources,
			TestFunc: func(t *testing.T) {
				// Run test setup
				if cts.setupTest != nil {
					if err := cts.setupTest(t); err != nil {
						t.Fatalf("Test setup failed: %v", err)
					}
					defer func() {
						if err := cts.teardownTest(t); err != nil {
							t.Errorf("Test teardown failed: %v", err)
						}
					}()
				}

				// Run actual test
				test.TestFunc(t)
			},
		}
	}

	// Configure parallel runner
	if cts.setupSuite != nil {
		cts.parallelRunner.SetSetupFunc(cts.setupSuite)
	}
	if cts.teardownSuite != nil {
		cts.parallelRunner.SetTeardownFunc(cts.teardownSuite)
	}

	// Run tests in parallel
	cts.parallelRunner.RunParallel(t, wrappedTests)
}

// RunSequential runs tests sequentially (for debugging).
func (cts *ConcurrentTestSuite) RunSequential(t *testing.T) {
	t.Logf("Running test suite sequentially: %s (%d tests)", cts.name, len(cts.tests))

	if len(cts.tests) == 0 {
		t.Log("No tests to run")
		return
	}

	// Run suite setup
	if cts.setupSuite != nil {
		if err := cts.setupSuite(); err != nil {
			t.Fatalf("Suite setup failed: %v", err)
		}
		defer func() {
			if err := cts.teardownSuite(); err != nil {
				t.Errorf("Suite teardown failed: %v", err)
			}
		}()
	}

	// Run tests sequentially
	for _, test := range cts.tests {
		t.Run(test.Name, func(t *testing.T) {
			// Run test setup
			if cts.setupTest != nil {
				if err := cts.setupTest(t); err != nil {
					t.Fatalf("Test setup failed: %v", err)
				}
				defer func() {
					if err := cts.teardownTest(t); err != nil {
						t.Errorf("Test teardown failed: %v", err)
					}
				}()
			}

			// Run test with timeout
			done := make(chan struct{})
			go func() {
				defer close(done)
				test.TestFunc(t)
			}()

			select {
			case <-done:
				// Test completed
			case <-time.After(test.Timeout):
				t.Fatalf("Test timed out after %v", test.Timeout)
			}
		})
	}
}
