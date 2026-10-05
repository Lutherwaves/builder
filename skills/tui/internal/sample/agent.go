package sample

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// registryEntry is the per-process file Claude Code keeps at
// ~/.claude/sessions/<pid>.json. Only the fields the cockpit reads.
type registryEntry struct {
	SessionID       string `json:"sessionId"`
	Name            string `json:"name"`
	Cwd             string `json:"cwd"`
	Status          string `json:"status"`
	StartedAt       int64  `json:"startedAt"`
	StatusUpdatedAt int64  `json:"statusUpdatedAt"`
}

// Usage is the context state read from the end of a transcript.
type Usage struct {
	Model  string    `json:"model"`
	Tokens int       `json:"tokens"`
	At     time.Time `json:"at"`
}

type fileKey struct {
	size  int64
	mtime time.Time
}

// agentReader caches everything keyed on file size and mtime, so an idle
// session costs one stat per tick and no reads.
type agentReader struct {
	home        string
	registry    map[string]cached[registryEntry]
	transcripts map[string]string    // sessionId -> path
	missing     map[string]time.Time // sessionId -> when the last search found nothing
	usage       map[string]cached[Usage]
}

type cached[T any] struct {
	key fileKey
	val T
}

func newAgentReader(home string) *agentReader {
	return &agentReader{
		home:        home,
		registry:    map[string]cached[registryEntry]{},
		transcripts: map[string]string{},
		missing:     map[string]time.Time{},
		usage:       map[string]cached[Usage]{},
	}
}

func statKey(path string) (fileKey, bool) {
	st, err := os.Stat(path)
	if err != nil {
		return fileKey{}, false
	}
	return fileKey{st.Size(), st.ModTime()}, true
}

func (r *agentReader) entry(pid int) (registryEntry, bool) {
	path := filepath.Join(r.home, ".claude", "sessions", strconv.Itoa(pid)+".json")
	key, ok := statKey(path)
	if !ok {
		return registryEntry{}, false
	}
	if c, ok := r.registry[path]; ok && c.key == key {
		return c.val, true
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return registryEntry{}, false
	}
	var e registryEntry
	if json.Unmarshal(b, &e) != nil {
		return registryEntry{}, false
	}
	r.registry[path] = cached[registryEntry]{key, e}
	return e, true
}

// transcript finds the session's .jsonl: first at the path derived from its
// working directory, then by searching every project. A session that has not
// written one yet is searched again at most every 30 s, because the search
// lists every transcript on the machine.
func (r *agentReader) transcript(sessionID, cwd string) string {
	if p, ok := r.transcripts[sessionID]; ok {
		return p
	}
	direct := filepath.Join(r.home, ".claude", "projects", projectDir(cwd), sessionID+".jsonl")
	if _, err := os.Stat(direct); err == nil {
		r.transcripts[sessionID] = direct
		return direct
	}
	if at, ok := r.missing[sessionID]; ok && time.Since(at) < 30*time.Second {
		return ""
	}
	matches, _ := filepath.Glob(filepath.Join(r.home, ".claude", "projects", "*", sessionID+".jsonl"))
	if len(matches) == 0 {
		r.missing[sessionID] = time.Now()
		return ""
	}
	delete(r.missing, sessionID)
	r.transcripts[sessionID] = matches[0]
	return matches[0]
}

// projectDir is how the agent names a project directory: every character
// outside [A-Za-z0-9] becomes '-'.
func projectDir(cwd string) string {
	b := []byte(cwd)
	for i, c := range b {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			b[i] = '-'
		}
	}
	return string(b)
}

func (r *agentReader) usageOf(sessionID, cwd string) (Usage, bool) {
	path := r.transcript(sessionID, cwd)
	if path == "" {
		return Usage{}, false
	}
	key, ok := statKey(path)
	if !ok {
		delete(r.transcripts, sessionID)
		return Usage{}, false
	}
	if c, ok := r.usage[path]; ok && c.key == key {
		return c.val, true
	}
	u, ok := lastUsage(path, key.size)
	if !ok {
		return Usage{}, false
	}
	r.usage[path] = cached[Usage]{key, u}
	return u, true
}

// tailBytes bounds the read: a transcript can be hundreds of megabytes and the
// last assistant turn is always near the end.
const tailBytes = 512 << 10

type transcriptLine struct {
	Type        string    `json:"type"`
	IsSidechain bool      `json:"isSidechain"`
	Timestamp   time.Time `json:"timestamp"`
	Message     struct {
		Model string `json:"model"`
		Usage *struct {
			Input         int `json:"input_tokens"`
			CacheRead     int `json:"cache_read_input_tokens"`
			CacheCreation int `json:"cache_creation_input_tokens"`
			Output        int `json:"output_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

func lastUsage(path string, size int64) (Usage, bool) {
	f, err := os.Open(path)
	if err != nil {
		return Usage{}, false
	}
	defer f.Close()
	off := max(size-tailBytes, 0)
	buf := make([]byte, size-off)
	if _, err := f.ReadAt(buf, off); err != nil && err != io.EOF {
		return Usage{}, false
	}
	lines := bytes.Split(buf, []byte{'\n'})
	for i := len(lines) - 1; i >= 0; i-- {
		l := lines[i]
		if !bytes.Contains(l, []byte(`"usage"`)) || !bytes.Contains(l, []byte(`"assistant"`)) {
			continue
		}
		var t transcriptLine
		// <synthetic> turns are local messages that report zero usage.
		if json.Unmarshal(l, &t) != nil || t.Type != "assistant" || t.IsSidechain || t.Message.Usage == nil || t.Message.Model == "<synthetic>" {
			continue
		}
		u := t.Message.Usage
		return Usage{
			Model:  t.Message.Model,
			Tokens: u.Input + u.CacheRead + u.CacheCreation + u.Output,
			At:     t.Timestamp,
		}, true
	}
	return Usage{}, false
}
