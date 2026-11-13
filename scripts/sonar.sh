#!/usr/bin/env bash
# Runs a SonarQube analysis of the project against a local server.
#
#   SONAR_GLOBAL_TOKEN=<token> scripts/sonar.sh
#
# SONAR_HOST_URL defaults to the Windows host when run from WSL, else to
# http://host.docker.internal:9000. Coverage (coverage.out) is generated first
# when Go is installed; the scanner itself runs in Docker.
set -euo pipefail
cd "$(dirname "$0")/.."

: "${SONAR_GLOBAL_TOKEN:?set SONAR_GLOBAL_TOKEN to a SonarQube token}"

if [ -z "${SONAR_HOST_URL:-}" ]; then
  if grep -qi microsoft /proc/version 2>/dev/null; then
    SONAR_HOST_URL="http://$(ip route | awk '/default/{print $3}'):9000" # the Windows host
  else
    SONAR_HOST_URL="http://host.docker.internal:9000"
  fi
fi

if command -v go >/dev/null 2>&1; then
  go test -coverpkg=./... -coverprofile=coverage.out ./... >/dev/null
fi

echo "Analyzing against $SONAR_HOST_URL"
docker run --rm --add-host=host.docker.internal:host-gateway \
  -e SONAR_HOST_URL="$SONAR_HOST_URL" -e SONAR_TOKEN="$SONAR_GLOBAL_TOKEN" \
  -v "$PWD:/usr/src" sonarsource/sonar-scanner-cli
