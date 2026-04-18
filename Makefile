SHELL := /bin/bash
ENV_FILE := .env

.PHONY: run
run:
	@if [ -f $(ENV_FILE) ]; then \
		set -a; source $(ENV_FILE); set +a; \
	fi; \
	reflex -c reflex.conf