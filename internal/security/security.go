// Package security computes what a finding may not invent: its score, its CWE id, its Top 25 flag,
// and what changed since the last audit.
package security

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Lutherwaves/builder/internal/cli"
	"github.com/Lutherwaves/builder/internal/harness"
)

const prefix = "CVSS:3.1/"

var metrics = []string{"AV", "AC", "PR", "UI", "S", "C", "I", "A"}

var values = map[string]map[string]float64{
	"AV": {"N": 0.85, "A": 0.62, "L": 0.55, "P": 0.2},
	"AC": {"L": 0.77, "H": 0.44},
	"PR": {"N": 0.85, "L": 0.62, "H": 0.27},
	"UI": {"N": 0.85, "R": 0.62},
	"S":  {"U": 0, "C": 0},
	"C":  {"H": 0.56, "L": 0.22, "N": 0.0},
	"I":  {"H": 0.56, "L": 0.22, "N": 0.0},
	"A":  {"H": 0.56, "L": 0.22, "N": 0.0},
}

var prChanged = map[string]float64{"N": 0.85, "L": 0.68, "H": 0.5}

// Top25_2025 is MITRE's 2025 CWE Top 25, in rank order.
// Source: https://cwe.mitre.org/top25/archive/2025/2025_cwe_top25.html (read 2026-10-24).
var Top25_2025 = []string{
	"CWE-79", "CWE-89", "CWE-352", "CWE-862", "CWE-787", "CWE-22", "CWE-416", "CWE-125",
	"CWE-78", "CWE-94", "CWE-120", "CWE-434", "CWE-476", "CWE-121", "CWE-502", "CWE-122",
	"CWE-863", "CWE-20", "CWE-284", "CWE-200", "CWE-306", "CWE-918", "CWE-77", "CWE-639",
	"CWE-770",
}

var Dimensions = []string{
	"osi-l1-l2", "osi-l3-l4", "osi-l5-l6", "osi-l7", "memory", "availability", "authn", "authz",
	"injection", "ssrf", "secrets", "supply-chain", "agent-boundary", "data-integrity",
}

var Modes = []string{"change", "audit"}

func ParseVector(v any) (map[string]string, error) {
	s, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("vector must be a string like %s..., got %s", prefix, cli.Repr(v))
	}
	if !strings.HasPrefix(s, prefix) {
		return nil, fmt.Errorf("vector must start with %s, got %s", prefix, cli.Repr(strings.Split(s, "/")[0]))
	}
	m := map[string]string{}
	for _, part := range strings.Split(s[len(prefix):], "/") {
		key, value, _ := strings.Cut(part, ":")
		if _, dup := m[key]; dup {
			return nil, fmt.Errorf("duplicate metric %s in %s", key, cli.Repr(s))
		}
		if _, ok := values[key][value]; !ok {
			return nil, fmt.Errorf("bad metric %s in %s", part, cli.Repr(s))
		}
		m[key] = value
	}
	var missing []string
	for _, k := range metrics {
		if _, ok := m[k]; !ok {
			missing = append(missing, k)
		}
	}
	if missing != nil {
		return nil, fmt.Errorf("missing metric %s in %s", strings.Join(missing, ", "), cli.Repr(s))
	}
	return m, nil
}

// roundup is CVSS v3.1 Appendix A: it avoids the float error of a plain ceil(x * 10).
func roundup(x float64) float64 {
	n := math.RoundToEven(x * 100000)
	if math.Mod(n, 10000) == 0 {
		return n / 100000
	}
	return (math.Floor(n/10000) + 1) / 10
}

func BaseScore(v any) (float64, error) {
	m, err := ParseVector(v)
	if err != nil {
		return 0, err
	}
	changed := m["S"] == "C"
	iss := 1 - (1-values["C"][m["C"]])*(1-values["I"][m["I"]])*(1-values["A"][m["A"]])
	impact := 6.42 * iss
	if changed {
		impact = 7.52*(iss-0.029) - 3.25*math.Pow(iss-0.02, 15)
	}
	if impact <= 0 {
		return 0, nil
	}
	pr := values["PR"][m["PR"]]
	if changed {
		pr = prChanged[m["PR"]]
	}
	total := impact + 8.22*values["AV"][m["AV"]]*values["AC"][m["AC"]]*pr*values["UI"][m["UI"]]
	if changed {
		total *= 1.08
	}
	return roundup(math.Min(total, 10)), nil
}

func Rating(s float64) string {
	switch {
	case s == 0:
		return "None"
	case s < 4:
		return "Low"
	case s < 7:
		return "Medium"
	case s < 9:
		return "High"
	}
	return "Critical"
}

var cweID = regexp.MustCompile(`^CWE-[1-9][0-9]*$`)

func CheckCWE(c any) (string, error) {
	s, ok := c.(string)
	if !ok || !cweID.MatchString(s) {
		return "", fmt.Errorf("bad CWE id %s (want CWE-<number>)", cli.Repr(c))
	}
	return s, nil
}

func InTop25(c any) (bool, error) {
	s, err := CheckCWE(c)
	return err == nil && slices.Contains(Top25_2025, s), err
}

func text(obj map[string]any, key string) (string, error) {
	s, ok := obj[key].(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "", fmt.Errorf("%s must be a non-empty string, got %s", key, cli.Repr(obj[key]))
	}
	return s, nil
}

func copyMap(m map[string]any, kv ...any) map[string]any {
	out := make(map[string]any, len(m)+len(kv)/2)
	for k, v := range m {
		out[k] = v
	}
	for n := 0; n+1 < len(kv); n += 2 {
		out[kv[n].(string)] = kv[n+1]
	}
	return out
}

func Enrich(d any) (map[string]any, error) {
	doc, ok := d.(map[string]any)
	if !ok {
		return nil, errors.New("the findings file must hold a JSON object")
	}
	if mode, _ := doc["mode"].(string); !slices.Contains(Modes, mode) {
		return nil, fmt.Errorf("mode must be one of %s, got %s", strings.Join(Modes, ", "), cli.Repr(doc["mode"]))
	}
	repo, err := text(doc, "repo")
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(repo) {
		return nil, fmt.Errorf("repo must be an absolute path to the person's checkout, got %s", cli.Repr(repo))
	}
	if _, err := text(doc, "sha"); err != nil {
		return nil, err
	}
	for _, key := range []string{"dimensions", "findings"} {
		if _, ok := doc[key].([]any); !ok {
			return nil, fmt.Errorf("%s must be a list (an empty list means none), got %s", key, cli.Repr(doc[key]))
		}
	}
	for _, dim := range doc["dimensions"].([]any) {
		if s, _ := dim.(string); !slices.Contains(Dimensions, s) {
			return nil, fmt.Errorf("unknown dimension %s in dimensions", cli.Repr(dim))
		}
	}
	seen := map[string]bool{}
	out := []map[string]any{}
	for _, raw := range doc["findings"].([]any) {
		f, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("each finding must be a JSON object, got %s", cli.Repr(raw))
		}
		fid, err := text(f, "id")
		if err != nil {
			return nil, err
		}
		enriched, err := enrichOne(f, fid, seen)
		if err != nil {
			return nil, fmt.Errorf("finding %s: %w", fid, err)
		}
		out = append(out, enriched)
	}
	return copyMap(doc, "findings", out), nil
}

func enrichOne(f map[string]any, fid string, seen map[string]bool) (map[string]any, error) {
	for _, key := range []string{"location", "title"} {
		if _, err := text(f, key); err != nil {
			return nil, err
		}
	}
	if seen[fid] {
		return nil, errors.New("duplicate id")
	}
	seen[fid] = true
	if dim, _ := f["dimension"].(string); !slices.Contains(Dimensions, dim) {
		return nil, fmt.Errorf("unknown dimension %s", cli.Repr(f["dimension"]))
	}
	vector, has := f["vector"]
	if !has {
		vector = ""
	}
	score, err := BaseScore(vector)
	if err != nil {
		return nil, err
	}
	top25, err := InTop25(f["cwe"])
	if err != nil {
		return nil, err
	}
	return copyMap(f, "score", score, "rating", Rating(score), "top25", top25), nil
}

// realpath is os.path.realpath: symlinks resolved where the path exists, cleaned where it does not.
func realpath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = filepath.Clean(p)
	}
	if r, err := filepath.EvalSymlinks(abs); err == nil {
		return r
	}
	return abs
}

func BaselinePath(repo string, env map[string]string, home string) string {
	sum := sha256.Sum256([]byte(realpath(repo)))
	return filepath.Join(harness.BuilderHome(env, home), "security", hex.EncodeToString(sum[:])[:16]+".json")
}

func isDate(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

// LoadBaseline returns nil when there is no baseline yet.
func LoadBaseline(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	corrupt := fmt.Errorf("corrupt baseline: %s (fix or delete it; the next audit then starts fresh)", path)
	var data map[string]any
	if err != nil || json.Unmarshal(b, &data) != nil || data == nil {
		return nil, corrupt
	}
	open, ok := data["open"].([]any)
	if !ok {
		return nil, corrupt
	}
	for _, raw := range open {
		f, ok := raw.(map[string]any)
		if _, hasID := f["id"]; !ok || !hasID || !isDate(f["first_seen"]) {
			return nil, corrupt
		}
	}
	return data, nil
}

func days(firstSeen string, today time.Time) int {
	t, _ := time.Parse("2006-01-02", firstSeen)
	return int(today.Sub(t).Hours() / 24)
}

func Diff(prior, current map[string]any, today time.Time) map[string][]map[string]any {
	var order []string
	known := map[string]map[string]any{}
	if prior != nil {
		for _, raw := range prior["open"].([]any) {
			f := raw.(map[string]any)
			id := fmt.Sprint(f["id"])
			if _, dup := known[id]; !dup {
				order = append(order, id)
			}
			known[id] = f
		}
	}
	// A change review looks only at the diff, so it can never close a gap.
	ran := map[string]bool{}
	if current["mode"] == "audit" {
		for _, d := range current["dimensions"].([]any) {
			ran[fmt.Sprint(d)] = true
		}
	}
	out := map[string][]map[string]any{"new": {}, "still_open": {}, "closed": {}, "not_rechecked": {}}
	found := map[string]bool{}
	for _, f := range current["findings"].([]map[string]any) {
		id := fmt.Sprint(f["id"])
		found[id] = true
		if old, ok := known[id]; ok {
			first := old["first_seen"].(string)
			out["still_open"] = append(out["still_open"], copyMap(f, "first_seen", first, "first_sha", old["first_sha"], "days_open", days(first, today)))
		} else {
			out["new"] = append(out["new"], copyMap(f, "first_seen", today.Format("2006-01-02"), "first_sha", current["sha"]))
		}
	}
	for _, id := range order {
		if found[id] {
			continue
		}
		old := known[id]
		if dim, _ := old["dimension"].(string); ran[dim] {
			out["closed"] = append(out["closed"], old)
		} else {
			out["not_rechecked"] = append(out["not_rechecked"], copyMap(old, "days_open", days(old["first_seen"].(string), today)))
		}
	}
	return out
}

func NextBaseline(prior, current map[string]any, today time.Time) map[string]any {
	d := Diff(prior, current, today)
	keep := []map[string]any{}
	for _, f := range slices.Concat(d["new"], d["still_open"], d["not_rechecked"]) {
		c := copyMap(f)
		delete(c, "days_open")
		keep = append(keep, c)
	}
	dims, _ := current["dimensions"].([]any)
	return map[string]any{"repo": current["repo"], "sha": current["sha"], "date": today.Format("2006-01-02"), "dimensions": slices.Clone(dims), "open": keep}
}
