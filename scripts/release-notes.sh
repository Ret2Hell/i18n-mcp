#!/bin/sh
set -eu

tag=${1:-${GITHUB_REF_NAME:-}}
if [ -z "$tag" ]; then
  tag=$(git describe --tags --exact-match 2>/dev/null || true)
fi
if [ -z "$tag" ]; then
  echo "usage: $0 <tag>" >&2
  exit 1
fi

version=${tag#v}
date=$(git log -1 --format=%cs "$tag" 2>/dev/null || date -u '+%Y-%m-%d')
previous_tag=$(git describe --tags --abbrev=0 "${tag}^" 2>/dev/null || true)

if [ -n "$previous_tag" ]; then
  range="${previous_tag}..${tag}"
else
  range="$tag"
fi

printf '%s (%s)\n\n' "$version" "$date"

git log --reverse --format='%H%x1f%s' "$range" | awk -F '\037' '
function short(hash) {
  return substr(hash, 1, 7)
}

function clean(message, prefix, rest, scope) {
  if (match(message, /^(feat|fix|docs|doc|perf|refactor|build|ci|chore|test|style)(\([^)]*\))?!?:[[:space:]]*/)) {
    prefix = substr(message, RSTART, RLENGTH)
    rest = substr(message, RLENGTH + 1)

    if (match(prefix, /\([^)]*\)/)) {
      scope = substr(prefix, RSTART + 1, RLENGTH - 2)
      if (scope != "") {
        return scope ": " rest
      }
    }

    return rest
  }

  return message
}

function category(message) {
  if (message ~ /^feat(\([^)]*\))?!?:/) {
    return "features"
  }
  if (message ~ /^(add|added|implement|implemented|expose|exposed)[[:space:]:]/) {
    return "features"
  }
  if (message ~ /^fix(\([^)]*\))?!?:/) {
    return "fixes"
  }
  if (message ~ /^(fixed|bugfix)[[:space:]:]/) {
    return "fixes"
  }
  if (message ~ /^(docs?|doc)(\([^)]*\))?:/) {
    return "docs"
  }
  if (message ~ /^document(ed)?[[:space:]:]/) {
    return "docs"
  }

  return "other"
}

function add(section, message, hash) {
  entries[section] = entries[section] "- " clean(message) " (" short(hash) ")\n"
}

$2 ~ /^Merge / { next }
$2 ~ /^bump version( to)? / { next }
$2 ~ /^chore(\([^)]*\))?: bump version/ { next }
$2 ~ /^ci(\([^)]*\))?:/ { next }
$2 ~ /^build(\([^)]*\))?:/ { next }
$2 ~ /^test(\([^)]*\))?:/ { next }
$2 ~ /^style(\([^)]*\))?:/ { next }
{
  add(category($2), $2, $1)
}

END {
  printed = 0

  if (entries["features"] != "") {
    print "### Features"
    print entries["features"]
    printed = 1
  }
  if (entries["fixes"] != "") {
    print "### Bug Fixes"
    print entries["fixes"]
    printed = 1
  }
  if (entries["docs"] != "") {
    print "### Documentation"
    print entries["docs"]
    printed = 1
  }
  if (entries["other"] != "") {
    print "### Other Changes"
    print entries["other"]
    printed = 1
  }

  if (!printed) {
    print "No user-facing changes in this release."
  }
}
'
