package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/ideamans/rakkokeyword-cli/internal/config"
	"github.com/ideamans/rakkokeyword-cli/internal/rakko"
)

// The two batch endpoints — search-volume and search-rank — are asynchronous:
// register returns a requestId, the job runs in the background, and the results
// are fetched once status reports isCompleted. The helpers here implement the
// register → poll → results chain that --wait performs.

// waitFlags are the polling flags shared by the batch commands.
type waitFlags struct {
	wait     bool
	interval time.Duration
	timeout  time.Duration
}

func (w *waitFlags) addTo(cmd *cobra.Command, note string) {
	f := cmd.Flags()
	f.BoolVar(&w.wait, "wait", false, "Poll until the job completes"+note)
	f.DurationVar(&w.interval, "poll-interval", 30*time.Second, "How often to poll while waiting (the API recommends 30s)")
	f.DurationVar(&w.timeout, "wait-timeout", time.Hour, "Give up waiting after this long; the job keeps running and can be fetched later by requestId")
}

// clientFor builds the API client from config and the global flags.
func clientFor() (*rakko.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	return rakko.New(cfg.APIKeyResolved(flagAPIKey), cfg.BaseURLResolved(), flagTimeout, flagRetries), nil
}

// requestIDOf extracts data.requestId from a register response. It is a number
// for search-volume and a string (ULID) for search-rank, so it comes back as a
// string either way.
func requestIDOf(resp *rakko.Response) (string, error) {
	var data struct {
		RequestID json.RawMessage `json:"requestId"`
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return "", fmt.Errorf("register response has no data object: %w", err)
	}
	raw := string(data.RequestID)
	if raw == "" || raw == "null" {
		return "", fmt.Errorf("register response has no requestId")
	}
	var asString string
	if json.Unmarshal(data.RequestID, &asString) == nil {
		return asString, nil
	}
	return raw, nil
}

// pollUntilComplete polls a status endpoint until isCompleted is true.
// Progress goes to stderr so it never pollutes parseable output.
func pollUntilComplete(cmd *cobra.Command, client *rakko.Client, statusPath string, w waitFlags) error {
	if w.interval <= 0 {
		w.interval = 30 * time.Second
	}
	deadline := time.Now().Add(w.timeout)
	stderr := cmd.ErrOrStderr()

	for attempt := 1; ; attempt++ {
		resp, err := client.Do(ctx(cmd), rakko.Request{Method: "GET", Path: statusPath})
		if err != nil {
			return err
		}
		var status struct {
			IsCompleted bool            `json:"isCompleted"`
			Statuses    json.RawMessage `json:"statuses"`
		}
		if err := json.Unmarshal(resp.Data, &status); err != nil {
			return fmt.Errorf("decode status: %w", err)
		}
		if status.IsCompleted {
			fmt.Fprintf(stderr, "job completed after %d status check(s)\n", attempt)
			return nil
		}
		if !flagQuiet {
			fmt.Fprintf(stderr, "waiting… (check %d, statuses %s)\n", attempt, string(status.Statuses))
		}
		if time.Now().Add(w.interval).After(deadline) {
			return fmt.Errorf("still processing after %s — the job keeps running; fetch it later with the requestId above", w.timeout)
		}
		select {
		case <-time.After(w.interval):
		case <-ctx(cmd).Done():
			return ctx(cmd).Err()
		}
	}
}

// historiesQuery builds the shared query string of the two histories endpoints.
func historiesQuery(cmd *cobra.Command, limit, offset int, status string) (url.Values, error) {
	if err := rakko.Enum("status", status, []string{"completed", "processing"}); err != nil {
		return nil, err
	}
	q := queryValues()
	q.setIfInt(cmd, "limit", "limit", limit)
	q.setIfInt(cmd, "offset", "offset", offset)
	q.setIf(cmd, "status", "status", status)
	return q.values, nil
}

// ── query string helper ──────────────────────────────────────────────────────

type queryBuilder struct{ values url.Values }

func queryValues() queryBuilder { return queryBuilder{values: url.Values{}} }

// setIf adds key=value when the flag was given.
func (q queryBuilder) setIf(cmd *cobra.Command, flag, key, value string) {
	if cmd.Flags().Changed(flag) && value != "" {
		q.values.Set(key, value)
	}
}

// setIfInt adds key=value when the flag was given.
func (q queryBuilder) setIfInt(cmd *cobra.Command, flag, key string, value int) {
	if cmd.Flags().Changed(flag) {
		q.values.Set(key, strconv.Itoa(value))
	}
}
