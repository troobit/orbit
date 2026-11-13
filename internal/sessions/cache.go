package sessions

import "slices"

// fileCache keeps per-file extraction results across ListAll passes, keyed by
// file fingerprints, so an unchanged refresh reads no transcript bodies
// (requirement 5.9, Decision 23). Callers hold Lister.mu for the whole pass;
// the cache has no lock of its own.
type fileCache struct {
	entries map[string]*cacheEntry
	pass    uint64
	// onCompute, when set, is called for every extraction that runs. Tests use
	// it to assert that an unchanged pass runs none.
	onCompute func(key string)
}

type cacheEntry struct {
	fingerprints []fileFingerprint
	value        any
	pass         uint64 // the last pass that used the entry
}

// beginPass starts a listing pass.
func (c *fileCache) beginPass() {
	if c != nil {
		c.pass++
	}
}

// endPass drops entries the pass did not use: files that were removed or are
// no longer reached.
func (c *fileCache) endPass() {
	if c == nil {
		return
	}
	for key, e := range c.entries {
		if e.pass != c.pass {
			delete(c.entries, key)
		}
	}
}

// cached returns the result of compute for key while every file in files
// (the main file first, then companions, absence included) keeps its
// fingerprint. A nil cache always computes.
//
// A result is stored only when compute succeeds, every fingerprint can be
// trusted and none changed during the extraction; otherwise it is used for
// this pass and recomputed on the next. A fingerprint error is never a hit.
// compute must return a value of type T that callers may share, so it must
// not alias state a caller mutates.
func cached[T any](c *fileCache, key string, files []string, compute func() (T, error)) (T, error) {
	if c == nil {
		return compute()
	}
	fps, ok := c.fingerprints(files)
	if ok {
		if e, found := c.entries[key]; found && slices.Equal(e.fingerprints, fps) {
			if v, isT := e.value.(T); isT {
				e.pass = c.pass
				return v, nil
			}
		}
	}
	delete(c.entries, key)
	if c.onCompute != nil {
		c.onCompute(key)
	}
	v, err := compute()
	if err != nil || !ok {
		return v, err
	}
	// A key that changed during extraction is not cached.
	if after, stable := c.fingerprints(files); !stable || !slices.Equal(after, fps) {
		return v, nil
	}
	if c.entries == nil {
		c.entries = make(map[string]*cacheEntry)
	}
	c.entries[key] = &cacheEntry{fingerprints: fps, value: v, pass: c.pass}
	return v, nil
}

// fingerprints fingerprints files, the first strictly and the rest as
// companions. ok is false when any fingerprint is unavailable.
func (c *fileCache) fingerprints(files []string) ([]fileFingerprint, bool) {
	fps := make([]fileFingerprint, len(files))
	for i, path := range files {
		fp, cacheable, err := fingerprintFile(path)
		if i > 0 {
			fp, cacheable, err = companionFingerprint(path)
		}
		if err != nil || !cacheable {
			return nil, false
		}
		fps[i] = fp
	}
	return fps, true
}
