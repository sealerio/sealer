# Making go-ipld-prime Releases

## Versioning strategy

go-ipld-prime follows **[WarpVer](https://gist.github.com/warpfork/98d2f4060c68a565e8ad18ea4814c25f)**, a form of SemVer that never bumps the major version number and uses minor version numbers to indicate degree of *changeness*: **even numbers should be easy upgrades; odd numbers may change things**. The patch version number is rarely used in this scheme.

## CHANGELOG.md

There is a CHANGELOG.md, it should be relevant and updated. Notable items in the commit history since the last release should be included. Where possible and practical, links to relevant pull requests or other issues with discussions on the items should be included.

To find the list of commits, it is recommended that you use a tool that can provide some extra metadata to help with matching commits to pull requests. [changelog-maker](https://github.com/nodejs/changelog-maker) can help with this (requires Node.js be installed and the `npx` command be available):

```
npx changelog-maker --start-ref=v0.16.0  --reverse=true --find-matching-prs=true --md=true ipld go-ipld-prime
```

Alternatively, you can use `git log` and perform mapping to pull requests manually, e.g.:

```
git log --all --graph --date-order --abbrev-commit --decorate --oneline
```

*(where `--start-ref` points to name of the previous release tag)*

### Curate and summarize

The CHANGELOG should be informative for developers wanting to know what changes may pose a risk (highlight these!) and what changes introduce features they may be interested in using.

1. Group commits to subsystem to create a two-level list. Subsections can include "Data Model", "Schemas", "Bindnode", "Selectors", "Codecs", and the meta-category of "Build" to describe changes local to the repository and not necessarily relevant to API consumers.
2. If there are breaking, or potentially breaking changes, list them under a `#### 🛠 Breaking Changes` section.
3. Otherwise, prune the list of commits down to the set of changes relevant to users, and list them under a `#### 🔦 Highlights` section.

Note that there is also a **Planned/Upcoming Changes** section near the top of the CHANGELOG.md. Update this to remove _done_ items and add other items that may be nearing completion but not yet released.

### Call-outs

Add "special thanks" call-outs to individuals who have contributed meaningful changes to the release.

## Propose a release

After updating the CHANGELOG.md entry, also bump the version number appropriately in **version.json** file so the auto-release procedure can take care of tagging for you.

Commit and propose the changes via a pull request to ipld/go-ipld-prime.

## Release

After a reasonable amount of time for feedback (usually at least a full global business day), the changes can be merged and a release tag will be created by the GitHub Actions.

Use the GitHub UI to make a [release](https://github.com/ipld/go-ipld-prime/releases), copying in the contents of the CHANGELOG.md for that release.

Drop in a note to the appropriate Matrix/Discord/Slack channel(s) for IPLD about the release.

Optional: Protocol Labs staff can send an email to shipped@protocol.ai to describe the release, these are typically well-read and appreciated.

## Checklist

Prior to opening a release proposal pull request, create an issue with the following markdown checklist to help ensure the requisite steps are taken. The issue can also be used to alert subscribed developers to the timeframe and the approximate scope of changes in the release.

```markdown
* [ ] Add new h3 to `CHANGELOG.md` under **Released Changes** with curated and subsystem-grouped list of changes and links to relevant PRs
  * [ ] Highlight any potentially breaking or disruptive changes under "🛠 Breaking Changes", including extended descriptions to help users make compatibility judgements
  * [ ] Add special-thanks call-outs to contributors making significant contributions
* [ ] Update **Planned/Upcoming Changes** section to remove completed items and add newly upcoming, but incomplete items
* [ ] Bump version number appropriately in `version.json`
* [ ] Propose release via pull request, merge after enough time for async global feedback
* [ ] Create GitHub [release](https://github.com/ipld/go-ipld-prime/releases) with the new tag, copying the new `CHANGELOG.md` contents
* [ ] Announce on relevant Discord/Matrix/Slack channel(s)
* [ ] (Optional) Announce to shipped@protocol.ai
```
