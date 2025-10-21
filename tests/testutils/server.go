package testutils

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/company/smartticket/internal/config"
	"github.com/company/smartticket/internal/database"
	"github.com/gin-gonic/gin"
)

// TestServer provides a test HTTP server.
type TestServer struct {
	server     *httptest.Server
	engine     *gin.Engine
	config     *config.Config
	database   *database.Database
	testConfig *TestConfig
	testDB     *TestDatabase
}

// NewTestServer creates a new test HTTP server.
func NewTestServer(t *testing.T) *TestServer {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test configuration
	tc := NewTestConfig(t)

	// Create test database
	td := NewTestDatabase(t)

	// Create Gin engine
	engine := gin.New()

	// Add middleware
	engine.Use(gin.Recovery())
	engine.Use(func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.Next()
	})

	// Create test server
	testServer := httptest.NewServer(engine)

	return &TestServer{
		server:     testServer,
		engine:     engine,
		config:     tc.Config,
		database:   td.Database,
		testConfig: tc,
		testDB:     td,
	}
}

// Close closes the test server and cleans up resources.
func (ts *TestServer) Close() error {
	ts.server.Close()

	if err := ts.database.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	if err := ts.testConfig.Close(); err != nil {
		return fmt.Errorf("failed to close test config: %w", err)
	}

	return nil
}

// GetURL returns the test server URL.
func (ts *TestServer) GetURL() string {
	return ts.server.URL
}

// GetEngine returns the Gin engine.
func (ts *TestServer) GetEngine() *gin.Engine {
	return ts.engine
}

// GetConfig returns the test configuration.
func (ts *TestServer) GetConfig() *config.Config {
	return ts.config
}

// GetDatabase returns the test database.
func (ts *TestServer) GetDatabase() *database.Database {
	return ts.database
}

// WithTestServer is a helper function that runs a test function with a test server.
func WithTestServer(t *testing.T, testFunc func(*testing.T, *TestServer)) {
	ts := NewTestServer(t)
	defer func() {
		if err := ts.Close(); err != nil {
			t.Errorf("Failed to close test server: %v", err)
		}
	}()

	testFunc(t, ts)
}

// HTTPClient provides an HTTP client for testing.
type HTTPClient struct {
	client  *http.Client
	baseURL string
}

// NewHTTPClient creates a new HTTP client for testing.
func NewHTTPClient(server *TestServer) *HTTPClient {
	return &HTTPClient{
		client:  server.server.Client(),
		baseURL: server.GetURL(),
	}
}

// Get performs an HTTP GET request.
func (c *HTTPClient) Get(path string) (*http.Response, error) {
	req, err := http.NewRequest("GET", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	return c.client.Do(req)
}

// Post performs an HTTP POST request.
func (c *HTTPClient) Post(path string, contentType string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("POST", c.baseURL+path, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	return c.client.Do(req)
}

// GetHTTPClient returns the underlying HTTP client for custom requests.
func (c *HTTPClient) GetHTTPClient() *http.Client {
	return c.client
}

// Put performs an HTTP PUT request.
func (c *HTTPClient) Put(path string, contentType string, body []byte) (*http.Response, error) {
	req, err := http.NewRequest("PUT", c.baseURL+path, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)
	return c.client.Do(req)
}

// Delete performs an HTTP DELETE request.
func (c *HTTPClient) Delete(path string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	return c.client.Do(req)
}

// WaitForServer waits for the server to be ready.
func (ts *TestServer) WaitForServer(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := ts.server.Client()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("server did not become ready within %v", timeout)
		case <-ticker.C:
			resp, err := client.Get(ts.server.URL + "/health")
			if resp != nil {
				defer func() { _ = resp.Body.Close() }()
			}
			if err == nil && resp.StatusCode == http.StatusOK {
				return nil
			}
		}
	}
}
