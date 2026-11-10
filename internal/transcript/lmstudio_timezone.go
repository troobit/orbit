package transcript

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	_ "time/tzdata" // IANA zones load without a system zone database (Q17)
)

// zoneResult is the zone LM Studio wall times are interpreted in. The log
// records no zone, so every interpretation is an assumption.
type zoneResult struct {
	loc     *time.Location // nil when an explicit override failed to load
	label   string         // what render output calls the zone
	warning string         // "" when resolution succeeded
}

// zoneFromEnvironment resolves the process TZ setting. os.LookupEnv keeps an
// unset TZ distinct from an explicitly empty one, which Go's own Unix
// local-zone initialisation collapses.
func zoneFromEnvironment() zoneResult {
	tz, ok := os.LookupEnv("TZ")
	return resolveZone(tz, ok)
}

// resolveZone selects the zone for a TZ value: the host zone when unset, UTC
// when empty, otherwise an IANA name or an absolute TZif file path, either
// with an optional leading colon. A value that fails to load yields a nil
// location and a warning rather than any substitute zone.
func resolveZone(tzEnv string, tzSet bool) zoneResult {
	if !tzSet {
		return zoneResult{loc: time.Local, label: "host local zone"}
	}
	if tzEnv == "" {
		return zoneResult{loc: time.UTC, label: "UTC"}
	}

	name := strings.TrimPrefix(tzEnv, ":")
	var loc *time.Location
	var err error
	switch {
	case name == "":
		err = errors.New("no zone follows the colon")
	case name == "Local":
		// time.LoadLocation maps "Local" to the host zone, which is not what
		// an explicit override asks for.
		err = errors.New("the name Local is not a zone")
	case filepath.IsAbs(name):
		var data []byte
		if data, err = os.ReadFile(name); err == nil {
			loc, err = time.LoadLocationFromTZData(name, data)
		}
	default:
		loc, err = time.LoadLocation(name)
	}
	if err != nil {
		return zoneResult{
			label:   tzEnv,
			warning: fmt.Sprintf("TZ=%q could not be loaded (%v); wall times are kept without interpreted timestamps", tzEnv, err),
		}
	}
	return zoneResult{loc: loc, label: name}
}

// interpret returns the instant a logged wall time ("2006-01-02 15:04:05")
// names in the zone, carrying the zone's offset. It reports false when the
// zone failed to load or the wall time is invalid, nonexistent or ambiguous
// there, so callers keep only the wall time.
func (z zoneResult) interpret(wall string) (time.Time, bool) {
	if z.loc == nil {
		return time.Time{}, false
	}
	w, err := time.ParseInLocation(time.DateTime, wall, time.UTC)
	if err != nil {
		return time.Time{}, false
	}

	// Any transition affecting this wall time lies within 14 hours of it,
	// so the offsets in force a day either side and at it are the only
	// candidates. Each candidate is valid when it maps back to the same wall
	// time; exactly one distinct valid instant is a unique interpretation.
	want := w.Format(time.DateTime)
	var found time.Time
	count := 0
	for _, probe := range []time.Time{w.Add(-24 * time.Hour), w, w.Add(24 * time.Hour)} {
		_, offset := probe.In(z.loc).Zone()
		candidate := w.Add(-time.Duration(offset) * time.Second).In(z.loc)
		if candidate.Format(time.DateTime) != want || (count > 0 && candidate.Equal(found)) {
			continue
		}
		found = candidate
		count++
	}
	if count != 1 {
		return time.Time{}, false
	}
	return found, true
}
