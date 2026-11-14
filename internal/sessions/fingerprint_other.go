//go:build !unix

package sessions

import "os"

// statFingerprint reports every file as not cacheable, so no listing cache
// hits on non-Unix builds and each refresh re-reads every file. Results
// stay correct; only the zero-read refresh is given up (Decisions 6 and 23, Q20).
func statFingerprint(path string) (fileFingerprint, bool, error) {
	if _, err := os.Stat(path); err != nil {
		return fileFingerprint{}, false, err
	}
	return fileFingerprint{}, false, nil
}
