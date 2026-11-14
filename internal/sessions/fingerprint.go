package sessions

import (
	"errors"
	"io/fs"
)

// fileFingerprint identifies a file's state. statFingerprint fills it on
// Unix; elsewhere it is never compared. absent marks a companion file that
// does not exist, so creating it changes the key.
type fileFingerprint struct {
	size, mtime, ctime int64
	inode, device      uint64
	absent             bool
}

// fingerprintFile fingerprints a file and reports whether the fingerprint can
// be trusted for a cache hit. Tests replace it.
var fingerprintFile = statFingerprint

// companionFingerprint is fingerprintFile for a file that may be missing: a
// missing file is a valid, cacheable state of its own.
func companionFingerprint(path string) (fileFingerprint, bool, error) {
	fp, cacheable, err := fingerprintFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return fileFingerprint{absent: true}, true, nil
	}
	return fp, cacheable, err
}
