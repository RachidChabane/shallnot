#!/usr/bin/env bash
# Copy the password-policy project into the run's working directory, with a
# virtual environment of its own: the run's HOME is not the author's, so
# nothing installed per user is reachable from inside it.
set -euo pipefail
cp -R "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_project/." .
python3 -m venv .venv
.venv/bin/pip install --quiet --disable-pip-version-check pytest
