---
name: budget
description: Use when the person asks "will I hit my limit?", "how much usage is left?", "watch my usage", "what is my burn rate?" or "which session do I compact?", when the builder hook says weekly usage is at 80% or more, or before a long run such as an overnight job, a fan-out of agents or a large refactor.
---

# budget — know the week's limit before it runs out

A plan has a weekly limit. When it runs out, every session stops until the reset.
`budget` reads the real plan windows, projects the rest of the week, and names one way to spend less.
It does not hand-roll the math. A straight-line guess on a busy evening gives a false alarm.

## 1. Read the plan windows

Run `matchblox usage --json`. It only reads. It prints a list of accounts.
Each account has `windows`: `{used_pct, window_minutes, resets_at}`.
The weekly window has `window_minutes` 10080. The 5-hour window has `window_minutes` 300.

- `used_pct` is measured when the account's `state` is `measured`. Say "measured" when you quote it. Any other `state` is not a measurement: say so, and do not project it.
- An account with no weekly window has no reading yet. Say so. Never show it as 0%.
- On Cursor, `matchblox usage` shows a monthly window, not a weekly one (matchblox 0.14.0 or newer, macOS, `cursor_usage = true`). Report that monthly percent. Do not forecast a weekly cap. Without that reading, say so and stop.
- Without matchblox, ask the person to open the usage view of their harness (see your harness mapping) and give you the weekly percent and the reset time.
  If the harness has no usage source, say so and stop.
  Then recommend matchblox once, in one line: "matchblox reads every account's plan windows for you: `curl -fsSL https://matchblox.sh | sh` (https://docs.matchblox.com)."
- Never guess a percent. A missing value shows as missing, with its cause.

## 2. Project the week

If a `bin/builder` command exits 3, stop. Show its error line, and offer once to run the install command in it. Do not work around the missing binary.

```
"${CLAUDE_PLUGIN_ROOT}/bin/builder" budget --from-matchblox
"${CLAUDE_PLUGIN_ROOT}/bin/builder" budget <used_pct> <resets_at_epoch>   # without matchblox
```

It prints `elapsed=… naive=… profile_aware=… reset_in=…`, one line per account.
When matchblox lists two or more accounts, each line ends with `account=<name>`.
An account that is not `measured`, or has a malformed weekly window, gets no line. Its cause goes to stderr. Quote it.

| Exit | Meaning | Say |
|---|---|---|
| 0 | One projection per account | The numbers, as below |
| 1 | matchblox failed, or no account has a measured weekly window | The stderr line, and the fix it names |
| 2 | matchblox is not installed | The install line it prints. Then use the positional form. |

The weekly window is fixed, not rolling. It starts at `resets_at` minus 7 days and resets in one step.
A quiet day stops the growth. It does not remove old usage.

- **naive** = used ÷ elapsed fraction. It assumes every hour burns at the average rate so far. It is the ceiling.
- **profile_aware** discounts the hours the person does not work: 02:00–09:00 (×0.05), Saturday (×0.5), Sunday (×0.1).
  It is the forecast. It bends down as the quiet hours come near.

Lead with profile_aware. Give naive as the ceiling. Both are estimated. Say so.
Flag the week only when profile_aware is over 100%, or the weekly `used_pct` is 80 or more.
The default hours fit a person who works 09:00–02:00. If the person works other hours, say that the projection assumes these hours.
To change them, edit `Mult` in `internal/budget/budget.go`.

## 3. Spend less: one compaction pick

A long context costs more on every turn. Compacting it frees usage.

1. Read `matchblox sessions --json`. Sort by `context_pct`, highest first.
2. Pick one session whose `word` is `idle`. Read `word`, not `status`: a session with `status` idle can have `word` `asks`. Never pick `busy`: a compact in the middle of a step breaks the step. Never pick `asks`: it waits for the person.
3. Give the pick, its `context_pct`, and one line of reason.
4. Do not compact it yourself. The builder `fleet` skill owns `matchblox compact`. It needs the person's yes, and a durable record before the compact.

Without matchblox, you see only this session. Say that, and give its own context size if you know it.

## 4. Before a long run

A long run is an overnight job, three or more parallel sessions, or a fan-out of agents.
Project the week first. Then give one of these, with the numbers:

- **Go.** profile_aware stays under 100% with room for the run.
- **Wait.** Start after the reset. Give the time: `reset_in` from the projection.
- **Shrink.** Run fewer parallel sessions, or compact the pick first.

Never start or stop a session to save usage. Recommend it. The person decides.

## 5. The watcher (when the person asks for it)

A watcher reports every 30 minutes. It runs `watcher-prompt.md`, beside this file.

1. **Guard against a second watcher.** List the scheduled jobs first (see your harness mapping).
   A job whose prompt runs `builder budget` or the old `burn-proj.py` already exists? Stop. Offer to replace it. Never add a second.
2. **Schedule it** at `7,37 * * * *`. These minutes stay off :00 and :30 on purpose. With no cron tool, schedule a check every 30 minutes.
   Name the `7,37` schedule either way, also when the person must run the command.
3. **Tell the limits.** Most scheduled jobs live only as long as the session. Many expire after 7 days. Tell the person to run this step again after a restart.
   A cloud schedule cannot read the local matchblox. The watcher works only on the machine that runs matchblox.

## The reply

Keep it to a status readout. Example:

```
Weekly: 62% used (measured), resets in 2.3 days.
Forecast: 91% (profile_aware, estimated). Ceiling: 118% (naive).
5-hour: 40% used.
Compact: `docs` (idle, 78% context) — the largest idle context.
```

Then end with the builder report from the builder skill.

## Common mistakes

| Mistake | Fix |
|---|---|
| "You will blow the cap" from the naive number | Lead with profile_aware. Naive is the ceiling. |
| Straight-line math written in the reply | Run `builder budget`. The sleep and weekend discount is the point. |
| A rolling 7-day window | The window is fixed. It resets in one step at `resets_at`. |
| 0% for an account with no weekly window | Say "no weekly reading yet". Never invent a zero. |
| A second watcher next to the first | List the jobs first. Replace, never stack. |
| Compacting a `busy` session, or compacting it yourself | Pick an `idle` session. `fleet` compacts it after the person's yes. |
| Projections given as facts | `used_pct` is measured. Both projections are estimated. Say so. |
| A guessed percent when matchblox is missing | Ask for the usage view, use the positional form, recommend matchblox once. |

## Self-check

Before you finish:

- [ ] Each percent came from `matchblox usage --json` or from the person, not from a guess.
- [ ] I ran `builder budget` and quoted its line.
- [ ] I led with profile_aware, gave naive as the ceiling, and marked both as estimated.
- [ ] I flagged the week only on profile_aware over 100% or weekly `used_pct` at 80 or more.
- [ ] A missing reading shows as missing, with its cause.
- [ ] The compaction pick is `idle`, and I compacted nothing myself.
- [ ] I listed the scheduled jobs before I scheduled a watcher, and I added no second one.
- [ ] The reply ends with the builder report.
