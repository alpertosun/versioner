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

- **Configuration flexibility**  
  Support a configurable mapping of branch patterns (e.g., `develop`, `feature/*`, `release/*`) via a JSON or YAML file, instead of hardcoding strategies.
