#!/usr/bin/env bash
# Remove machine-identifying data from JUnit XML files: the host name, the
# project's absolute path, the home directory and the user name.
# Usage: sanitize-results.sh <project directory> <file>...
set -euo pipefail

project_dir="$1"
shift
user_name="${USER:-$(id -un)}"

PROJECT_DIR="$project_dir" HOME_DIR="$HOME" USER_NAME="$user_name" perl -pi -e '
  s/hostname="[^"]*"/hostname="localhost"/g;
  s{\Q$ENV{PROJECT_DIR}\E/?}{}g;
  s{\Q$ENV{HOME_DIR}\E}{/home/user}g;
  s{\b\Q$ENV{USER_NAME}\E\b}{user}g;
' "$@"
