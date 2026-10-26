package ledger

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/Lutherwaves/builder/internal/harness"
)

const root = "../.."

func empty() Ledger { return Ledger{Answers: map[string]Answer{}} }

func item(id string, when map[string]any) Item {
	if when == nil {
		when = map[string]any{}
	}
	return Item{"id": id, "order": 0.0, "when": when}
}

func with(i Item, kv ...any) Item {
	out := Item{}
	for k, v := range i {
		out[k] = v
	}
	for n := 0; n+1 < len(kv); n += 2 {
		out[kv[n].(string)] = kv[n+1]
	}
	return out
}

func led(onboarded bool, kv ...any) Ledger {
	l := Ledger{Answers: map[string]Answer{}, Onboarded: onboarded}
	for n := 0; n+2 < len(kv)+1; n += 3 {
		l.Answers[kv[n].(string)] = Answer{Verdict: kv[n+1].(string), At: kv[n+2].(float64)}
	}
	return l
}

func id(t *testing.T, i Item) string {
	t.Helper()
	if i == nil {
		return "<nil>"
	}
	return i["id"].(string)
}

func ids(cat []Item) []string {
	out := []string{}
	for _, i := range cat {
		out = append(out, i["id"].(string))
	}
	return out
}

func catalog(t *testing.T) []Item {
	t.Helper()
	c, err := LoadCatalog(root)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func sortedCatalog(t *testing.T) []Item {
	c := catalog(t)
	sort.SliceStable(c, func(a, b int) bool { return num(c[a]["order"]) < num(c[b]["order"]) })
	return c
}

func byID(cat []Item) map[string]Item {
	m := map[string]Item{}
	for _, i := range cat {
		m[i["id"].(string)] = i
	}
	return m
}

func sig(kv ...any) map[string]any {
	m := map[string]any{}
	for n := 0; n+1 < len(kv); n += 2 {
		m[kv[n].(string)] = kv[n+1]
	}
	return m
}

func next(l Ledger, cat []Item, s map[string]any, now float64, onboarding bool) Item {
	return Next(l, cat, s, now, CooldownS, onboarding)
}

func TestCorruptLedgerIsEmpty(t *testing.T) {
	p := filepath.Join(t.TempDir(), "ledger.json")
	os.WriteFile(p, []byte("{not json"), 0o644)
	if got := Load(p); !reflect.DeepEqual(got, empty()) {
		t.Fatalf("got %#v", got)
	}
}

func TestWrongStructureIsDropped(t *testing.T) {
	p := filepath.Join(t.TempDir(), "ledger.json")
	os.WriteFile(p, []byte(`{"answers": {"a": "yes", "b": {"verdict": "later", "at": "x"}, "c": {"verdict": "maybe", "at": 1}, "d": {"verdict": "never", "at": 5}, "e": {"verdict": "yes", "at": true}}, "onboarded": "yes"}`), 0o644)
	l := Load(p)
	if len(l.Answers) != 1 || l.Answers["d"].Verdict != "never" || !l.Onboarded {
		t.Fatalf("got %#v", l)
	}
	if got := next(l, []Item{item("d", nil)}, sig(), 10, false); got != nil {
		t.Fatalf("got %v", got)
	}
}

func TestMissingLedgerIsEmpty(t *testing.T) {
	if got := Load(filepath.Join(t.TempDir(), "none.json")); !reflect.DeepEqual(got, empty()) {
		t.Fatalf("got %#v", got)
	}
}

func TestLoadsALedgerPythonWrote(t *testing.T) {
	p := filepath.Join(t.TempDir(), "ledger.json")
	os.WriteFile(p, []byte(`{"answers": {"gh": {"verdict": "later", "at": 1792996234.123456}}, "onboarded": false}`), 0o644)
	if a := Load(p).Answers["gh"]; a.At != 1792996234.123456 || a.Verdict != "later" {
		t.Fatalf("got %#v", a)
	}
}

func TestConcurrentSavesParse(t *testing.T) {
	p := filepath.Join(t.TempDir(), "ledger.json")
	var wg sync.WaitGroup
	for n := 0; n < 20; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			Save(p, led(true, "k", "yes", float64(n)))
		}(n)
	}
	wg.Wait()
	var v map[string]any
	b, _ := os.ReadFile(p)
	if err := json.Unmarshal(b, &v); err != nil || v["answers"] == nil {
		t.Fatalf("%v %s", err, b)
	}
}

func TestNextRespectsOrderAndAnswers(t *testing.T) {
	cat := []Item{with(item("c", nil), "order", 3.0), with(item("a", nil), "order", 1.0), with(item("b", nil), "order", 2.0)}
	if got := id(t, next(led(false, "a", "never", 1.0), cat, sig(), 100, false)); got != "b" {
		t.Fatal(got)
	}
}

func TestYesIsNotRecommendedAgain(t *testing.T) {
	if got := next(led(false, "a", "yes", 1.0), []Item{item("a", nil)}, sig(), 100, false); got != nil {
		t.Fatal(got)
	}
}

func TestLaterCoolsDown(t *testing.T) {
	now := 1_000_000.0
	cat := []Item{item("a", nil)}
	if got := next(led(false, "a", "later", now-100), cat, sig(), now, false); got != nil {
		t.Fatal(got)
	}
	if got := id(t, next(led(false, "a", "later", now-604801), cat, sig(), now, false)); got != "a" {
		t.Fatal(got)
	}
}

func TestGeCondition(t *testing.T) {
	cat := []Item{item("a", map[string]any{"sessions": map[string]any{">=": 3.0}})}
	if got := next(empty(), cat, sig("sessions", 2.0), 1, false); got != nil {
		t.Fatal(got)
	}
	if got := id(t, next(empty(), cat, sig("sessions", 3.0), 1, false)); got != "a" {
		t.Fatal(got)
	}
	if got := next(empty(), cat, sig("sessions", true), 1, false); got != nil {
		t.Fatal("a bool is not a number")
	}
}

func TestNoneSignalNeverSatisfies(t *testing.T) {
	for _, c := range []struct {
		when map[string]any
		s    map[string]any
	}{
		{map[string]any{"sessions": map[string]any{">=": 0.0}}, sig("sessions", nil)},
		{map[string]any{"sessions": 0.0}, sig("sessions", nil)},
		{map[string]any{"x": true}, sig()},
	} {
		if got := next(empty(), []Item{item("a", c.when)}, c.s, 1, false); got != nil {
			t.Fatalf("%v: %v", c.when, got)
		}
	}
}

func TestNoneOfferedOnceInOnboarding(t *testing.T) {
	cat := []Item{item("a", map[string]any{"superpowers": false})}
	if got := id(t, next(empty(), cat, sig("superpowers", nil), 1, true)); got != "a" {
		t.Fatal(got)
	}
	if got := next(led(false, "a", "later", 1.0), cat, sig("superpowers", nil), 2, true); got != nil {
		t.Fatal(got)
	}
}

func TestNoneNeverOfferedJit(t *testing.T) {
	cat := []Item{item("a", map[string]any{"superpowers": false})}
	if got := next(empty(), cat, sig("superpowers", nil), 1, false); got != nil {
		t.Fatal(got)
	}
}

func TestNoneAnsweredNotOfferedAgain(t *testing.T) {
	cat := []Item{item("a", map[string]any{"superpowers": false})}
	for _, v := range []string{"yes", "never"} {
		if got := next(led(false, "a", v, 1.0), cat, sig("superpowers", nil), 9e9, true); got != nil {
			t.Fatal(v, got)
		}
	}
}

func TestNoneLaterNotOfferedAfterCooldown(t *testing.T) {
	cat := []Item{item("a", map[string]any{"superpowers": false})}
	l := led(false, "a", "later", 1.0)
	now := 1.0 + CooldownS + 1
	if got := next(l, cat, sig("superpowers", nil), now, true); got != nil {
		t.Fatal(got)
	}
	if got := id(t, next(l, cat, sig("superpowers", false), now, true)); got != "a" {
		t.Fatal(got)
	}
}

func TestOnboardingDoesNotMakeUnknownNumbersOrOtherConditionsHold(t *testing.T) {
	if got := next(empty(), []Item{item("a", map[string]any{"sessions": map[string]any{">=": 1.0}})}, sig("sessions", nil), 1, true); got != nil {
		t.Fatal(got)
	}
	cat := []Item{item("a", map[string]any{"superpowers": false, "gh": false})}
	if got := next(empty(), cat, sig("superpowers", nil, "gh", true), 1, true); got != nil {
		t.Fatal(got)
	}
}

func TestEqualityWithNullExpected(t *testing.T) {
	cat := []Item{item("ch", map[string]any{"channel": nil})}
	if got := id(t, next(empty(), cat, sig("channel", nil), 1, false)); got != "ch" {
		t.Fatal(got)
	}
	if got := next(empty(), cat, sig("channel", "slack"), 1, false); got != nil {
		t.Fatal(got)
	}
}

func TestDueCatalogHoldsOnlyOnboardingItemsUntilOnboarded(t *testing.T) {
	cat := []Item{with(item("a", nil), "onboarding", true), item("b", nil)}
	if got := ids(DueCatalog(empty(), cat)); !reflect.DeepEqual(got, []string{"a"}) {
		t.Fatal(got)
	}
	if got := ids(DueCatalog(led(true), cat)); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatal(got)
	}
}

func TestChannelIsNeverOffered(t *testing.T) {
	cat := catalog(t)
	for _, i := range ids(cat) {
		if i == "channel" {
			t.Fatal("channel is in the catalog")
		}
	}
	for _, l := range []Ledger{empty(), led(true)} {
		if got := next(l, cat, sig("channel", nil), 1, !l.Onboarded); got != nil && id(t, got) == "channel" {
			t.Fatal("channel offered")
		}
	}
}

func TestOldLedgerWithAChannelAnswerStillLoads(t *testing.T) {
	p := filepath.Join(t.TempDir(), "ledger.json")
	os.WriteFile(p, []byte(`{"answers": {"channel": {"verdict": "later", "at": 1.0}}, "onboarded": false}`), 0o644)
	if got := next(Load(p), catalog(t), sig(), 2, true); got != nil && id(t, got) == "channel" {
		t.Fatal("channel offered")
	}
}

func TestReachAsksForTheChannelOnFirstUse(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(root, "skills/reach/SKILL.md"))
	if err != nil || !strings.Contains(string(b), "If `channel` is missing, ask once") {
		t.Fatal("reach SKILL.md lost its first-use channel question")
	}
}

func TestGstackInstallGoesWhereTheSignalLooks(t *testing.T) {
	how := byID(ForHarness(catalog(t), "claude"))["gstack"]["how"].(string)
	target := `"${CLAUDE_CONFIG_DIR:-$HOME/.claude}/skills/gstack"`
	if !strings.Contains(how, "CLAUDE_CONFIG_DIR") || !strings.Contains(how, target) {
		t.Fatal(how)
	}
	for _, c := range []struct{ cfg, want string }{{"/cfg", "/cfg/skills/gstack"}, {"", "/h/.claude/skills/gstack"}} {
		cmd := exec.Command("bash", "-c", "printf %s "+target)
		cmd.Env = []string{"HOME=/h", "PATH=" + os.Getenv("PATH")}
		if c.cfg != "" {
			cmd.Env = append(cmd.Env, "CLAUDE_CONFIG_DIR="+c.cfg)
		}
		out, _ := cmd.Output()
		if string(out) != c.want {
			t.Fatalf("got %q want %q", out, c.want)
		}
	}
}

func TestSigningItemComesRightAfterGh(t *testing.T) {
	cat := sortedCatalog(t)
	list := ids(cat)
	n := indexOf(list, "gh")
	if list[n+1] != "signing" {
		t.Fatal(list)
	}
	orders := map[float64]bool{}
	for _, i := range cat {
		orders[num(i["order"])] = true
	}
	if len(orders) != len(cat) {
		t.Fatal("two items share an order")
	}
	s := cat[n+1]
	if s["onboarding"] != true || !reflect.DeepEqual(s["when"], map[string]any{"signing": false}) {
		t.Fatal(s)
	}
}

func TestParityItemsFollowMatchbloxAndFireOnTheirSignals(t *testing.T) {
	cat := sortedCatalog(t)
	list := ids(cat)
	by := byID(ForHarness(cat, "claude"))
	if list[indexOf(list, "matchblox")+1] != "matchblox-setup" {
		t.Fatal(list)
	}
	maxOnboarding := 0.0
	for _, c := range cat {
		if c["onboarding"] == true && num(c["order"]) > maxOnboarding {
			maxOnboarding = num(c["order"])
		}
	}
	for _, i := range []string{"matchblox-setup", "code-graph", "autocompact"} {
		if by[i]["onboarding"] != false || num(by[i]["order"]) <= maxOnboarding {
			t.Fatal(i)
		}
	}
	eqWhen := func(i string, want map[string]any) {
		if !reflect.DeepEqual(by[i]["when"], want) {
			t.Fatalf("%s when %v", i, by[i]["when"])
		}
	}
	eqWhen("matchblox-setup", map[string]any{"matchblox": true, "matchblox_hooks": false})
	eqWhen("code-graph", map[string]any{"code_graph": false})
	eqWhen("autocompact", map[string]any{"autocompact": false})
	for i, want := range map[string]string{"matchblox-setup": "matchblox setup", "code-graph": "harness skill"} {
		if !strings.Contains(by[i]["how"].(string), want) {
			t.Fatal(i)
		}
	}
	ac := by["autocompact"]
	if !strings.Contains(ac["how"].(string), `"CLAUDE_AUTOCOMPACT_PCT_OVERRIDE": "85"`) || !strings.Contains(ac["why"].(string), "record") || !strings.Contains(ac["how"].(string), "CLAUDE_CONFIG_DIR") {
		t.Fatal(ac)
	}
	s := sig("matchblox", true, "matchblox_hooks", false, "code_graph", false, "autocompact", false)
	done := sig("matchblox", true, "matchblox_hooks", true, "code_graph", true, "autocompact", true)
	for _, i := range []string{"matchblox-setup", "code-graph", "autocompact"} {
		if got := id(t, next(empty(), []Item{by[i]}, s, 1, false)); got != i {
			t.Fatal(got)
		}
		if got := next(empty(), []Item{by[i]}, done, 1, false); got != nil {
			t.Fatal(got)
		}
	}
	if got := next(empty(), []Item{by["matchblox-setup"]}, sig("matchblox", false, "matchblox_hooks", false), 1, false); got != nil {
		t.Fatal(got)
	}
}

func TestForHarnessDropsMissingEntry(t *testing.T) {
	cat := []Item{with(item("a", nil), "how", map[string]any{"claude": "c"}), with(item("b", nil), "how", "s")}
	if got := ids(ForHarness(cat, "codex")); !reflect.DeepEqual(got, []string{"b"}) {
		t.Fatal(got)
	}
}

func TestForHarnessResolvesMap(t *testing.T) {
	cat := []Item{with(item("a", nil), "how", map[string]any{"claude": "c", "codex": "x"})}
	if got := ForHarness(cat, "codex")[0]["how"]; got != "x" {
		t.Fatal(got)
	}
	if !reflect.DeepEqual(cat[0]["how"], map[string]any{"claude": "c", "codex": "x"}) {
		t.Fatal("the catalog changed")
	}
}

func TestHarnessesFilter(t *testing.T) {
	cat := []Item{with(item("a", nil), "how", "s", "harnesses", []any{"claude"}), with(item("b", nil), "how", "s")}
	if got := ids(ForHarness(cat, "claude")); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatal(got)
	}
	if got := ids(ForHarness(cat, "cursor")); !reflect.DeepEqual(got, []string{"b"}) {
		t.Fatal(got)
	}
}

func TestCatalogEveryItemOfferedSomewhere(t *testing.T) {
	for _, i := range catalog(t) {
		offered := 0
		for _, h := range harness.Names {
			if got := ForHarness([]Item{i}, h); len(got) == 1 {
				offered++
				if _, ok := got[0]["how"].(string); !ok {
					t.Fatal(i["id"], h)
				}
			}
		}
		if offered == 0 {
			t.Fatal(i["id"])
		}
	}
}

func TestCatalogSourcesForMapHow(t *testing.T) {
	for _, i := range catalog(t) {
		how, ok := i["how"].(map[string]any)
		if !ok {
			continue
		}
		sources, _ := i["sources"].(map[string]any)
		for h := range how {
			if src, _ := sources[h].(string); h != "claude" && !strings.HasPrefix(src, "https://") {
				t.Fatal(i["id"], h)
			}
		}
	}
}

func TestNeutralHowHasNoSlashCommand(t *testing.T) {
	rx := regexp.MustCompile("(^|[\\s(`])/(loop|usage|compact|clear|plugin|builder:[a-z-]+)\\b")
	for _, i := range catalog(t) {
		if how, ok := i["how"].(string); ok && rx.MatchString(how) {
			t.Fatal(i["id"])
		}
	}
}

func claudeOnly(t *testing.T, name string) {
	t.Helper()
	if got := byID(catalog(t))[name]["harnesses"]; !reflect.DeepEqual(got, []any{"claude"}) {
		t.Fatal(got)
	}
	for _, h := range []string{"codex", "cursor", "opencode"} {
		if indexOf(ids(ForHarness(catalog(t), h)), name) >= 0 {
			t.Fatal(h)
		}
	}
}

func TestAutocompactClaudeOnly(t *testing.T)                        { claudeOnly(t, "autocompact") }
func TestCodeGraphClaudeOnlyUntilOtherHarnessesWireIt(t *testing.T) { claudeOnly(t, "code-graph") }

func TestPerHarnessItems(t *testing.T) {
	for _, h := range harness.Names {
		got := ids(ForHarness(catalog(t), h))
		for _, need := range []string{"superpowers", "gstack", "matchblox-setup", "fleet", "budget", "canon"} {
			if indexOf(got, need) < 0 {
				t.Fatal(h, need)
			}
		}
	}
	if !strings.Contains(byID(ForHarness(catalog(t), "cursor"))["superpowers"]["how"].(string), "/add-plugin superpowers") {
		t.Fatal("cursor superpowers how")
	}
}

func indexOf(list []string, s string) int {
	for n, v := range list {
		if v == s {
			return n
		}
	}
	return -1
}
