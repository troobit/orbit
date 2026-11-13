package transcript

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedTZif returns a version-1 TZif file for a zone with one fixed offset.
func fixedTZif(abbrev string, offsetSeconds int32) []byte {
	data := append([]byte("TZif"), make([]byte, 16)...) // version 0 and reserved bytes
	// isutcnt, isstdcnt, leapcnt, timecnt, typecnt, charcnt
	for _, n := range []uint32{0, 0, 0, 0, 1, uint32(len(abbrev) + 1)} {
		data = binary.BigEndian.AppendUint32(data, n)
	}
	data = binary.BigEndian.AppendUint32(data, uint32(offsetSeconds))
	data = append(data, 0, 0) // isdst, designation index
	return append(data, abbrev+"\x00"...)
}

func writeFixedZone(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixed-zone")
	require.NoError(t, os.WriteFile(path, fixedTZif("FIX", 5*3600+30*60), 0o600))
	return path
}

// assertZoneOffset checks that z loaded and applies wantOffset at a fixed instant.
func assertZoneOffset(t *testing.T, z zoneResult, wantOffset time.Duration) {
	t.Helper()
	require.Empty(t, z.warning)
	require.NotNil(t, z.loc)
	reference := time.Date(2026, time.January, 15, 12, 0, 0, 0, time.UTC)
	_, offset := reference.In(z.loc).Zone()
	assert.Equal(t, wantOffset, time.Duration(offset)*time.Second)
}

func TestResolveZone(t *testing.T) {
	zonePath := writeFixedZone(t)
	notZone := filepath.Join(t.TempDir(), "not-a-zone")
	require.NoError(t, os.WriteFile(notZone, []byte("not a TZif file"), 0o600))

	loaded := map[string]struct {
		tz         string
		wantLabel  string
		wantOffset time.Duration
	}{
		"explicitly empty selects UTC": {tz: "", wantLabel: "UTC"},
		"UTC name":                     {tz: "UTC", wantLabel: "UTC"},
		"colon-prefixed UTC":           {tz: ":UTC", wantLabel: "UTC"},
		"IANA name":                    {tz: "Australia/Melbourne", wantLabel: "Australia/Melbourne", wantOffset: 11 * time.Hour},
		"colon-prefixed IANA name":     {tz: ":America/New_York", wantLabel: "America/New_York", wantOffset: -5 * time.Hour},
		"absolute TZif path":           {tz: zonePath, wantLabel: zonePath, wantOffset: 5*time.Hour + 30*time.Minute},
		"colon-prefixed TZif path":     {tz: ":" + zonePath, wantLabel: zonePath, wantOffset: 5*time.Hour + 30*time.Minute},
	}
	for name, tc := range loaded {
		t.Run(name, func(t *testing.T) {
			got := resolveZone(tc.tz, true)
			assertZoneOffset(t, got, tc.wantOffset)
			assert.Equal(t, tc.wantLabel, got.label)
		})
	}

	failed := map[string]string{
		"unknown IANA name":     "Not/AZone",
		"colon without a zone":  ":",
		"Local is not a zone":   "Local",
		"missing TZif path":     filepath.Join(t.TempDir(), "missing"),
		"directory path":        t.TempDir(),
		"file that is not TZif": notZone,
	}
	for name, tz := range failed {
		t.Run(name, func(t *testing.T) {
			got := resolveZone(tz, true)
			assert.Nil(t, got.loc, "a failed override must not select another zone")
			assert.Equal(t, tz, got.label)
			assert.Contains(t, got.warning, `TZ="`)
		})
	}

	t.Run("unset selects the host zone", func(t *testing.T) {
		got := resolveZone("Asia/Tokyo", false)
		assert.Same(t, time.Local, got.loc)
		assert.Empty(t, got.warning)
	})
}

func TestZoneFromEnvironment(t *testing.T) {
	t.Run("unset", func(t *testing.T) {
		t.Setenv("TZ", "Asia/Tokyo") // restores the original value on cleanup
		require.NoError(t, os.Unsetenv("TZ"))
		assert.Same(t, time.Local, zoneFromEnvironment().loc)
	})
	t.Run("explicitly empty", func(t *testing.T) {
		t.Setenv("TZ", "")
		assert.Same(t, time.UTC, zoneFromEnvironment().loc)
	})
	t.Run("IANA name", func(t *testing.T) {
		t.Setenv("TZ", "Asia/Tokyo")
		assertZoneOffset(t, zoneFromEnvironment(), 9*time.Hour)
	})
	t.Run("invalid", func(t *testing.T) {
		t.Setenv("TZ", "Not/AZone")
		got := zoneFromEnvironment()
		assert.Nil(t, got.loc)
		assert.NotEmpty(t, got.warning)
	})
}

func TestZoneInterpret(t *testing.T) {
	newYork := resolveZone("America/New_York", true)
	fixed := resolveZone(writeFixedZone(t), true)

	tests := map[string]struct {
		zone zoneResult
		wall string
		want string // RFC 3339; "" = no unique interpretation
	}{
		"UTC":                            {zone: resolveZone("", true), wall: "2026-01-15 10:00:00", want: "2026-01-15T10:00:00Z"},
		"standard time":                  {zone: newYork, wall: "2026-01-15 10:00:00", want: "2026-01-15T10:00:00-05:00"},
		"daylight time":                  {zone: newYork, wall: "2026-07-15 10:00:00", want: "2026-07-15T10:00:00-04:00"},
		"fixed-offset TZif file":         {zone: fixed, wall: "2026-01-15 10:00:00", want: "2026-01-15T10:00:00+05:30"},
		"last second before the gap":     {zone: newYork, wall: "2026-03-08 01:59:59", want: "2026-03-08T01:59:59-05:00"},
		"nonexistent in the gap":         {zone: newYork, wall: "2026-03-08 02:30:00"},
		"first second after the gap":     {zone: newYork, wall: "2026-03-08 03:00:00", want: "2026-03-08T03:00:00-04:00"},
		"ambiguous in the repeated hour": {zone: newYork, wall: "2026-11-01 01:30:00"},
		"after the repeated hour":        {zone: newYork, wall: "2026-11-01 02:30:00", want: "2026-11-01T02:30:00-05:00"},
		"invalid calendar date":          {zone: newYork, wall: "2026-02-30 10:00:00"},
		"not a wall time":                {zone: newYork, wall: "yesterday"},
		"failed zone override":           {zone: resolveZone("Not/AZone", true), wall: "2026-01-15 10:00:00"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := tc.zone.interpret(tc.wall)
			if tc.want == "" {
				assert.False(t, ok)
				assert.True(t, got.IsZero())
				return
			}
			require.True(t, ok)
			assert.Equal(t, tc.want, got.Format(time.RFC3339))
		})
	}
}
