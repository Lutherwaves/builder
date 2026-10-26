// Package ledger is builder's local memory of what it recommended and what the person answered.
package ledger

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"

	"github.com/Lutherwaves/builder/internal/harness"
)

const CooldownS = 604800 // "not now" stays quiet for 7 days

var Verdicts = []string{"yes", "later", "never"}

type Answer struct {
	Verdict string  `json:"verdict"`
	At      float64 `json:"at"`
}

type Ledger struct {
	Answers   map[string]Answer `json:"answers"`
	Onboarded bool              `json:"onboarded"`
}

// Item is one catalog entry, kept whole so `ledger next` prints every field.
type Item map[string]any

func Home() string { return harness.BuilderHome(harness.Env(), userHome()) }

func userHome() string {
	h, _ := os.UserHomeDir()
	return h
}

func Load(path string) Ledger {
	out := Ledger{Answers: map[string]Answer{}}
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	var data map[string]any
	if json.Unmarshal(b, &data) != nil {
		return out
	}
	answers, ok := data["answers"].(map[string]any)
	if !ok {
		return out
	}
	for k, v := range answers {
		a, _ := v.(map[string]any)
		verdict, _ := a["verdict"].(string)
		at, isNum := a["at"].(float64)
		if isNum && slices.Contains(Verdicts, verdict) {
			out.Answers[k] = Answer{verdict, at}
		}
	}
	out.Onboarded = truthy(data["onboarded"])
	return out
}

// truthy is Python's bool() of a JSON value.
func truthy(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case bool:
		return x
	case float64:
		return x != 0
	case string:
		return x != ""
	case []any:
		return len(x) > 0
	case map[string]any:
		return len(x) > 0
	}
	return true
}

func Save(path string, l Ledger) error { return SaveJSON(path, l, "") }

// SaveJSON writes v as JSON through a temp file in the same folder, then renames it into place.
func SaveJSON(path string, v any, indent string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if indent != "" {
		b, err = json.MarshalIndent(v, "", indent)
	}
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Rename(f.Name(), path)
	}
	if err != nil {
		os.Remove(f.Name())
	}
	return err
}

func LoadCatalog(root string) ([]Item, error) {
	b, err := os.ReadFile(filepath.Join(root, "lib", "recommendations.json"))
	if err != nil {
		return nil, err
	}
	var cat []Item
	return cat, json.Unmarshal(b, &cat)
}

func num(v any) float64 {
	f, _ := v.(float64)
	return f
}

func isNumber(v any) bool { _, ok := v.(float64); return ok }

func when(i Item) map[string]any {
	w, _ := i["when"].(map[string]any)
	return w
}

func holds(w, sig map[string]any, onboarding bool) bool {
	for key, expected := range w {
		actual := sig[key]
		if actual == nil && expected != nil {
			if _, isBool := expected.(bool); onboarding && isBool {
				continue // builder cannot see this on this harness: onboarding offers it once
			}
			return false
		}
		if bound, ok := expected.(map[string]any); ok {
			if !isNumber(actual) || num(actual) < num(bound[">="]) {
				return false
			}
		} else if actual != expected {
			return false
		}
	}
	return true
}

func unseen(w, sig map[string]any) bool {
	for k, v := range w {
		if _, isBool := v.(bool); isBool && sig[k] == nil {
			return true
		}
	}
	return false
}

// ForHarness keeps the items that work on this harness, each with `how` resolved to this harness's own string.
func ForHarness(cat []Item, name string) []Item {
	var out []Item
	for _, i := range cat {
		if hs, ok := i["harnesses"].([]any); ok && !slices.Contains(hs, any(name)) {
			continue
		}
		how := i["how"]
		if m, ok := how.(map[string]any); ok {
			if how, ok = m[name]; !ok {
				continue
			}
		}
		c := Item{}
		for k, v := range i {
			c[k] = v
		}
		c["how"] = how
		out = append(out, c)
	}
	return out
}

// DueCatalog: until onboarding ends, only onboarding items are due; the hook and `next` agree.
func DueCatalog(l Ledger, cat []Item) []Item {
	if l.Onboarded {
		return slices.Clone(cat)
	}
	var out []Item
	for _, i := range cat {
		if truthy(i["onboarding"]) {
			out = append(out, i)
		}
	}
	return out
}

func Next(l Ledger, cat []Item, sig map[string]any, now, cooldownS float64, onboarding bool) Item {
	sorted := slices.Clone(cat)
	sort.SliceStable(sorted, func(a, b int) bool { return num(sorted[a]["order"]) < num(sorted[b]["order"]) })
	for _, i := range sorted {
		id, _ := i["id"].(string)
		a, answered := l.Answers[id]
		if answered && onboarding && unseen(when(i), sig) {
			continue // an unknown step is asked once, whatever the answer
		}
		if answered && (a.Verdict == "yes" || a.Verdict == "never" || a.Verdict == "later" && now-a.At < cooldownS) {
			continue
		}
		if holds(when(i), sig, onboarding) {
			return i
		}
	}
	return nil
}
