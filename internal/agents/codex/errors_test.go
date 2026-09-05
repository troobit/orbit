package codex

import (
	"testing"
	"time"

	"github.com/arjenschwarz/orbit/internal/agents"
)

func TestClassifier_Classify_RateLimit(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name   string
		stderr string
		stdout string
	}{
		{"rate limited in stderr", "Error: rate limited", ""},
		{"rate_limit in stderr", "rate_limit error occurred", ""},
		{"429 in stdout", "", "HTTP 429 Too Many Requests"},
		{"too many requests", "too many requests", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Classify(1, tt.stderr, tt.stdout, nil)
			if err.Class != agents.ErrorClassRetryable {
				t.Errorf("Class = %v, want %v", err.Class, agents.ErrorClassRetryable)
			}
			if err.Agent != "codex" {
				t.Errorf("Agent = %q, want %q", err.Agent, "codex")
			}
		})
	}
}

func TestClassifier_Classify_Auth(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name   string
		stderr string
	}{
		{"authentication failed", "Error: authentication failed"},
		{"invalid token", "invalid token provided"},
		{"unauthorized", "401 unauthorized"},
		{"api key", "invalid api key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Classify(1, tt.stderr, "", nil)
			if err.Class != agents.ErrorClassFatal {
				t.Errorf("Class = %v, want %v", err.Class, agents.ErrorClassFatal)
			}
		})
	}
}

func TestClassifier_Classify_SessionInvalid(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name   string
		stderr string
	}{
		{"session not found", "session not found"},
		{"invalid session", "Error: invalid session id"},
		{"session expired", "session expired"},
		{"no session", "no session to resume"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Classify(1, tt.stderr, "", nil)
			if err.Class != agents.ErrorClassSessionInvalid {
				t.Errorf("Class = %v, want %v", err.Class, agents.ErrorClassSessionInvalid)
			}
		})
	}
}

func TestClassifier_Classify_Connection(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name   string
		stderr string
	}{
		{"connection", "connection refused"},
		{"network", "network error"},
		{"timeout", "request timeout"},
		{"dns", "dns lookup failed"},
		{"unreachable", "host unreachable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Classify(1, tt.stderr, "", nil)
			if err.Class != agents.ErrorClassRetryable {
				t.Errorf("Class = %v, want %v", err.Class, agents.ErrorClassRetryable)
			}
		})
	}
}

func TestClassifier_Classify_Overloaded(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name   string
		stderr string
	}{
		{"overloaded", "API is overloaded"},
		{"503", "HTTP 503 error"},
		{"service unavailable", "service unavailable"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Classify(1, tt.stderr, "", nil)
			if err.Class != agents.ErrorClassRetryable {
				t.Errorf("Class = %v, want %v", err.Class, agents.ErrorClassRetryable)
			}
			if err.RetryAfter != 30*time.Second {
				t.Errorf("RetryAfter = %v, want %v", err.RetryAfter, 30*time.Second)
			}
		})
	}
}

func TestClassifier_Classify_Unknown(t *testing.T) {
	c := NewClassifier()

	err := c.Classify(1, "some random error", "", nil)
	if err.Class != agents.ErrorClassUnknown {
		t.Errorf("Class = %v, want %v", err.Class, agents.ErrorClassUnknown)
	}
}

func TestClassifier_Classify_ErrMsgs(t *testing.T) {
	c := NewClassifier()

	// Error messages from JSON output should be checked
	err := c.Classify(1, "", "", []string{"rate limited"})
	if err.Class != agents.ErrorClassRetryable {
		t.Errorf("Class = %v, want %v", err.Class, agents.ErrorClassRetryable)
	}
}

func TestClassifier_ImplementsInterface(t *testing.T) {
	var _ = NewClassifier()
}

func TestClassifier_Classify_UsageLimit(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name   string
		stderr string
		stdout string
	}{
		{
			"bare clock time in stderr",
			"ERROR: You've hit your usage limit. Upgrade to Pro (https://chatgpt.com/explore/pro), visit https://chatgpt.com/codex/settings/usage to purchase more credits or try again at 4:18 PM.",
			"",
		},
		{
			"bare clock time in stdout",
			"",
			"ERROR: You've hit your usage limit. Upgrade to Pro or try again at 11:04 AM.",
		},
		{
			"full date form",
			"ERROR: You've hit your usage limit. Upgrade to Pro (https://chatgpt.com/explore/pro), visit https://chatgpt.com/codex/settings/usage to purchase more credits or try again at Sep 5th, 2099 1:39 AM.",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.Classify(1, tt.stderr, tt.stdout, nil)
			if err.Class != agents.ErrorClassRateLimitWait {
				t.Errorf("Class = %v, want %v", err.Class, agents.ErrorClassRateLimitWait)
			}
			if err.Agent != "codex" {
				t.Errorf("Agent = %q, want %q", err.Agent, "codex")
			}
			if err.RetryAfter <= 0 {
				t.Errorf("RetryAfter = %v, want > 0", err.RetryAfter)
			}
		})
	}
}

func TestClassifier_Classify_UsageLimitNoRetryTime(t *testing.T) {
	c := NewClassifier()

	// When the usage limit message has no parseable retry time, it should
	// fall through to unknown rather than wait for an unknown duration.
	err := c.Classify(1, "You've hit your usage limit.", "", nil)
	if err.Class == agents.ErrorClassRateLimitWait {
		t.Errorf("Class = %v, should not be RateLimitWait when no retry time present", err.Class)
	}
	if err.Class != agents.ErrorClassUnknown {
		t.Errorf("Class = %v, want %v", err.Class, agents.ErrorClassUnknown)
	}
}

func TestParseUsageLimitRetryTime(t *testing.T) {
	// Pin the clock: 2026-09-05 11:47 local time.
	now := time.Date(2026, time.September, 5, 11, 47, 0, 0, time.UTC)

	tests := []struct {
		name     string
		msg      string
		expected time.Duration
	}{
		{
			"bare clock time later today",
			"or try again at 4:18 PM.",
			4*time.Hour + 31*time.Minute + time.Minute,
		},
		{
			"bare clock time already passed rolls to tomorrow",
			"or try again at 9:00 AM.",
			21*time.Hour + 13*time.Minute + time.Minute,
		},
		{
			"full date with ordinal day",
			"or try again at Sep 6th, 2026 1:39 AM.",
			13*time.Hour + 52*time.Minute + time.Minute,
		},
		{
			"full month name",
			"or try again at September 6, 2026 1:39 AM.",
			13*time.Hour + 52*time.Minute + time.Minute,
		},
		{
			"explicit date in the past retries promptly",
			"or try again at Sep 5th, 2026 1:39 AM.",
			time.Minute,
		},
		{
			"hour without minutes",
			"or try again at 12 PM.",
			13*time.Minute + time.Minute,
		},
		{
			"no retry time",
			"You've hit your usage limit.",
			0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseUsageLimitRetryTime(tt.msg, now)
			if got != tt.expected {
				t.Errorf("parseUsageLimitRetryTime(%q) = %v, want %v", tt.msg, got, tt.expected)
			}
		})
	}
}
