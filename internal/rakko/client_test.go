package rakko

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	c := New("test-key", server.URL, 5*time.Second, 3)
	c.Sleep = func(time.Duration) {} // no real backoff in tests
	return c
}

func TestDoSendsKeyAndBody(t *testing.T) {
	var gotKey, gotPath, gotBody string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-API-Key")
		gotPath = r.URL.Path
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		_, _ = w.Write([]byte(`{"result":true,"meta":{"consumedCredit":1.5},"data":{"items":[]},"errors":[]}`))
	})

	resp, err := c.Do(context.Background(), Request{
		Method: "POST", Path: "/v1/suggest-keywords", Body: Body{"keyword": "ラッコ"},
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	if gotKey != "test-key" {
		t.Errorf("X-API-Key = %q", gotKey)
	}
	if gotPath != "/v1/suggest-keywords" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"keyword":"ラッコ"`) {
		t.Errorf("body = %q", gotBody)
	}
	if resp.Meta.ConsumedCredit != 1.5 {
		t.Errorf("consumedCredit = %v", resp.Meta.ConsumedCredit)
	}
	if !json.Valid(resp.Raw) {
		t.Error("Raw should hold the response bytes verbatim")
	}
}

func TestDoRetriesRateLimit(t *testing.T) {
	var calls int
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"result":false,"errors":["Rate limit exceeded. Please try again later."]}`))
			return
		}
		_, _ = w.Write([]byte(`{"result":true,"data":{}}`))
	})

	if _, err := c.Do(context.Background(), Request{Method: "GET", Path: "/v1/metadata/languages"}); err != nil {
		t.Fatalf("Do should have succeeded on the third attempt: %v", err)
	}
	if calls != 3 {
		t.Errorf("attempts = %d, want 3", calls)
	}
}

func TestDoDoesNotRetryClientErrors(t *testing.T) {
	var calls int
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"result":false,"errors":["Invalid API key"]}`))
	})

	_, err := c.Do(context.Background(), Request{Method: "GET", Path: "/v1/search-volume/histories"})
	if err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Errorf("a 403 must not be retried; attempts = %d", calls)
	}
	// The message has to say what to do about it, not just the status code.
	if !strings.Contains(err.Error(), "API key") {
		t.Errorf("unhelpful error: %v", err)
	}
	var apiErr *APIError
	if !asAPIError(err, &apiErr) || apiErr.Status != http.StatusForbidden {
		t.Errorf("expected an APIError with status 403, got %#v", err)
	}
}

func TestDoRequiresAPIKeyExceptForMetadata(t *testing.T) {
	c := New("", "https://example.invalid", time.Second, 0)
	if _, err := c.Do(context.Background(), Request{Method: "GET", Path: "/v1/search-volume/histories"}); err == nil {
		t.Fatal("expected a missing-key error")
	} else if !strings.Contains(err.Error(), "RAKKOKEYWORD_API_KEY") {
		t.Errorf("error should name the environment variable: %v", err)
	}

	// The metadata endpoints are public, so an unconfigured user can still
	// look up the region and language names.
	called := false
	pub := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Header.Get("X-API-Key") != "" {
			t.Error("no key should be sent when none is configured")
		}
		_, _ = w.Write([]byte(`{"result":true,"data":{"languages":[]}}`))
	})
	pub.APIKey = ""
	if _, err := pub.Do(context.Background(), Request{Method: "GET", Path: "/v1/metadata/languages", NoAuth: true}); err != nil {
		t.Fatalf("public endpoint should work without a key: %v", err)
	}
	if !called {
		t.Error("request was never sent")
	}
}

func TestBackoffHonoursRetryAfter(t *testing.T) {
	if got := backoff(1, 7*time.Second); got != 7*time.Second {
		t.Errorf("Retry-After should win, got %v", got)
	}
	if got := backoff(1, 0); got != time.Second {
		t.Errorf("first retry = %v, want 1s", got)
	}
	if got := backoff(10, 0); got != 30*time.Second {
		t.Errorf("backoff should cap at 30s, got %v", got)
	}
}

// asAPIError is errors.As without importing errors into every test.
func asAPIError(err error, target **APIError) bool {
	e, ok := err.(*APIError)
	if ok {
		*target = e
	}
	return ok
}
