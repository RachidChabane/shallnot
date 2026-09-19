package agenthook

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var unsafeFileCharacters = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// Attempts counts, per session, how many times in a row a hook sent the agent
// back to work, for harnesses that do not count it themselves.
type Attempts struct {
	// Directory holds one small counter file per session.
	Directory string
}

func (a Attempts) path(sessionID string) string {
	return filepath.Join(a.Directory, "shallnot-hook-"+unsafeFileCharacters.ReplaceAllString(sessionID, "_")+".count")
}

// Count returns the consecutive blocks recorded for a session.
func (a Attempts) Count(sessionID string) int {
	content, err := os.ReadFile(a.path(sessionID))
	if err != nil {
		return 0
	}
	count, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil {
		return 0
	}
	return count
}

// Record stores the consecutive blocks of a session.
func (a Attempts) Record(sessionID string, count int) error {
	return os.WriteFile(a.path(sessionID), []byte(strconv.Itoa(count)), 0o600)
}

// Reset forgets a session, once its gate passes or the hook gives up.
func (a Attempts) Reset(sessionID string) {
	_ = os.Remove(a.path(sessionID))
}
