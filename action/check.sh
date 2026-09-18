#!/usr/bin/env bash
# Run `shallnot check`, publish its Markdown summary as the job summary, emit
# annotations, and exit with shallnot's own exit code.
set -uo pipefail

summary="${RUNNER_TEMP:-.}/shallnot-summary.md"
json_report="${SHALLNOT_JSON_REPORT:-shallnot-report.json}"

# shellcheck disable=SC2086 # SHALLNOT_ARGS is a list of arguments by contract
shallnot check --github-annotations --markdown-out "$summary" --json-out "$json_report" ${SHALLNOT_ARGS:-}
exit_code=$?

if [ -f "$summary" ] && [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
  cat "$summary" >> "$GITHUB_STEP_SUMMARY"
fi
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "exit-code=${exit_code}" >> "$GITHUB_OUTPUT"
  echo "json-report=${json_report}" >> "$GITHUB_OUTPUT"
fi
exit "$exit_code"
