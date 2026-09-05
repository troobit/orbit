package codex

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/arjenschwarz/orbit/internal/agents"
)

func init() {
	agents.RegisterClassifier("codex", NewClassifier)
}

// Classifier implements agents.ErrorClassifier for Codex.
type Classifier struct{}

// NewClassifier creates a new Codex error classifier.
func NewClassifier() agents.ErrorClassifier {
	return &Classifier{}
}

// Classify examines error output and classifies it by type.
func (c *Classifier) Classify(exitCode int, stderr, stdout string, errMsgs []string) *agents.ClassifiedError {
	// Combine all error sources for pattern matching
	combined := stderr + stdout + strings.Join(errMsgs, " ")
	combinedLower := strings.ToLower(combined)

	// Check for usage limits - must check before regular rate limits.
	// Example messages:
	//   "You've hit your usage limit. Upgrade to Pro (...), visit ... to
	//    purchase more credits or try again at 4:18 PM."
	//   "You've hit your usage limit. ... or try again at Sep 5th, 2026 1:39 AM."
	if strings.Contains(combinedLower, "hit your usage limit") {
		waitDuration := parseUsageLimitRetryTime(combined, time.Now())
		if waitDuration > 0 {
			return agents.NewRateLimitWaitError("codex", waitDuration)
		}
	}

	// Check for rate limiting
	if agents.MatchesRateLimit(combinedLower) {
		return agents.NewRateLimitError("codex", combinedLower)
	}

	// Check for authentication errors (fatal)
	if agents.MatchesAuthError(combinedLower, "authentication failed") {
		return agents.NewAuthError("codex")
	}

	// Check for session-related errors
	if agents.MatchesSessionInvalid(combinedLower, "no session") {
		return agents.NewSessionInvalidError("codex")
	}

	// Check for connection errors (retryable)
	if agents.MatchesConnectionError(combinedLower) {
		return agents.NewConnectionError("codex")
	}

	// Check for API overload (retryable)
	if agents.MatchesOverload(combinedLower) {
		return agents.NewOverloadError("codex")
	}

	return agents.NewUnknownError("codex", errMsgs, stderr, stdout)
}

// usageLimitRetryPattern matches the retry time in Codex usage-limit messages:
// "try again at <time>" where <time> is either a bare clock time ("4:18 PM")
// or a full date ("Sep 5th, 2026 1:39 AM").
var usageLimitRetryPattern = regexp.MustCompile(
	`(?i)try again at\s+(?:([a-z]{3,9})\.?\s+(\d{1,2})(?:st|nd|rd|th)?,?\s+(\d{4})\s+)?(\d{1,2})(?::(\d{2}))?\s*(am|pm)`)

// parseUsageLimitRetryTime parses the retry time from Codex usage-limit
// messages. Codex reports the time in the user's local timezone, so times are
// interpreted in now's location. A bare clock time earlier than now refers to
// tomorrow. Returns the duration to wait, or 0 if no retry time is found.
func parseUsageLimitRetryTime(msg string, now time.Time) time.Duration {
	matches := usageLimitRetryPattern.FindStringSubmatch(msg)
	if matches == nil {
		return 0
	}

	hour, err := strconv.Atoi(matches[4])
	if err != nil {
		return 0
	}
	minute := 0
	if matches[5] != "" {
		minute, err = strconv.Atoi(matches[5])
		if err != nil {
			return 0
		}
	}

	// Convert 12-hour to 24-hour format
	ampm := strings.ToLower(matches[6])
	if ampm == "pm" && hour != 12 {
		hour += 12
	} else if ampm == "am" && hour == 12 {
		hour = 0
	}

	loc := now.Location()

	var resetTime time.Time
	if matches[1] != "" {
		// Full date form: "Sep 5th, 2026 1:39 AM"
		month, ok := parseMonth(matches[1])
		if !ok {
			return 0
		}
		day, err := strconv.Atoi(matches[2])
		if err != nil {
			return 0
		}
		year, err := strconv.Atoi(matches[3])
		if err != nil {
			return 0
		}
		resetTime = time.Date(year, month, day, hour, minute, 0, 0, loc)
	} else {
		// Bare clock time form: "4:18 PM"
		resetTime = time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc)
		if resetTime.Before(now) {
			resetTime = resetTime.Add(24 * time.Hour)
		}
	}

	// Add a small buffer (1 minute) to ensure we're past the reset
	waitDuration := resetTime.Sub(now) + time.Minute
	if waitDuration <= 0 {
		// An explicit date in the past means the limit should already have
		// reset; retry promptly rather than falling through as unknown.
		return time.Minute
	}
	return waitDuration
}

// parseMonth parses a month name in either full ("September") or
// abbreviated ("Sep") form.
func parseMonth(name string) (time.Month, bool) {
	if t, err := time.Parse("January", name); err == nil {
		return t.Month(), true
	}
	if t, err := time.Parse("Jan", name); err == nil {
		return t.Month(), true
	}
	return 0, false
}
