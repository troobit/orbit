//go:build unix

package sessions

import (
	"io/fs"

	"golang.org/x/sys/unix"
)

// statFingerprint fingerprints the file at path (symlinks followed) by size,
// modification and change times, inode and device. An append, truncation or
// replacement changes at least one of them even when the modification time
// is set back, because the change time cannot be (Decision 6).
func statFingerprint(path string) (lmFileFingerprint, bool, error) {
	var st unix.Stat_t
	if err := unix.Stat(path, &st); err != nil {
		return lmFileFingerprint{}, false, &fs.PathError{Op: "stat", Path: path, Err: err}
	}
	return lmFileFingerprint{
		size:   st.Size,
		mtime:  st.Mtim.Nano(),
		ctime:  st.Ctim.Nano(),
		inode:  uint64(st.Ino),
		device: uint64(st.Dev),
	}, true, nil
}
