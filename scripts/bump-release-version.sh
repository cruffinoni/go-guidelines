#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/bump-release-version.sh <patch|minor|major|stable> [--pre LABEL] [--commit] [--push]

Dry-run is the default. This tool manages Go release tags only; it never edits
source files, Python package metadata, or changelogs.

Examples:
  scripts/bump-release-version.sh patch --pre alpha
  scripts/bump-release-version.sh patch --pre alpha --commit --push
  scripts/bump-release-version.sh stable --commit --push
EOF
}

command_name=""
pre_label=""
apply=false
push_remote=false
while (($#)); do
  case "$1" in
    patch|minor|major|stable)
      if [[ -n "$command_name" ]]; then
        echo "error: specify exactly one version command" >&2
        exit 2
      fi
      command_name=$1
      shift
      ;;
    --pre)
      if (($# < 2)) || [[ "$2" == --* ]]; then
        echo "error: --pre requires a label" >&2
        exit 2
      fi
      pre_label=$2
      shift 2
      ;;
    --commit)
      apply=true
      shift
      ;;
    --push)
      push_remote=true
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$command_name" ]]; then
  echo "error: a version command is required" >&2
  usage >&2
  exit 2
fi
if [[ "$command_name" == stable && -n "$pre_label" ]]; then
  echo "error: stable cannot be combined with --pre" >&2
  exit 2
fi
if [[ "$command_name" != stable && -n "$pre_label" && ! "$pre_label" =~ ^[A-Za-z][A-Za-z0-9-]*$ ]]; then
  echo "error: prerelease labels must start with a letter and contain only letters, digits, or hyphens" >&2
  exit 2
fi
if [[ "$push_remote" == true && "$apply" != true ]]; then
  echo "error: --push requires --commit" >&2
  exit 2
fi
if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: run this command inside a Git working tree" >&2
  exit 2
fi

tag_list=$(git tag --list 'v*')
remote_tag_list=""
if [[ "$push_remote" == true ]]; then
  if ! git remote get-url origin >/dev/null 2>&1; then
    echo "error: --push requires an origin remote" >&2
    exit 1
  fi
  remote_output=$(git ls-remote --tags origin 'refs/tags/v*') || {
    echo "error: could not read tags from origin; refusing to push" >&2
    exit 1
  }
  while read -r _ ref; do
    [[ -z "${ref:-}" || "$ref" == *'^{}' ]] && continue
    tag=${ref#refs/tags/}
    remote_tag_list+="${tag}"$'\n'
    tag_list+=$'\n'"${tag}"
  done <<<"$remote_output"
fi

major= minor= patch= label= counter=
parse_version() {
  local version=${1#v}
  if [[ "$version" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-([A-Za-z][A-Za-z0-9-]*)\.([1-9][0-9]*))?$ ]]; then
    major=${BASH_REMATCH[1]}
    minor=${BASH_REMATCH[2]}
    patch=${BASH_REMATCH[3]}
    label=${BASH_REMATCH[5]:-}
    counter=${BASH_REMATCH[6]:-}
    return 0
  fi
  return 1
}

version_compare() {
  local left=$1 right=$2 lmajor lminor lpatch llabel lcounter rmajor rminor rpatch rlabel rcounter
  parse_version "$left" || return 2
  lmajor=$major; lminor=$minor; lpatch=$patch; llabel=$label; lcounter=$counter
  parse_version "$right" || return 2
  rmajor=$major; rminor=$minor; rpatch=$patch; rlabel=$label; rcounter=$counter
  for pair in "$lmajor:$rmajor" "$lminor:$rminor" "$lpatch:$rpatch"; do
    local a=${pair%%:*} b=${pair#*:}
    ((10#$a > 10#$b)) && { echo 1; return; }
    ((10#$a < 10#$b)) && { echo -1; return; }
  done
  if [[ -z "$llabel" && -n "$rlabel" ]]; then echo 1; return; fi
  if [[ -n "$llabel" && -z "$rlabel" ]]; then echo -1; return; fi
  if [[ "$llabel" != "$rlabel" ]]; then
    [[ "$llabel" > "$rlabel" ]] && echo 1 || echo -1
    return
  fi
  if [[ -z "$llabel" ]]; then echo 0; return; fi
  ((10#$lcounter > 10#$rcounter)) && { echo 1; return; }
  ((10#$lcounter < 10#$rcounter)) && { echo -1; return; }
  echo 0
}

latest_tag=""
latest_stable=""
while IFS= read -r tag; do
  parse_version "$tag" || continue
  if [[ -z "$latest_tag" ]] || [[ "$(version_compare "$tag" "$latest_tag")" == 1 ]]; then
    latest_tag=$tag
  fi
  if [[ "$tag" != *-* ]] && { [[ -z "$latest_stable" ]] || [[ "$(version_compare "$tag" "$latest_stable")" == 1 ]]; }; then
    latest_stable=$tag
  fi
done <<<"$tag_list"

new_tag=""
if [[ "$command_name" == stable ]]; then
  if [[ -z "$latest_tag" || "$latest_tag" != *-* ]]; then
    echo "error: stable promotion requires the latest tag to be a prerelease" >&2
    exit 1
  fi
  parse_version "$latest_tag"
  new_tag="v${major}.${minor}.${patch}"
elif [[ -n "$pre_label" ]]; then
  if [[ -n "$latest_tag" && "$latest_tag" == *-* ]]; then
    parse_version "$latest_tag"
    base="${major}.${minor}.${patch}"
    if [[ "$label" == "$pre_label" ]]; then
      next_counter=$((10#$counter + 1))
    else
      next_counter=1
    fi
  else
    base=${latest_stable#v}
    if [[ -z "$latest_stable" ]]; then base=0.0.0; fi
    IFS=. read -r major minor patch <<<"$base"
    case "$command_name" in
      patch) patch=$((patch + 1)) ;;
      minor) minor=$((minor + 1)); patch=0 ;;
      major) major=$((major + 1)); minor=0; patch=0 ;;
    esac
    base="${major}.${minor}.${patch}"
    next_counter=1
  fi
  new_tag="v${base}-${pre_label}.${next_counter}"
else
  if [[ -n "$latest_tag" ]]; then
    parse_version "$latest_tag"
  else
    major=0; minor=0; patch=0
  fi
  case "$command_name" in
    patch) patch=$((patch + 1)) ;;
    minor) minor=$((minor + 1)); patch=0 ;;
    major) major=$((major + 1)); minor=0; patch=0 ;;
  esac
  new_tag="v${major}.${minor}.${patch}"
fi

if ! parse_version "$new_tag"; then
  echo "error: internal calculation produced invalid version: $new_tag" >&2
  exit 2
fi
if [[ -n "$latest_tag" && "$(version_compare "$new_tag" "$latest_tag")" != 1 ]]; then
  echo "error: proposed tag $new_tag is not greater than latest tag $latest_tag" >&2
  exit 1
fi
if git rev-parse -q --verify "refs/tags/$new_tag" >/dev/null; then
  echo "error: tag already exists locally: $new_tag" >&2
  exit 1
fi
if [[ "$push_remote" == true ]]; then
  while IFS= read -r remote_tag; do
    if [[ "$remote_tag" == "$new_tag" ]]; then
      echo "error: tag already exists on origin: $new_tag" >&2
      exit 1
    fi
  done <<<"$remote_tag_list"
fi

printf 'Current tag: %s\n' "${latest_tag:-none}"
printf 'Proposed tag: %s\n' "$new_tag"
if [[ "$new_tag" == *-* ]]; then
  printf 'Classification: prerelease\n'
else
  printf 'Classification: stable\n'
fi
if [[ "$apply" != true ]]; then
  printf 'Dry run only; pass --commit to create the annotated tag.\n'
  exit 0
fi
if [[ -n "$(git status --porcelain --untracked-files=normal)" ]]; then
  echo "error: working tree has changes; commit or stash them before tagging" >&2
  exit 1
fi

git tag -a "$new_tag" -m "Release $new_tag"
printf 'Created annotated tag %s at HEAD.\n' "$new_tag"
if [[ "$push_remote" == true ]]; then
  git push origin HEAD
  git push origin "$new_tag"
  printf 'Pushed HEAD and %s to origin.\n' "$new_tag"
fi
