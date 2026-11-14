package sessions

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// kiroIDEChatHeader is a lightweight struct for parsing .chat files during discovery.
type kiroIDEChatHeader struct {
	ExecutionID string            `json:"executionId"`
	Chat        []json.RawMessage `json:"chat"`
	Metadata    *kiroIDEMetadata  `json:"metadata"`
}

type kiroIDEMetadata struct {
	StartTime     int64  `json:"startTime"`
	ModelID       string `json:"modelId"`
	ModelProvider string `json:"modelProvider"`
}

// kiroIDECandidate is a .chat file holding an execution's conversation. The
// same execution can be saved in several files and workspaces.
type kiroIDECandidate struct {
	path       string
	entryCount int
	mtime      time.Time
	size       int64
	metadata   kiroIDEMetadata
}

// betterKiroIDECandidate reports whether a, rather than b, is the copy of an
// execution to list and open: more chat messages, then the newer modification
// time, then the lexically smaller path.
func betterKiroIDECandidate(a, b kiroIDECandidate) bool {
	if a.entryCount != b.entryCount {
		return a.entryCount > b.entryCount
	}
	if !a.mtime.Equal(b.mtime) {
		return a.mtime.After(b.mtime)
	}
	return a.path < b.path
}

// bestKiroIDECandidate returns the candidate betterKiroIDECandidate prefers.
// candidates must not be empty.
func bestKiroIDECandidate(candidates []kiroIDECandidate) kiroIDECandidate {
	best := candidates[0]
	for _, c := range candidates[1:] {
		if betterKiroIDECandidate(c, best) {
			best = c
		}
	}
	return best
}

// kiroIDEHead is what a .chat file's header yields. An empty executionID marks
// a file that cannot be decoded or records no execution; it is skipped, and
// that outcome is as cacheable as any other. A file that cannot be read is
// skipped too, but not cached.
type kiroIDEHead struct {
	executionID string
	entryCount  int
	metadata    kiroIDEMetadata
}

// readKiroIDEHead decodes the whole .chat file, as listing always has. A read
// failure is returned so the empty head is not cached.
func readKiroIDEHead(path string) (kiroIDEHead, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return kiroIDEHead{}, err
	}
	var header kiroIDEChatHeader
	if err := json.Unmarshal(data, &header); err != nil || header.ExecutionID == "" {
		return kiroIDEHead{}, nil
	}
	head := kiroIDEHead{executionID: header.ExecutionID, entryCount: len(header.Chat)}
	if header.Metadata != nil {
		head.metadata = *header.Metadata
	}
	return head, nil
}

// kiroIDEWorkspaceCandidates reads the .chat files of one workspace and keeps
// the best candidate per execution ID. Files that cannot be read or decoded,
// that record no execution ID, or that are symlinks escaping the workspace are
// skipped.
func kiroIDEWorkspaceCandidates(cache *fileCache, workspaceDir string) (map[string]kiroIDECandidate, error) {
	entries, err := os.ReadDir(workspaceDir)
	if err != nil {
		return nil, fmt.Errorf("read kiro ide workspace: %w", err)
	}

	best := make(map[string]kiroIDECandidate)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".chat") {
			continue
		}

		path := filepath.Join(workspaceDir, entry.Name())
		if entry.Type()&fs.ModeSymlink != 0 {
			resolved, err := filepath.EvalSymlinks(path)
			if err != nil || !isWithinDir(resolved, workspaceDir) {
				continue
			}
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}
		head, err := cached(cache, "kiro-ide:"+path, []string{path}, func() (kiroIDEHead, error) {
			return readKiroIDEHead(path)
		})
		if err != nil || head.executionID == "" {
			continue
		}

		candidate := kiroIDECandidate{
			path:       path,
			entryCount: head.entryCount,
			mtime:      info.ModTime(),
			size:       info.Size(),
			metadata:   head.metadata,
		}
		if existing, ok := best[head.executionID]; !ok || betterKiroIDECandidate(candidate, existing) {
			best[head.executionID] = candidate
		}
	}
	return best, nil
}

// kiroIDEAllCandidates returns, per execution ID, the best candidate of each
// workspace under base, in workspace order. Unreadable workspaces are skipped.
func kiroIDEAllCandidates(cache *fileCache, base string) (map[string][]kiroIDECandidate, error) {
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, fmt.Errorf("read kiro ide base: %w", err)
	}

	all := make(map[string][]kiroIDECandidate)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		best, err := kiroIDEWorkspaceCandidates(cache, filepath.Join(base, entry.Name()))
		if err != nil {
			continue
		}
		for id, c := range best {
			all[id] = append(all[id], c)
		}
	}
	return all, nil
}

// kiroIDESessionInfo is the listing row of an execution's selected candidate.
func kiroIDESessionInfo(executionID string, c kiroIDECandidate) SessionInfo {
	session := SessionInfo{
		ID:       executionID,
		Size:     c.size,
		Source:   SourceKiroIDE,
		Model:    c.metadata.ModelID,
		Provider: c.metadata.ModelProvider,
	}
	if c.metadata.StartTime > 0 {
		session.CreatedAt = time.UnixMilli(c.metadata.StartTime)
	} else {
		session.CreatedAt = c.mtime
		session.TimeFromFilesystem = true
	}
	return session
}
