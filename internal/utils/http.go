package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/company/smartticket/internal/errors"
)

// HTTP utilities

// HTTPClient wraps http.Client with additional functionality
type HTTPClient struct {
	client  *http.Client
	baseURL string
	headers map[string]string
	timeout time.Duration
}

// NewHTTPClient creates a new HTTP client with default settings
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		headers: make(map[string]string),
		timeout: 30 * time.Second,
	}
}

// NewHTTPClientWithTimeout creates a new HTTP client with custom timeout
func NewHTTPClientWithTimeout(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
		},
		headers: make(map[string]string),
		timeout: timeout,
	}
}

// SetBaseURL sets the base URL for all requests
func (c *HTTPClient) SetBaseURL(baseURL string) *HTTPClient {
	c.baseURL = strings.TrimSuffix(baseURL, "/")
	return c
}

// SetHeader sets a default header for all requests
func (c *HTTPClient) SetHeader(key, value string) *HTTPClient {
	c.headers[key] = value
	return c
}

// SetHeaders sets multiple default headers
func (c *HTTPClient) SetHeaders(headers map[string]string) *HTTPClient {
	for key, value := range headers {
		c.headers[key] = value
	}
	return c
}

// SetTimeout sets the timeout for requests
func (c *HTTPClient) SetTimeout(timeout time.Duration) *HTTPClient {
	c.timeout = timeout
	c.client.Timeout = timeout
	return c
}

// SetBearerToken sets authorization header with bearer token
func (c *HTTPClient) SetBearerToken(token string) *HTTPClient {
	c.headers["Authorization"] = "Bearer " + token
	return c
}

// SetBasicAuth sets basic authentication
func (c *HTTPClient) SetBasicAuth(username, password string) *HTTPClient {
	c.headers["Authorization"] = "Basic " + Base64Encode([]byte(username+":"+password))
	return c
}

// RequestOptions contains options for HTTP requests
type RequestOptions struct {
	Headers    map[string]string
	Query      map[string]string
	Body       interface{}
	BodyType   string // "json", "form", "raw"
	Context    context.Context
	RetryCount int
	RetryDelay time.Duration
}

// HTTPResponse contains the response from HTTP requests
type HTTPResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Request    *http.Request
}

// Get performs a GET request
func (c *HTTPClient) Get(url string, options *RequestOptions) (*HTTPResponse, error) {
	return c.doRequest("GET", url, options)
}

// Post performs a POST request
func (c *HTTPClient) Post(url string, options *RequestOptions) (*HTTPResponse, error) {
	return c.doRequest("POST", url, options)
}

// Put performs a PUT request
func (c *HTTPClient) Put(url string, options *RequestOptions) (*HTTPResponse, error) {
	return c.doRequest("PUT", url, options)
}

// Patch performs a PATCH request
func (c *HTTPClient) Patch(url string, options *RequestOptions) (*HTTPResponse, error) {
	return c.doRequest("PATCH", url, options)
}

// Delete performs a DELETE request
func (c *HTTPClient) Delete(url string, options *RequestOptions) (*HTTPResponse, error) {
	return c.doRequest("DELETE", url, options)
}

// doRequest performs the actual HTTP request
func (c *HTTPClient) doRequest(method, rawURL string, options *RequestOptions) (*HTTPResponse, error) {
	// Prepare URL
	fullURL := c.buildURL(rawURL)
	if options != nil && options.Query != nil {
		query := url.Values{}
		for key, value := range options.Query {
			query.Add(key, value)
		}
		fullURL += "?" + query.Encode()
	}

	// Prepare body
	var bodyReader io.Reader
	var contentType string

	if options != nil && options.Body != nil {
		switch options.BodyType {
		case "json":
			jsonData, err := json.Marshal(options.Body)
			if err != nil {
				return nil, errors.NewInternalError("Failed to marshal JSON body", err)
			}
			bodyReader = bytes.NewReader(jsonData)
			contentType = "application/json"
		case "form":
			values := url.Values{}
			if bodyMap, ok := options.Body.(map[string]interface{}); ok {
				for key, value := range bodyMap {
					values.Add(key, ToString(value))
				}
			}
			bodyReader = strings.NewReader(values.Encode())
			contentType = "application/x-www-form-urlencoded"
		default:
			if bodyStr, ok := options.Body.(string); ok {
				bodyReader = strings.NewReader(bodyStr)
			} else if bodyBytes, ok := options.Body.([]byte); ok {
				bodyReader = bytes.NewReader(bodyBytes)
			}
		}
	}

	// Create request
	req, err := http.NewRequest(method, fullURL, bodyReader)
	if err != nil {
		return nil, errors.NewInternalError("Failed to create HTTP request", err)
	}

	// Set context
	if options != nil && options.Context != nil {
		req = req.WithContext(options.Context)
	}

	// Set headers
	for key, value := range c.headers {
		req.Header.Set(key, value)
	}
	if options != nil && options.Headers != nil {
		for key, value := range options.Headers {
			req.Header.Set(key, value)
		}
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	// Set user agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "SmartTicket-HTTPClient/1.0")
	}

	// Perform request with retry logic
	retryCount := 1
	if options != nil && options.RetryCount > 0 {
		retryCount = options.RetryCount
	}

	var lastErr error
	for attempt := 0; attempt < retryCount; attempt++ {
		if attempt > 0 {
			// Wait before retry
			retryDelay := time.Second
			if options != nil && options.RetryDelay > 0 {
				retryDelay = options.RetryDelay
			}
			time.Sleep(retryDelay)
		}

		resp, err := c.client.Do(req)
		if err != nil {
			lastErr = err
			if !isRetryableError(err) {
				break
			}
			continue
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		func() { _ = resp.Body.Close() }()

		if err != nil {
			return nil, errors.NewInternalError("Failed to read response body", err)
		}

		return &HTTPResponse{
			StatusCode: resp.StatusCode,
			Headers:    resp.Header,
			Body:       body,
			Request:    req,
		}, nil
	}

	return nil, errors.NewExternalServiceError("HTTP request failed after retries", lastErr)
}

// buildURL builds the full URL
func (c *HTTPClient) buildURL(rawURL string) string {
	if c.baseURL == "" || strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}
	return c.baseURL + "/" + strings.TrimPrefix(rawURL, "/")
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Add logic to determine if error is retryable
	// For now, return false for all errors
	return false
}

// JSONResponse performs a request and returns JSON response
func (c *HTTPClient) JSONResponse(method, url string, options *RequestOptions, result interface{}) error {
	resp, err := c.doRequest(method, url, options)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return errors.NewExternalServiceError("HTTP request failed", fmt.Errorf("status: %d, body: %s", resp.StatusCode, string(resp.Body)))
	}

	if result != nil {
		if err := json.Unmarshal(resp.Body, result); err != nil {
			return errors.NewInternalError("Failed to unmarshal JSON response", err)
		}
	}

	return nil
}

// GetJSON performs a GET request and returns JSON response
func (c *HTTPClient) GetJSON(url string, options *RequestOptions, result interface{}) error {
	return c.JSONResponse("GET", url, options, result)
}

// PostJSON performs a POST request and returns JSON response
func (c *HTTPClient) PostJSON(url string, body interface{}, result interface{}) error {
	options := &RequestOptions{
		Body:     body,
		BodyType: "json",
	}
	return c.JSONResponse("POST", url, options, result)
}

// PutJSON performs a PUT request and returns JSON response
func (c *HTTPClient) PutJSON(url string, body interface{}, result interface{}) error {
	options := &RequestOptions{
		Body:     body,
		BodyType: "json",
	}
	return c.JSONResponse("PUT", url, options, result)
}

// IsSuccess checks if HTTP status code indicates success
func IsSuccess(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

// IsClientError checks if HTTP status code indicates client error
func IsClientError(statusCode int) bool {
	return statusCode >= 400 && statusCode < 500
}

// IsServerError checks if HTTP status code indicates server error
func IsServerError(statusCode int) bool {
	return statusCode >= 500 && statusCode < 600
}

// GetStatusText returns human-readable status text
func GetStatusText(statusCode int) string {
	switch {
	case IsSuccess(statusCode):
		switch statusCode {
		case 200:
			return "OK"
		case 201:
			return "Created"
		case 202:
			return "Accepted"
		case 204:
			return "No Content"
		default:
			return "Success"
		}
	case IsClientError(statusCode):
		switch statusCode {
		case 400:
			return "Bad Request"
		case 401:
			return "Unauthorized"
		case 403:
			return "Forbidden"
		case 404:
			return "Not Found"
		case 405:
			return "Method Not Allowed"
		case 409:
			return "Conflict"
		case 422:
			return "Unprocessable Entity"
		case 429:
			return "Too Many Requests"
		default:
			return "Client Error"
		}
	case IsServerError(statusCode):
		switch statusCode {
		case 500:
			return "Internal Server Error"
		case 502:
			return "Bad Gateway"
		case 503:
			return "Service Unavailable"
		case 504:
			return "Gateway Timeout"
		default:
			return "Server Error"
		}
	default:
		return "Unknown Status"
	}
}

// ParseURL parses a URL and returns its components
func ParseURL(rawURL string) (*url.URL, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.NewValidationError("Invalid URL").WithCause(err).WithDetails(fmt.Sprintf("URL: %s", rawURL))
	}
	return parsed, nil
}

// BuildURL constructs a URL from components
func BuildURL(base string, path string, params map[string]string) string {
	baseURL := strings.TrimSuffix(base, "/")
	pathURL := strings.TrimPrefix(path, "/")

	resultURL := baseURL + "/" + pathURL

	if params != nil && len(params) > 0 {
		values := url.Values{}
		for key, value := range params {
			values.Add(key, value)
		}
		resultURL += "?" + values.Encode()
	}

	return resultURL
}

// IsValidURL checks if a string is a valid URL
func IsValidURL(rawURL string) bool {
	_, err := url.ParseRequestURI(rawURL)
	return err == nil
}

// GetDomain extracts domain from URL
func GetDomain(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.NewValidationError("Invalid URL").WithCause(err)
	}
	return parsed.Hostname(), nil
}

// GetURLPath extracts path from URL
func GetURLPath(rawURL string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.NewValidationError("Invalid URL").WithCause(err)
	}
	return parsed.Path, nil
}

// AddQueryParam adds a query parameter to URL
func AddQueryParam(rawURL, key, value string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.NewValidationError("Invalid URL").WithCause(err)
	}

	values := parsed.Query()
	values.Add(key, value)
	parsed.RawQuery = values.Encode()

	return parsed.String(), nil
}

// RemoveQueryParam removes a query parameter from URL
func RemoveQueryParam(rawURL, key string) (string, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.NewValidationError("Invalid URL").WithCause(err)
	}

	values := parsed.Query()
	values.Del(key)
	parsed.RawQuery = values.Encode()

	return parsed.String(), nil
}

// EncodeQuery encodes query parameters
func EncodeQuery(params map[string]string) string {
	values := url.Values{}
	for key, value := range params {
		values.Add(key, value)
	}
	return values.Encode()
}

// DecodeQuery decodes query parameters
func DecodeQuery(query string) (map[string]string, error) {
	values, err := url.ParseQuery(query)
	if err != nil {
		return nil, errors.NewValidationError("Invalid query string").WithCause(err)
	}

	result := make(map[string]string)
	for key, value := range values {
		if len(value) > 0 {
			result[key] = value[0]
		}
	}

	return result, nil
}

// Cookie utilities

// Cookie represents an HTTP cookie
type Cookie struct {
	Name     string
	Value    string
	Domain   string
	Path     string
	Expires  time.Time
	MaxAge   int
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

// ParseCookies parses cookies from HTTP response headers
func ParseCookies(headers http.Header) []*Cookie {
	var cookies []*Cookie

	for _, cookieHeader := range headers["Set-Cookie"] {
		if parsedCookie := parseCookieHeader(cookieHeader); parsedCookie != nil {
			cookies = append(cookies, parsedCookie)
		}
	}

	return cookies
}

// parseCookieHeader parses a single cookie header
func parseCookieHeader(header string) *Cookie {
	parts := strings.Split(header, ";")
	if len(parts) == 0 {
		return nil
	}

	// Parse name=value
	nameValue := strings.TrimSpace(parts[0])
	name, value, found := strings.Cut(nameValue, "=")
	if !found {
		return nil
	}

	cookie := &Cookie{
		Name:  strings.TrimSpace(name),
		Value: strings.TrimSpace(value),
		Path:  "/",
	}

	// Parse attributes
	for _, part := range parts[1:] {
		attr := strings.TrimSpace(part)
		if strings.HasPrefix(attr, "Domain=") {
			cookie.Domain = strings.TrimPrefix(attr, "Domain=")
		} else if strings.HasPrefix(attr, "Path=") {
			cookie.Path = strings.TrimPrefix(attr, "Path=")
		} else if strings.HasPrefix(attr, "Max-Age=") {
			if age, err := strconv.Atoi(strings.TrimPrefix(attr, "Max-Age=")); err == nil {
				cookie.MaxAge = age
			}
		} else if attr == "Secure" {
			cookie.Secure = true
		} else if attr == "HttpOnly" {
			cookie.HttpOnly = true
		} else if attr == "SameSite=Strict" {
			cookie.SameSite = http.SameSiteStrictMode
		} else if attr == "SameSite=Lax" {
			cookie.SameSite = http.SameSiteLaxMode
		} else if attr == "SameSite=None" {
			cookie.SameSite = http.SameSiteNoneMode
		} else if strings.HasPrefix(attr, "Expires=") {
			if expires, err := time.Parse(time.RFC1123, strings.TrimPrefix(attr, "Expires=")); err == nil {
				cookie.Expires = expires
			}
		}
	}

	return cookie
}

// CookiesToHeader converts cookies to HTTP header format
func CookiesToHeader(cookies []*Cookie) string {
	var headers []string

	for _, cookie := range cookies {
		header := fmt.Sprintf("%s=%s", cookie.Name, cookie.Value)

		if cookie.Domain != "" {
			header += "; Domain=" + cookie.Domain
		}

		if cookie.Path != "" && cookie.Path != "/" {
			header += "; Path=" + cookie.Path
		}

		if !cookie.Expires.IsZero() {
			header += "; Expires=" + cookie.Expires.Format(time.RFC1123)
		}

		if cookie.MaxAge != 0 {
			header += fmt.Sprintf("; Max-Age=%d", cookie.MaxAge)
		}

		if cookie.Secure {
			header += "; Secure"
		}

		if cookie.HttpOnly {
			header += "; HttpOnly"
		}

		switch cookie.SameSite {
		case http.SameSiteStrictMode:
			header += "; SameSite=Strict"
		case http.SameSiteLaxMode:
			header += "; SameSite=Lax"
		case http.SameSiteNoneMode:
			header += "; SameSite=None"
		}

		headers = append(headers, header)
	}

	return strings.Join(headers, ", ")
}
