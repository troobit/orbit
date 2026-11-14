//go:build !unix

package sessions

import "os"

// statFingerprint reports every file as not cacheable, so the LM Studio cache
// never hits on this platform and each refresh re-reads every file. Results
// stay correct; only the zero-read refresh is given up (Decision 6, Q20).
func statFingerprint(path string) (lmFileFingerprint, bool, error) {
	if _, err := os.Stat(path); err != nil {
		return lmFileFingerprint{}, false, err
	}
	return lmFileFingerprint{}, false, nil
}
