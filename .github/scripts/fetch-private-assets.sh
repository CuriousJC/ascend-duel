#!/usr/bin/env bash
#
# Fetches the audio bundle into privateassets/audio/ so the release build can embed it.
#
# Those files are not in git — they are synced from a private bucket instead. See
# privateassets/README.md.
#
# **One script called from both build jobs, rather than the same twenty lines pasted twice.**
# Two copies of a credential exchange is two copies to keep in step, and the one that drifts is
# whichever platform was released less recently.
#
# **No marketplace action, which is why the OIDC exchange is written out by hand.** This repo
# runs first-party actions only, and a release workflow is the last place to hand execution to
# code nobody here reviews. `aws-actions/configure-aws-credentials` would do what the twelve
# lines below do; the twelve lines are the price of that rule and they are not complicated.
#
# **No long-lived keys anywhere.** GitHub mints a short-lived OIDC token for this run, STS
# exchanges it for session credentials that expire in fifteen minutes, and the role is assumable
# only by a workflow running in this repository. Nothing is stored in repository secrets, so there is
# nothing to leak, rotate or revoke.
#
# **Doing nothing is a success.** With no bucket configured the script says so and exits clean,
# which is what keeps a fork buildable and what keeps a release cuttable before the bucket
# exists. What stops that from becoming a release that silently ships quiet is the step *after*
# this one: `go run ./tools/privateassets -require` fails when the manifest names a file the
# directory does not hold.
#
# **Both steps are gated on SOUNDS_BUCKET in the workflow**, so the guard arrives together with
# the sync. This check stays anyway: the variable is set in one place and this script is called
# from two, and a step that assumes its gate is a step assuming something it cannot see.

set -euo pipefail

: "${SOUNDS_BUCKET:=}"
: "${SOUNDS_ROLE_ARN:=}"
: "${AWS_REGION:=us-east-1}"

if [ -z "$SOUNDS_BUCKET" ] || [ -z "$SOUNDS_ROLE_ARN" ]; then
  echo "no private asset bucket configured (SOUNDS_BUCKET / SOUNDS_ROLE_ARN); building without sound effects"
  exit 0
fi

# The bundle prefix comes out of the committed manifest rather than out of the workflow, so
# adopting a new set of sounds is one edit in a file that is reviewed, and the version the
# release fetched is the version the release recorded.
bundle="$(jq -r '.Bundle' privateassets/audio/manifest.json)"
if [ -z "$bundle" ] || [ "$bundle" = "null" ]; then
  echo "::error::privateassets/audio/manifest.json names no Bundle"
  exit 1
fi

if [ -z "${ACTIONS_ID_TOKEN_REQUEST_URL:-}" ]; then
  echo "::error::no OIDC token available; this job needs 'permissions: id-token: write'"
  exit 1
fi

# `audience=sts.amazonaws.com` has to match the audience condition on the role's trust policy.
token="$(curl -sSf \
  -H "Authorization: bearer ${ACTIONS_ID_TOKEN_REQUEST_TOKEN}" \
  "${ACTIONS_ID_TOKEN_REQUEST_URL}&audience=sts.amazonaws.com" | jq -r '.value')"

creds="$(aws sts assume-role-with-web-identity \
  --role-arn "$SOUNDS_ROLE_ARN" \
  --role-session-name "ascend-duel-release" \
  --web-identity-token "$token" \
  --duration-seconds 900 \
  --query 'Credentials.[AccessKeyId,SecretAccessKey,SessionToken]' \
  --output text)"

read -r key secret session <<<"$creds"

# Masked before they are used, so a later step that prints its environment cannot put them in a
# log. They expire in fifteen minutes regardless; masking is the cheap half of the belt.
echo "::add-mask::$secret"
echo "::add-mask::$session"

export AWS_ACCESS_KEY_ID="$key"
export AWS_SECRET_ACCESS_KEY="$secret"
export AWS_SESSION_TOKEN="$session"
export AWS_REGION

# `--exact-timestamps` because a synced file that differs only in mtime is still the wrong file
# to skip: the hash check downstream is what decides, and it wants the bytes actually fetched.
echo "syncing bundle ${bundle} from s3://${SOUNDS_BUCKET}/sounds/${bundle}/"
aws s3 sync "s3://${SOUNDS_BUCKET}/sounds/${bundle}/" privateassets/audio/ \
  --exact-timestamps \
  --exclude "*" \
  --include "*.ogg" \
  --include "*.wav" \
  --include "*.mp3"

# The credentials are deliberately not written to $GITHUB_ENV. They are wanted for one sync and
# nothing after it, and exporting them would leave a working set of AWS credentials in the
# environment of every later step in the job, including `go build`.
echo "sync complete"
