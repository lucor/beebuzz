#!/usr/bin/env bash
set -euo pipefail

# --- Guards --------------------------------------------------------------

BRANCH=$(git branch --show-current)
if [ "$BRANCH" != "main" ]; then
  echo "Error: must be on main branch (currently on '$BRANCH')" >&2
  exit 1
fi

if ! git diff-index --quiet HEAD --; then
  echo "Error: working tree has uncommitted changes" >&2
  exit 1
fi

# Fetch latest main so preview is accurate
git fetch origin main
LOCAL=$(git rev-parse main)
REMOTE=$(git rev-parse origin/main)
if [ "$LOCAL" != "$REMOTE" ]; then
  echo "Error: local main is behind origin/main. Run: git pull origin main" >&2
  exit 1
fi

# --- Compute tag ---------------------------------------------------------

SHORT_SHA=$(git rev-parse --short HEAD)
TAG="beebuzz@${SHORT_SHA}"

if git rev-parse "$TAG" >/dev/null 2>&1; then
  echo "Error: tag $TAG already exists" >&2
  exit 1
fi

# --- Preview -------------------------------------------------------------

PREV=$(git tag --list 'beebuzz@*' --sort=-creatordate | sed -n '1p')

echo ""
echo "================================"
echo "  BeeBuzz Server Release Preview"
echo "================================"
echo ""
echo "  Tag:        ${TAG}"
echo "  Commit:     ${LOCAL}"
echo "  Subject:    $(git log -1 --pretty=%s)"
echo "  Author:     $(git log -1 --pretty=%an)"
echo "  Date:       $(git log -1 --pretty=%ci)"
echo ""

if [ -n "$PREV" ]; then
  COUNT=$(git rev-list --count "${PREV}..HEAD")
  echo "  Since last: ${PREV}"
  echo "  Commits:    ${COUNT}"
  echo ""
  echo "  Changes:"
  git log --oneline --no-decorate "${PREV}..HEAD" | sed 's/^/    /'
else
  echo "  (No previous beebuzz@ tag found)"
fi

echo ""
echo "================================"
echo ""

# --- Confirmation --------------------------------------------------------

read -r -p "Create and push tag ${TAG}? [y/N] " CONFIRM
case "$CONFIRM" in
  [yY]|[yY][eE][sS]) ;;
  *) echo "Aborted."; exit 0 ;;
esac

# --- Release -------------------------------------------------------------

git tag "$TAG"
git push origin "$TAG"
echo ""
echo "Released ${TAG}"
