# Versioner

**Versioner** is a semantic versioning engine designed for Git-driven workflows.  
It generates consistent [SemVer](https://semver.org/) compliant versions based on the current branch, commit distance, and repository state.  
The tool is lightweight and CI/CD-friendly, intended to standardize how projects calculate versions across pipelines.

---

## Prerequisites

Before running Versioner, ensure your repository is fully synchronized:

    git fetch --all --tags --prune

- **Remote branches** are detected via `git branch -r`.  
- **Tags** are read from the local repository after fetching.  

> Without fetching, newly created tags or release branches will not be visible to the tool.

---

## Versioning Rules

Versioner determines the next version according to the **current branch type**:

- **main / master**  
  Always produces a **final release version** (`X.Y.Z`).  
  No pre-release identifiers, no build metadata.

- **release/X.Y.0**  
  Used for release candidates.  
  Format: `X.Y.0-rcN+<sha>` where `N` is the commit distance since the latest tag.

- **develop**  
  Always targets the **next minor** after the highest active release branch.  
  Format: `X.(Y+1).0-betaN+<sha>`.

- **feature/***  
  Same base as `develop`, but marked as alpha.  
  Format: `X.(Y+1).0-alphaN+<sha>`.

- **hotfix/***  
  Based on the latest stable tag, incrementing the patch number.  
  Format: `X.Y.(Z+1)+<sha>` (no pre-release identifiers).

### General Notes
- Pre-release identifiers always include a number (e.g., `alpha0`, `beta3`, `rc5`).  
- Build metadata (`+<sha>`) is appended to non-final versions.  
- Final releases (`main/master`) strip both pre-release and metadata.  
- The highest remote release branch determines the base `X.Y` used for `develop` and `feature` branches.


---


## Configuration

Versioner can be driven by a JSON config file to map branch patterns → strategies and to decorate the output with prefixes/suffixes.  
Default path is `versioner.config.json`. You can override with `VERSIONER_CONFIG=/path/to/file.json`.

Example:

    {
      "rules": [
        { "pattern": "^main$",                         "strategy": "master",  "prefix": "", "suffix": "" },
        { "pattern": "^master$",                       "strategy": "master",  "prefix": "", "suffix": "" },
        { "pattern": "^release\\/(\\d+)\\.(\\d+)\\.0$", "strategy": "release", "prefix": "", "suffix": "+{sha}" },
        { "pattern": "^develop$",                      "strategy": "develop", "prefix": "", "suffix": "+{sha}" },
        { "pattern": "^feature\\/.*$",                 "strategy": "feature", "prefix": "", "suffix": "+{sha}" },
        { "pattern": "^hotfix\\/.*$",                  "strategy": "hotfix",  "prefix": "", "suffix": "+{sha}" }
      ]
    }

You control the format: if you prefer `+{sha}` (build metadata) or `-{sha}` (extra pre-release-ish suffix), set it in config.  
Versioner always produces a canonical core X.Y.Z from strategies; prefix/suffix are pasted around it using placeholders.

Strategies:
- master (or main) → final releases (X.Y.Z)
- release → rcN pre-releases on release/X.Y.0 branches
- develop → next-minor betas
- feature → next-minor alphas
- hotfix → patch bump off the latest stable tag

Placeholders:
- {version} : Canonical base version (e.g., 1.3.0-beta5)
- {sha} : short commit SHA
- {N} : Commit distance since latest tag
- {branch} : Branch name
- {slug} : Sanitized branch name
- {major}, {minor}, {patch} : Numeric parts
- {pre} : Pre-release string (rc3, beta5, etc.)
- {buildMeta} : Existing build metadata
- {tag} : Latest known tag string

If a suffix is set, Versioner renders the canonical version without existing build metadata to prevent duplication.

Matching behavior:
- Rules are tested in file order (first match).
- Order your rules from specific to general.

Fallback:
- If no config file, Versioner uses built-in defaults.
- If config exists but no rule matches, fallback applies as well.


---

## Usage

Build and run:

    go build -o versioner main.go
    ./versioner

Output is a single semantic version string, for example:

    1.3.0-beta5+1a2b3c

---

## Design Principles

- **Remote-first resolution**: release branches are always detected from the remote (`origin/release/*`).  
- **Tag-based stability**: the latest semver-compatible tag defines the current stable version.  
- **Branch-driven semantics**: the branch name determines whether a version is final, release candidate, beta, alpha, or hotfix.  
- **CI/CD ready**: no external configuration required; deterministic output for any given Git state.


## Roadmap

Planned improvements and open areas for contribution:

- Tests → add unit tests for strategies, parsing, and rendering.
- Extended config → YAML support
- Strict mode → optional flag to enforce SemVer compliance when required.
