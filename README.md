# Versioner

Versioner calculates versions from Git history, branch strategies, and a JSON
configuration. The same rules format new versions and recognize existing tags.

Each strategy defines its version format. Rendering and tag parsing share that
format, the template layout, placeholder definitions, and metadata policy.

## Requirements and usage

Go 1.27.3 or newer and Git are required.

Fetch the refs you want Versioner to see before running it:

```sh
git fetch --all --tags --prune
go build -o versioner ./cmd
./versioner
```

On Windows the executable can be built as `versioner.exe`. Run it from the
repository whose version should be calculated. Successful output is one version
on stdout; errors go to stderr with a nonzero exit code. Versioner does not fetch,
change refs, or create tags.

## Default rules

Without a config file, Versioner uses these rules. The checked-in
`versioner.config.json` contains the same defaults.

```json
{
  "rules": [
    { "pattern": "^main$", "strategy": "master", "prefix": "", "suffix": "" },
    { "pattern": "^master$", "strategy": "master", "prefix": "", "suffix": "" },
    { "pattern": "^release\\/(\\d+)\\.(\\d+)\\.0$", "strategy": "release", "prefix": "", "suffix": "" },
    { "pattern": "^develop$", "strategy": "develop", "prefix": "", "suffix": "" },
    { "pattern": "^feature\\/.*$", "strategy": "feature", "prefix": "", "suffix": "" },
    { "pattern": "^hotfix\\/.*$", "strategy": "hotfix", "prefix": "", "suffix": "" }
  ]
}
```

| Branch | Strategy | Default output |
| --- | --- | --- |
| main / master | master | X.Y.Z |
| release/X.Y.0 | release | X.Y.0-rcN |
| develop | develop | X.Y.0-betaN |
| feature/* | feature | X.Y.0-alphaN |
| hotfix/* | hotfix | X.Y.Z |

- Main/master uses the nearest stable base and increments its patch when HEAD
  is ahead of that base.
- Hotfix always increments the stable base's patch.
- Develop and feature target the minor after the highest
  `origin/release/X.Y.0` branch. Without one, they target the minor after the
  stable base.
- Release takes X.Y from its branch name and always includes `rcN`, including
  `rc0` at zero distance.
- N counts commits after the nearest matching tag. It is not added to the
  counter written in that tag. A new tag starts a new distance count.
- With no matching stable tag, the base is `0.1.0`. With no matching reference
  tag, N counts all commits reachable from HEAD.

The defaults omit commit SHA metadata. The `rcN`, `betaN` and `alphaN` spelling
is preserved; the tool does not impose a strict SemVer policy on custom output.

## Configuration

The default config path is `versioner.config.json` in the current directory.
`VERSIONER_CONFIG` overrides the **file path**, not the JSON contents.

| Environment variable | Meaning |
| --- | --- |
| VERSIONER_CONFIG | Config file path. A missing file uses the built-in defaults. |
| VERSIONER_BRANCH | Explicit short branch name, such as develop or release/1.3.0. Overrides branch detection without checking out another branch. |
| VERSIONER_MERGE | When exactly true, main/master adopts the highest active release branch's X.Y.0 version. |

Malformed JSON, unreadable files, invalid regular expressions, and unknown
strategies are errors. They do not silently select another configuration.
An empty rules list uses all built-in rules.

`pattern` matches a branch name. Rules are evaluated in file order by default.
Set `"matching": "longest"` at the top level to choose the longest matching
substring; ties retain file order. If no rule matches a branch, its built-in
rule is selected and included in the tag-reading rules. Unsupported branches
produce an error.

### Prefixes, suffixes, and placeholders

The strategy creates a canonical version. Prefix and suffix templates surround
that version.

| Suffix setting | Behavior |
| --- | --- |
| `"suffix": ""` | Remove the strategy's build metadata and append nothing. |
| `"suffix": "+{sha}"` | Replace metadata with the short SHA. |
| `"suffix": "-{sha}"` | Append the SHA using a dash. |
| Missing or null suffix | Preserve the strategy's metadata: +SHA for develop/feature, +gSHA for release/hotfix, none for main/master. |

Supported placeholders:

| Placeholder | Value |
| --- | --- |
| {version} | Canonical version, without metadata when a suffix is specified |
| {major}, {minor}, {patch} | Core version components |
| {pre} | Strategy prerelease text, such as rc10 or beta5 |
| {buildMeta} | Strategy metadata before suffix replacement |
| {sha} | Short commit SHA |
| {N} | Commit distance from the matching reference tag |
| {branch} | Branch name |
| {slug} | Sanitized branch name, limited to 16 characters |
| {tag} | Original reference tag name, including its configured decoration; empty if none matched |

Unknown placeholders are removed from the template. Replacement values are
inserted once: a branch name containing `{sha}` remains literal data.

### Reading tags

Tags must match a complete output layout described by an active rule: its
strategy, prefix, suffix, and placeholders. All active rules are considered
because historical tags can have been produced on other branches.

For example, a master rule with `"prefix": "app-"` and `"suffix": "-{sha}"`
recognizes `app-1.2.3-abc1234`. It extracts a stable core version of `1.2.3` and
a separate SHA. It does not treat the SHA as a version counter.

A tag that matches no rule is ignored. There is no generic SemVer fallback or
implicit `v` prefix. The same field repeated in a template must have the same
value. A captured branch must match the rule's branch pattern.

Use separators when adjacent placeholders could have different interpretations.
Ambiguous layouts, including one tag interpreted as different versions by
different rules, produce an error. Matching work is bounded to reject excessively
ambiguous inputs rather than spend unbounded time searching.

Version components and counters must fit the platform's integer range.
Unrepresentable tag values do not match; arithmetic that would overflow during
generation produces an error.

### Choosing a tag

Only tags reachable from HEAD are candidates. The reference is the matching tag
with the smallest `git rev-list refs/tags/TAG..HEAD --count` distance. Version
text, prerelease counter text, and SHA ordering do not determine which is newer.

Main/master and hotfix resolve a stable base separately from the reference used
for N. Stability is determined by the configured strategy's canonical version,
not by the decoration chosen in the suffix.

Different meanings at the same nearest distance produce an ambiguity error.
Equivalent tags are selected deterministically by tag name. Unrelated tags,
including tags on unmerged branches, cannot change the result. Git failures are
reported instead of being treated as absent tags or zero distance.

### Detached HEAD

Without `VERSIONER_BRANCH`, a detached checkout must have exactly one
config-matching tag pointing at HEAD. Versioner prints that original tag,
including its prefix and suffix. No matching tag or multiple matching tags
produce an error explaining how to use `VERSIONER_BRANCH`.

An explicit branch override uses the normal generation path. Versioner does not
guess a branch from remote refs.

## Compatibility notes

- Default output no longer contains SHA metadata. Add `"suffix": "+{sha}"` to
  the desired rules to include it.
- Existing tags must match the active configuration. Previously accepted
  `v1.2.3` or metadata-bearing tags are ignored unless a rule describes them.
  Keep rules for historical formats if those tags should remain candidates.
- Missing config uses the same rendering path as an explicit default file.
- Invalid config now stops execution.
- Main/master patch calculation uses distance from the stable base, even when
  a nearer prerelease tag exists.
- Go 1.27.3 is the minimum version for source builds.

## Development

The CLI registers the strategy selected by the config with `Engine.Register`
and runs it through `Engine.Resolve`. Custom strategies still only need to
implement `Strategy.NextVersion`. Compiled rules retain their configuration at
compile time; call `Compile` again after changing a config used through the Go API.

```sh
go test -count=1 ./...
go vet ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
```

Tests create isolated temporary Git repositories. They cover config round trips,
tag reachability and ambiguity, stable bases, detached checkouts, counters,
overflow, and CLI output.

PR checks and release checks run on Linux/amd64, Windows/amd64, and macOS/arm64.
The release workflow waits for those checks before building and publishing.
Actions are pinned to commit IDs; only the publication job has repository write
permission.
