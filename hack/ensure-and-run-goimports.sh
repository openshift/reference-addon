#!/bin/bash
set -euo pipefail

# this script ensures that the `goimports` dependency is present
# and then executes goimport passing all arguments forward
export GOFLAGS=""
make -s goimports
.cache/dependencies/bin/goimports -local github.com/openshift/reference-addon -w -l "$@"
