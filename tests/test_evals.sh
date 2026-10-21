#!/usr/bin/env bash
# Test tests/evals/run.sh --check and --dry-run.
set -uo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
fails=0
check() { if [ "$2" = "ok" ]; then echo "  ✓ $1"; else echo "  ✗ $1"; fails=1; fi; }
run() { BUILDER_EVAL_DIR="$tmp/evals" bash "$root/tests/evals/run.sh" "$@"; }

mkdir -p "$tmp/evals"
out="$(run --check 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && [ "$out" = "fixtures ok (0)" ] && check "zero fixtures" ok || check "zero fixtures ($out)" bad

mkdir -p "$tmp/evals/good"
echo "Do the thing." > "$tmp/evals/good/prompt.md"
printf -- '- a\n- b\n- c\n' > "$tmp/evals/good/expect.md"
out="$(run --check 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && [ "$out" = "fixtures ok" ] && check "good fixture passes" ok || check "good fixture passes ($out)" bad

mkdir -p "$tmp/evals/nope"; echo "x" > "$tmp/evals/nope/prompt.md"
out="$(run --check 2>&1)"; rc=$?
[ "$rc" -ne 0 ] && grep -q 'nope: expect.md is missing' <<<"$out" && check "missing expect.md fails" ok || check "missing expect.md fails" bad

rm -rf "$tmp/evals/nope"; mkdir -p "$tmp/evals/few"; echo x > "$tmp/evals/few/prompt.md"; printf -- '- a\n- b\n' > "$tmp/evals/few/expect.md"
out="$(run --check 2>&1)"; rc=$?
[ "$rc" -ne 0 ] && grep -q 'few: expect.md has 2 bullet' <<<"$out" && check "too few bullets fails" ok || check "too few bullets fails" bad
rm -rf "$tmp/evals/few"

out="$(BUILDER_EVAL_OUT="$tmp/out" run good --dry-run 2>&1)"; rc=$?
[ "$rc" -eq 0 ] && [ "$(grep -c '^claude ' <<<"$out")" -eq 2 ] \
  && grep -q -- '--plugin-dir' <<<"$out" && grep -q "$tmp/out/good/" <<<"$out" \
  && ! grep -q -- '--bare' <<<"$out" && [ ! -e "$tmp/out" ] \
  && check "dry-run prints two commands, writes nothing" ok || check "dry-run prints two commands, writes nothing" bad

# A fake claude on PATH: echoes stdin and its args to stdout, writes a marker to stderr.
mkdir -p "$tmp/bin"
cat > "$tmp/bin/claude" <<'FAKE'
#!/usr/bin/env bash
echo "ARGS: $*"; echo "STDIN: $(cat)"; echo "fake-stderr" >&2
FAKE
chmod +x "$tmp/bin/claude"
mkdir -p "$tmp/evals/dash"
printf -- '- leading dash prompt\n' > "$tmp/evals/dash/prompt.md"
printf -- '- a\n- b\n- c\n' > "$tmp/evals/dash/expect.md"
PATH="$tmp/bin:$PATH" BUILDER_EVAL_OUT="$tmp/out2" run dash </dev/null >/dev/null 2>&1
d="$(ls -d "$tmp"/out2/dash/*/ 2>/dev/null | head -1)"
[ -n "$d" ] && grep -q '^STDIN: - leading dash prompt' "$d/baseline.md" && grep -q '^STDIN: - leading dash prompt' "$d/with.md" \
  && ! grep -q 'leading dash' <(grep '^ARGS:' "$d/baseline.md") \
  && check "prompt goes to claude on stdin, not as an argument" ok || check "prompt goes to claude on stdin, not as an argument" bad
[ -n "$d" ] && grep -q 'fake-stderr' "$d/baseline.err" && grep -q 'fake-stderr' "$d/with.err" \
  && ! grep -q 'fake-stderr' "$d/baseline.md" "$d/with.md" \
  && check "stderr goes to .err files, not the transcript" ok || check "stderr goes to .err files, not the transcript" bad
rm -rf "$tmp/evals/dash"

exit "$fails"
