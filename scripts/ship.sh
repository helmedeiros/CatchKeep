#!/usr/bin/env bash
# Ship the staged change as one PR: branch from an up-to-date, green main,
# commit (local gates run in hooks), open a PR, wait for CI, squash-merge the
# exact commit CI tested, then wait for main to be green again.
#
# usage: scripts/ship.sh <branch> <conventional-commit-subject> <pr-body>
set -euo pipefail

branch=${1:?branch required}
subject=${2:?commit subject required}
body=${3:?PR body required}

die() { echo "ship: $*" >&2; exit 1; }

[[ $(git branch --show-current) == main ]] || die "must start on main"
git diff --cached --quiet && die "nothing staged"
git diff --quiet || die "unstaged changes present; stage or stash them"
[[ -z $(git ls-files --others --exclude-standard) ]] || die "untracked files present"

git fetch -q origin main
[[ $(git rev-parse HEAD) == $(git rev-parse origin/main) ]] || die "local main is not origin/main"

main_status=$(gh run list --branch main --workflow CI --limit 1 --json headSha,conclusion \
  --jq ".[] | select(.headSha == \"$(git rev-parse HEAD)\") | .conclusion")
[[ $main_status == success ]] || die "CI on main HEAD is not green (${main_status:-missing})"

npm run --silent check

git switch -q -c "$branch"
if ! git commit -q -m "$subject"; then
  # Local gates failed in the commit hooks: go back to main with the change
  # still staged so it can be fixed and shipped again.
  git switch -q main
  git branch -q -D "$branch"
  die "commit hooks failed; back on main with changes staged"
fi
head_sha=$(git rev-parse HEAD)
git push -q -u origin "$branch"

url=$(gh pr create --title "$subject" --body "$body")
echo "PR: $url"

# Wait for checks to register, then watch them to completion.
for _ in {1..30}; do gh pr checks "$url" >/dev/null 2>&1 && break; sleep 2; done
gh pr checks "$url" --watch --fail-fast --interval 5 >/dev/null || die "CI failed on $url"
# The watch can end before the aggregating required check exists, so wait
# for "Quality gates" itself to pass.
gate_state() {
  gh pr checks "$url" --json name,state --jq '.[] | select(.name == "Quality gates") | .state' 2>/dev/null
}
for _ in {1..180}; do
  case "$(gate_state)" in
    SUCCESS) break ;;
    FAILURE | CANCELLED | ERROR | TIMED_OUT | ACTION_REQUIRED) die "Quality gates failed on $url" ;;
  esac
  sleep 5
done
[[ $(gate_state) == SUCCESS ]] || die "timed out waiting for Quality gates on $url"

gh pr merge "$url" --squash --delete-branch --match-head-commit "$head_sha" >/dev/null

git switch -q main
git pull -q --ff-only origin main
merged_sha=$(git rev-parse HEAD)

run_id=""
for _ in {1..30}; do
  run_id=$(gh run list --branch main --workflow CI --limit 5 --json databaseId,headSha \
    --jq ".[] | select(.headSha == \"$merged_sha\") | .databaseId" | head -1)
  [[ -n $run_id ]] && break
  sleep 2
done
[[ -n $run_id ]] || die "no CI run found for $merged_sha on main"
gh run watch "$run_id" --exit-status --interval 5 >/dev/null || die "CI failed on main at $merged_sha"

echo "merged and green: $(git log --oneline -1)"
