package transcript

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestParseLMStudio_Document(t *testing.T) {
	res := parseLMFixture(t, "threads.log")
	assert.Equal(t, FormatLMStudio, res.Format)
	assert.Equal(t, &ParseResultMetadata{
		StartedAt:     "2026-03-05T14:00:00Z",
		FirstModel:    "example-model-a",
		FirstProvider: lmProvider,
		TimeZone:      "UTC",
	}, res.Metadata)

	type annotation struct {
		thread        string
		prefix        int
		discontinuity bool
	}
	var got []annotation
	ids := map[string]bool{}
	for _, req := range res.LMStudioRequests {
		got = append(got, annotation{req.ThreadID, req.PriorPrefixCount, req.ContextDiscontinuity})
		ids[req.ID] = true
	}
	assert.Equal(t, []annotation{
		{"thread-alpha", 0, false}, // first in its thread
		{"", 0, false},             // singleton, never compared
		{"thread-alpha", 2, false}, // repeats instructions and the user message, keys reordered
		{"thread-alpha", 0, true},  // compaction shortened the history
		{"thread-alpha", 0, true},  // memory request with other input
		{"thread-beta", 0, false},
		{"", 0, false},
		{"", 0, false},
	}, got)

	for _, e := range res.Entries {
		if !e.Unattributed {
			assert.True(t, ids[e.RequestID], "entry joins a request by its key")
		}
	}

	auto, err := Parse(strings.NewReader(readLMFixture(t, "threads.log")))
	require.NoError(t, err)
	assert.Equal(t, res, auto, "auto-detection parses the same document")
}

func TestParseLMStudio_TimeZone(t *testing.T) {
	input := readLMFixture(t, "endpoints.log")

	t.Run("unloadable override", func(t *testing.T) {
		t.Setenv("TZ", "Not/AZone")
		res, err := ParseLMStudio(strings.NewReader(input))
		require.NoError(t, err)
		require.NotEmpty(t, res.Warnings)
		assert.Equal(t, WarningTimeZone, res.Warnings[0].Code)
		assert.Equal(t, 0, res.Warnings[0].Line)
		assert.Equal(t, 1, countWarnings(res.Warnings, WarningTimeZone), "once per transcript")
		assert.Equal(t, "Not/AZone", res.Metadata.TimeZone)
		assert.Empty(t, res.Metadata.StartedAt)
		for _, req := range res.LMStudioRequests {
			assert.True(t, req.ResolvedTime.IsZero())
		}
	})

	t.Run("named zone", func(t *testing.T) {
		t.Setenv("TZ", ":Australia/Melbourne")
		res, err := ParseLMStudio(strings.NewReader(input))
		require.NoError(t, err)
		assert.Zero(t, countWarnings(res.Warnings, WarningTimeZone))
		assert.Equal(t, "Australia/Melbourne", res.Metadata.TimeZone)
		assert.Equal(t, "2026-03-02T09:15:01+11:00", res.Metadata.StartedAt)
	})

	t.Run("unset", func(t *testing.T) {
		t.Setenv("TZ", "UTC")
		require.NoError(t, os.Unsetenv("TZ"))
		res, err := ParseLMStudio(strings.NewReader(input))
		require.NoError(t, err)
		assert.Equal(t, "host local zone", res.Metadata.TimeZone)
	})
}

func countWarnings(warnings []ParseWarning, code string) int {
	n := 0
	for _, w := range warnings {
		if w.Code == code {
			n++
		}
	}
	return n
}

// lmTestEntries builds request entries whose items are the given logged JSON values.
func lmTestEntries(requestID string, items ...string) []Entry {
	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		entries = append(entries, Entry{RequestID: requestID, ItemDigest: lmDigest("input", json.RawMessage(item))})
	}
	return entries
}

func TestAnnotateExtensions(t *testing.T) {
	type got struct {
		prefix        int
		discontinuity bool
	}
	tests := map[string]struct {
		requests []LMStudioRequest
		items    map[string][]string
		want     []got
	}{
		"identical request": {
			requests: []LMStudioRequest{{ID: "a", ThreadID: "t"}, {ID: "b", ThreadID: "t"}},
			items:    map[string][]string{"a": {`"x"`, `"y"`}, "b": {`"x"`, `"y"`}},
			want:     []got{{0, false}, {2, false}},
		},
		"empty predecessor": {
			requests: []LMStudioRequest{{ID: "a", ThreadID: "t"}, {ID: "b", ThreadID: "t"}},
			items:    map[string][]string{"b": {`"x"`}},
			want:     []got{{0, false}, {0, false}},
		},
		"predecessor body not translated": {
			requests: []LMStudioRequest{{ID: "a", ThreadID: "t", BodyState: BodyStateNotLogged}, {ID: "b", ThreadID: "t"}, {ID: "c", ThreadID: "t"}},
			items:    map[string][]string{"b": {`"x"`}, "c": {`"y"`}},
			want:     []got{{0, false}, {0, false}, {0, true}},
		},
		"request body not translated": {
			requests: []LMStudioRequest{{ID: "a", ThreadID: "t"}, {ID: "b", ThreadID: "t", BodyState: BodyStateMalformed}},
			items:    map[string][]string{"a": {`"x"`}},
			want:     []got{{0, false}, {0, false}},
		},
		"interleaved threads and singletons": {
			requests: []LMStudioRequest{{ID: "a", ThreadID: "t"}, {ID: "s"}, {ID: "u1", ThreadID: "u"}, {ID: "b", ThreadID: "t"}, {ID: "u2", ThreadID: "u"}},
			items: map[string][]string{
				"a": {`"x"`}, "s": {`"x"`, `"z"`}, "u1": {`"q"`, `"r"`},
				"b": {`"x"`, `"y"`}, "u2": {`"q"`},
			},
			want: []got{{0, false}, {0, false}, {0, false}, {1, false}, {0, true}},
		},
		"stale annotations are reset": {
			requests: []LMStudioRequest{{ID: "a", ThreadID: "t", PriorPrefixCount: 3, ContextDiscontinuity: true}, {ID: "s", PriorPrefixCount: 1}},
			want:     []got{{0, false}, {0, false}},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			byRequest := map[string][]Entry{}
			for id, items := range tc.items {
				byRequest[id] = lmTestEntries(id, items...)
			}
			annotateExtensions(tc.requests, byRequest)
			var annotations []got
			for _, req := range tc.requests {
				annotations = append(annotations, got{req.PriorPrefixCount, req.ContextDiscontinuity})
			}
			assert.Equal(t, tc.want, annotations)
		})
	}
}

// lmJSONValue is a generated logged value: a string, an integer, an array of
// strings or an object of them.
var lmJSONValue = rapid.Custom(func(t *rapid.T) any {
	word := rapid.SampledFrom([]string{"user", "assistant", "hello", "", "a ... <Truncated in logs> ... z"})
	switch rapid.IntRange(0, 2).Draw(t, "kind") {
	case 0:
		return word.Draw(t, "string")
	case 1:
		return rapid.IntRange(0, 2).Draw(t, "int")
	}
	words := rapid.SliceOfN(word, 0, 3).Draw(t, "array")
	out := make([]any, len(words))
	for i, w := range words {
		out[i] = w
	}
	return out
})

// lmLoggedItem is a generated logged item. A small key and value alphabet
// makes equal items and near misses common.
var lmLoggedItem = rapid.Custom(func(t *rapid.T) map[string]any {
	item := map[string]any{}
	for _, key := range []string{"type", "role", "content", "call_id"} {
		if rapid.Bool().Draw(t, "has "+key) {
			item[key] = lmJSONValue.Draw(t, key)
		}
	}
	return item
})

// marshalShuffled writes an item with its keys in a drawn order, so equal
// items reach the comparison with different logged key orders.
func marshalShuffled(t *rapid.T, item map[string]any) string {
	keys := make([]string, 0, len(item))
	for key := range item {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	keys = rapid.Permutation(keys).Draw(t, "key order")
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, key := range keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		k, _ := json.Marshal(key)
		v, _ := json.Marshal(item[key])
		buf.Write(k)
		buf.WriteByte(':')
		buf.Write(v)
	}
	buf.WriteByte('}')
	return buf.String()
}

// A threaded request extends its predecessor exactly when the predecessor's
// items are a leading prefix of its own, with object-key order irrelevant and
// array order and string contents significant (requirements 3.5, 3.6).
// Requests of other threads and singletons in between never take part.
func TestPropertyAnnotateExtensions(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		prior := rapid.SliceOfN(lmLoggedItem, 0, 4).Draw(rt, "prior")
		var current []map[string]any
		switch rapid.IntRange(0, 2).Draw(rt, "shape") {
		case 0: // an extension
			current = append(slices.Clone(prior), rapid.SliceOfN(lmLoggedItem, 0, 3).Draw(rt, "suffix")...)
		case 1: // unrelated
			current = rapid.SliceOfN(lmLoggedItem, 0, 5).Draw(rt, "current")
		default: // a near miss
			current = slices.Clone(prior)
			if len(current) > 0 {
				i := rapid.IntRange(0, len(current)-1).Draw(rt, "changed item")
				switch rapid.IntRange(0, 2).Draw(rt, "change") {
				case 0:
					current = slices.Delete(current, i, i+1)
				case 1:
					current[i] = lmLoggedItem.Draw(rt, "replacement")
				default:
					current = append(current[:i:i], current[i+1:]...)
					current = append(current, prior[i])
				}
			}
		}

		toEntries := func(id string, items []map[string]any) []Entry {
			raws := make([]string, len(items))
			for i, item := range items {
				raws[i] = marshalShuffled(rt, item)
			}
			return lmTestEntries(id, raws...)
		}
		byRequest := map[string][]Entry{
			"a":  toEntries("a", prior),
			"s":  toEntries("s", rapid.SliceOfN(lmLoggedItem, 0, 3).Draw(rt, "singleton")),
			"u1": toEntries("u1", rapid.SliceOfN(lmLoggedItem, 0, 3).Draw(rt, "other thread")),
			"b":  toEntries("b", current),
		}
		requests := []LMStudioRequest{{ID: "a", ThreadID: "t"}, {ID: "s"}, {ID: "u1", ThreadID: "u"}, {ID: "b", ThreadID: "t"}}
		annotateExtensions(requests, byRequest)

		isPrefix := len(prior) <= len(current) && reflect.DeepEqual(prior, current[:len(prior)])
		if len(prior) == 0 {
			isPrefix = true // DeepEqual distinguishes nil from empty slices
		}
		b := requests[3]
		if isPrefix {
			if b.ContextDiscontinuity || b.PriorPrefixCount != len(prior) {
				rt.Fatalf("extension of %d items annotated as prefix %d, discontinuity %v", len(prior), b.PriorPrefixCount, b.ContextDiscontinuity)
			}
		} else if !b.ContextDiscontinuity || b.PriorPrefixCount != 0 {
			rt.Fatalf("non-extension annotated as prefix %d, discontinuity %v", b.PriorPrefixCount, b.ContextDiscontinuity)
		}
		for _, other := range requests[:3] {
			if other.ContextDiscontinuity || other.PriorPrefixCount != 0 {
				rt.Fatalf("request %s has no predecessor but was annotated", other.ID)
			}
		}
	})
}
