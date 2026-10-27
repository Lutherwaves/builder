# Releasing

A merge to `main` that `validate` passed makes a release. Nobody tags by hand.

1. **Release** (`release.yml`) picks the next version from the conventional
   commits (`feat` → minor, `fix` → patch, a breaking change → major), checks
   that it is above the plugin's version and that the previous release's pull
   request merged, makes the draft release `vX.Y.Z` and pushes its tag on the
   commit `validate` verified. A tag that exists on another commit stops it. Its notes are the
   `## [Unreleased]` section of `CHANGELOG.md`, so write that section in the
   pull request.
2. **Publish** (`publish.yml`) refuses a tag whose commit is not on `main`. It builds `builder-{linux,darwin}-{amd64,arm64}`
   with a `.sha256`, a CycloneDX SBOM and provenance and SBOM attestations,
   attaches all twelve files to the draft and publishes it as a pre-release.
   The repository's releases are immutable: once published, a release takes
   no more assets, so the draft is what lets them in. Then it installs the
   release with `scripts/install.sh --version vX.Y.Z` of the tag, under the
   system Bash, on Linux and macOS, amd64 and arm64. It checks that the
   installer says `checksum ok, provenance ok`, that `builder version` names
   the tag, and that the SBOM is attested. Then it makes it the latest release.
3. A pull request `docs: vX.Y.Z is released` moves `[Unreleased]` under the
   version and sets `version` in the four manifests. The release job waits up
   to 60 minutes until Publish verified the release and took it out of
   pre-release. Then the pull request merges itself once its checks pass, and
   the release job keeps it up to date with `main` for 40 minutes. If either
   wait runs out, the job fails. Until it
   merges, `main` names the previous version, so the plugin installs the
   previous binary. For the first release, `main` names 2.1.0, and no release
   has a 2.1.0 binary. Until the pull request merges, every session offers an
   install that cannot succeed.

## Turn releases on or off

Releases run while the repository variable `RELEASE_ENABLED` is `true`:

```sh
gh variable set RELEASE_ENABLED --body true -R Lutherwaves/builder
```

Any other value, or no variable, holds them.

## The release token

`RELEASE_TOKEN` is a fine-grained token for this repository only, with
Contents and Pull requests set to read and write. When it is missing or
expired, the Create Release job fails and says so. To replace it:

1. GitHub → Settings → Developer settings → Fine-grained tokens → Generate
   new token. Resource owner `Lutherwaves`, only `Lutherwaves/builder`.
2. `gh secret set RELEASE_TOKEN -R Lutherwaves/builder`, and paste it.
3. Re-run the failed Release run.

## Before the first release

go-semantic-release counts from the newest `v*` tag. The releases before
builder 3 have `builder-v*` tags, so with no seed tag it would pick a first
version below 2.1.0, and the Create Release job stops. The owner does these
five steps once:

1. Allow auto-merge, so the changelog pull request merges itself:
   `gh api -X PATCH repos/Lutherwaves/builder -F allow_auto_merge=true`.
2. Create `RELEASE_TOKEN` as "The release token" says.
3. Push the seed tag `v2.1.0` on the 2.1.0 release commit:
   `git tag -s v2.1.0 "$(git rev-list -n1 builder-v2.1.0)" -m v2.1.0 && git push origin v2.1.0`.
   The 2.1.0 commit has no `publish.yml`, so the tag starts no workflow.
4. Limit who can create `v*` tags. Settings → Rules → Rulesets → New tag
   ruleset: target `v*`, enforcement Active, rules "Restrict creations",
   "Restrict updates" and "Restrict deletions", bypass list "Repository
   admin". `RELEASE_TOKEN` acts as the owner, so the Release job still pushes
   tags.
5. Set `RELEASE_ENABLED=true` as "Turn releases on or off" says.

Then: squash-merge `feat/go` with a `feat!:` title, so the release is 3.0.0.
Do the five steps first, and merge when you can watch the run. Watch the
Release run and the Publish run to the end. Until the 3.0.0 pull request
merges, sessions offer an install that fails. If the release stalls, choose
one:

- **Merge the 3.0.0 pull request by hand**, once Publish verified the release
  and `gh release view v3.0.0 --json isDraft,isPrerelease` shows both
  `false`. `main` then names 3.0.0, and sessions install its binary.
- **Revert the `feat/go` merge.** Hold releases first ("Turn releases on or
  off"), so the revert makes no release. `main` then holds builder 2.1.0
  again, which needs no binary, and sessions stop the offer. Fix the cause,
  turn releases on, and merge `feat/go` again.

## When something fails

| What failed | What to do |
|---|---|
| Create Release says "the plugin is already X.Y.Z" | Push the seed tag (step 3 above), then re-run the Release run |
| Publish, from a runner or network hiccup | Publish the tag again: `gh workflow run publish.yml --ref vX.Y.Z` |
| Publish says the release is published without a file | It went out before its binaries and cannot take them now. Merge a fix: the next release carries them |
| Publish, from a bug in the code or the workflow | Merge the fix: the next release is published and becomes the latest. The broken one stays a pre-release |
| Create Release says "the bump PR for vX.Y.Z has not merged" | vX.Y.Z is published and verified, but `main` does not name it yet. Merge the `docs: vX.Y.Z is released` pull request (rows below). If there is none, open it as the last row says. Its merge starts the next release. A tag whose release never published does not hold releases back |
| Create Release says "tag vX.Y.Z exists at <sha>, not at the verified <sha>" | Someone pushed the tag on another commit. The job stops before it makes a draft. If no published release uses the tag, delete the tag and any draft of it (`gh api -X DELETE repos/Lutherwaves/builder/git/refs/tags/vX.Y.Z`, then `gh release delete vX.Y.Z --yes` if `gh release view vX.Y.Z` finds one), and re-run the Release run. If a published release uses the tag, it stays: open its bump pull request by hand as the last row says, and merge it. Its merge starts the next release, on the next version |
| Publish says "refusing to publish vX.Y.Z: its commit … is not on main" | The tag names a commit outside `main`. Delete the tag and its draft release, and check the tag ruleset (step 4 above). Releases come only from `main` |
| Promote changelog fails: "release vX.Y.Z is not published and verified after 60 min" | The pull request stays open without auto-merge. Fix Publish (rows above). Once `gh release view vX.Y.Z --json isDraft,isPrerelease` shows both `false`, merge the pull request as the next row says |
| Promote changelog fails: "Auto-merge is off", "has not merged in 40 min", or the pull request did not merge itself | Allow auto-merge (step 1 above). Update the pull request with `main` (`gh pr update-branch --rebase <branch>`: a merge commit fails the subject check), then merge it once its checks pass |
| Promote changelog fails: "the vX.Y.Z pull request was closed" | Reopen it and merge it as the row above says. If it is gone, open one as the next row says |
| No changelog pull request after a release | Run `.github/scripts/promote-changelog.sh X.Y.Z <date>` and `.github/scripts/bump-version.sh X.Y.Z`, and open the pull request yourself |

Tags are immutable: a bad release is followed by a fixed one, never moved.
