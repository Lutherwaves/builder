# review — the bar for a finding

The hunt finds candidates. This file decides which candidates stay as findings.
Apply it to each candidate before you write the review.

## The keep test

A finding stays when it names two things:

- Its trigger: who does what, with which input.
- Its harm: what the person loses, sees wrong or cannot do.

When one of the two is missing, the candidate does not stay as a finding.
Example: "A person uploads a 2 GB file, the handler reads it whole into memory, and the server stops for everyone" names both.
Example: "this loop could be cleaner" names neither.

## The drop list

Drop a candidate when it is one of these:

- An abstraction or an option that nobody asked for.
- A case that the callers, the types or the validation rule out. Read the callers before you drop it.
- A guard that a caller or the framework already gives.
- A rare case with no real harm.
- Style and taste: names, layout, order, wording.
- A problem that the code already handles in another place.
- A premise that the code refutes. Example: "this can be null" when the code assigns it on every path.

Write each dropped candidate under "Declined to judge", one line, with the item of this list that drops it.

## When the two disagree

The keep test wins.
A style-shaped point that names a trigger and a harm is a finding. Example: a misleading name that makes a caller pass seconds where the function expects milliseconds.
Never drop a real finding to make the review look tidy.

## When unsure

Drop the candidate.
Write it under "Declined to judge", one line, with the reason. Example: "`user_id` has no empty check: the only caller reads it from the signed session."
Never drop a candidate in silence. The person sees each one you set aside.

## The sweep

Make one more pass after the hunt. Look only where no finding points:

- Error paths: each `except`, each early return, each failed call.
- Inputs that nobody handles: empty, very large, malformed, from another user.
- Effects across files: a caller, a config, a doc or a test that the change does not touch but depends on.
- Assumptions under load or under hostile input: many requests, slow dependencies, crafted values.

Apply the keep test and the drop list to what the sweep finds.
Report only what is new. Do not repeat a finding from the hunt.
When the sweep finds nothing, say so in one line. Example: "Sweep: error paths, unhandled inputs and callers checked; nothing new."
An empty sweep is a good result. Do not pad it.
