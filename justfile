set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

binary := "ek9s"

# Default: list recipes
default:
    @just --list

# Run the test suite
test:
    go test ./...

# Compile a binary for the current OS/arch
build:
    go build -o {{binary}} .

# Cut a release: rewrite CHANGELOG, commit, and tag. bump = auto|patch|minor|major|X.Y.Z
release bump="auto":
    #!/usr/bin/env bash
    set -euo pipefail

    if [[ -n "$(git status --porcelain)" ]]; then
        echo "error: working tree is dirty; commit or stash first" >&2
        exit 1
    fi

    last_tag="$(git tag --list 'v*' --sort=-v:refname | head -n1)"
    last_tag="${last_tag:-v0.0.0}"
    IFS='.' read -r major minor patch <<<"${last_tag#v}"

    bump='{{bump}}'
    if [[ "$bump" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        new="$bump"
    else
        if [[ "$bump" == "auto" ]]; then
            unreleased="$(awk '/^## \[Unreleased\]/{f=1;next} /^## \[/{f=0} f' CHANGELOG.md)"
            has_section() { awk -v s="### $1" '$0==s{f=1;next} /^### /{f=0} f && NF' <<<"$unreleased" | grep -q .; }
            if has_section Breaking; then bump=major
            elif has_section Added || has_section Changed || has_section Removed; then bump=minor
            elif has_section Fixed || has_section Security || has_section Deprecated; then bump=patch
            else
                echo "error: [Unreleased] has no entries; nothing to release" >&2
                exit 1
            fi
            echo "inferred bump: $bump"
        fi
        case "$bump" in
            major) new="$((major+1)).0.0" ;;
            minor) new="${major}.$((minor+1)).0" ;;
            patch) new="${major}.${minor}.$((patch+1))" ;;
            *) echo "error: bump must be auto|major|minor|patch|X.Y.Z" >&2; exit 1 ;;
        esac
    fi

    tag="v${new}"
    if git rev-parse "$tag" >/dev/null 2>&1; then
        echo "error: tag $tag already exists" >&2
        exit 1
    fi

    today="$(date +%Y-%m-%d)"
    echo "releasing $tag ($today)"

    python3 -c 'import re,sys,pathlib; new,today=sys.argv[1],sys.argv[2]; p=pathlib.Path("CHANGELOG.md"); s="## [Unreleased]\n\n### Breaking\n\n### Added\n\n### Changed\n\n### Removed\n\n### Fixed\n\n"; t,n=re.subn(r"## \[Unreleased\]\n", s+f"## [{new}] - {today}\n", p.read_text(),count=1); sys.exit("CHANGELOG.md: no [Unreleased] heading") if n!=1 else p.write_text(t)' "$new" "$today"

    git add CHANGELOG.md
    git commit -m "chore(release): ${tag}"
    git tag -a "$tag" -m "Release ${tag}"
    echo "done. push with: git push && git push origin ${tag}"
