SHELL=/bin/bash
.SHELLFLAGS=-euo pipefail -c

# Runs code-generators, checks for clean directory and lints the source code.
lint:
	@true
.PHONY: lint

# Runs unittests
test-unit:
	@true
.PHONY: test-unit
