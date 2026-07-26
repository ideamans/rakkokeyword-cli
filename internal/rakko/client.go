// Package rakkokeyword is a client for the ラッコキーワード (rakkokeyword) API.
//
// Every endpoint answers with the same envelope — result / meta / data /
// errors — so the client keeps the response body verbatim and only decodes the
// envelope. Commands hand the raw bytes to the output package, which means
// `-f json` prints exactly what the API returned and no field is ever lost to
// an out-of-date Go struct.
//
// See https://api.rakkokeyword.com/api-docs.json
package rakko

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is the production API.
const DefaultBaseURL = "https://api.rakkokeyword.com"

// Client calls the rakkokeyword API with an API key.
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client

	// Retries is the number of extra attempts made for a rate limit (429) or a
	// server error (5xx). Zero disables retrying.
	Retries int

	// Sleep is swapped out in tests so backoff does not really wait.
	Sleep func(time.Duration)
}

// New returns a Client with sane defaults.
func New(apiKey, baseURL string, timeout time.Duration, retries int) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &Client{
		APIKey:  apiKey,
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTP:    &http.Client{Timeout: timeout},
		Retries: retries,
		Sleep:   time.Sleep,
	}
}

// Meta is the meta block every response carries.
type Meta struct {
	ConsumedCredit float64 `json:"consumedCredit"`
}

// Response is one API response: the decoded envelope plus the untouched body.
type Response struct {
	Status int
	Raw    []byte
	Result bool            `json:"result"`
	Meta   Meta            `json:"meta"`
	Data   json.RawMessage `json:"data"`
	Errors []string        `json:"errors"`
}

// APIError is a non-2xx response. The API reports the human-readable reason in
// the errors array of the same envelope used by successful calls.
type APIError struct {
	Status  int
	Errors  []string
	Raw     []byte
	Message string
}

func (e *APIError) Error() string {
	detail := strings.Join(e.Errors, "; ")
	if detail == "" {
		detail = strings.TrimSpace(string(e.Raw))
		if len(detail) > 300 {
			detail = detail[:300] + "…"
		}
	}
	if e.Message != "" {
		if detail == "" {
			return fmt.Sprintf("rakkokeyword api: HTTP %d: %s", e.Status, e.Message)
		}
		return fmt.Sprintf("rakkokeyword api: HTTP %d: %s: %s", e.Status, e.Message, detail)
	}
	return fmt.Sprintf("rakkokeyword api: HTTP %d: %s", e.Status, detail)
}

// statusMessage explains a status code in the terms the API documents, so the
// user is told what to do rather than just what broke.
func statusMessage(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "validation error — check the parameters against `rakkokeyword llm`"
	case http.StatusPaymentRequired:
		return "insufficient credit on the rakkokeyword account"
	case http.StatusUnauthorized, http.StatusForbidden:
		return "authentication failed — check the API key (`rakkokeyword auth status`)"
	case http.StatusNotFound:
		return "not found — check the path or requestId"
	case http.StatusTooManyRequests:
		return "rate limit exceeded — retry later or space the calls out"
	case http.StatusServiceUnavailable:
		return "service unavailable"
	}
	if status >= 500 {
		return "server error"
	}
	return ""
}

// Request is one API call.
type Request struct {
	Method string
	Path   string
	Query  url.Values
	Body   any // marshalled to JSON when non-nil

	// NoAuth marks the endpoints that need no API key (the metadata lists), so
	// they still work before anyone has configured one.
	NoAuth bool
}

// URL returns the absolute URL the request would be sent to.
func (c *Client) URL(r Request) string {
	u := c.BaseURL + r.Path
	if len(r.Query) > 0 {
		u += "?" + r.Query.Encode()
	}
	return u
}

// retriable reports whether another attempt has any chance of a better outcome.
func retriable(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

// Do performs the request, retrying rate limits and server errors with
// exponential backoff.
func (c *Client) Do(ctx context.Context, r Request) (*Response, error) {
	if c.APIKey == "" && !r.NoAuth {
		return nil, fmt.Errorf("no API key: set RAKKOKEYWORD_API_KEY, pass --api-key, or run `rakkokeyword auth set-api-key <key>`")
	}

	var payload []byte
	if r.Body != nil {
		var err error
		payload, err = json.Marshal(r.Body)
		if err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
	}

	attempts := c.Retries + 1
	var lastErr error
	var hint time.Duration // Retry-After from the previous attempt
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			if c.Sleep != nil {
				c.Sleep(backoff(attempt, hint))
			}
		}

		var body io.Reader
		if payload != nil {
			body = bytes.NewReader(payload)
		}
		req, err := http.NewRequestWithContext(ctx, r.Method, c.URL(r), body)
		if err != nil {
			return nil, err
		}
		if c.APIKey != "" {
			req.Header.Set("X-API-Key", c.APIKey)
		}
		req.Header.Set("Accept", "application/json")
		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.HTTP.Do(req)
		if err != nil {
			// A transport failure is worth one more try; a cancelled context is not.
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			lastErr = fmt.Errorf("rakkokeyword api: request failed: %w", err)
			hint = 0
			continue
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("rakkokeyword api: read response: %w", readErr)
			hint = 0
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			apiErr := &APIError{Status: resp.StatusCode, Raw: data, Message: statusMessage(resp.StatusCode)}
			var env struct {
				Errors []string `json:"errors"`
			}
			if json.Unmarshal(data, &env) == nil {
				apiErr.Errors = env.Errors
			}
			if retriable(resp.StatusCode) && attempt < attempts-1 {
				lastErr = apiErr
				hint = retryAfter(resp.Header.Get("Retry-After"))
				continue
			}
			return nil, apiErr
		}

		out := &Response{Status: resp.StatusCode, Raw: data}
		if err := json.Unmarshal(data, out); err != nil {
			return nil, fmt.Errorf("rakkokeyword api: decode response: %w", err)
		}
		return out, nil
	}
	return nil, lastErr
}

func retryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}
	if secs, err := strconv.Atoi(strings.TrimSpace(header)); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}

// backoff returns the wait before attempt n (1-based): 1s, 2s, 4s … capped at
// 30s, or the server's Retry-After when it gave one.
func backoff(attempt int, hint time.Duration) time.Duration {
	if hint > 0 {
		return hint
	}
	d := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}
