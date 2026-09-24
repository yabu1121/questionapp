MYSQL_DSN ?= mysql://root:password@tcp(127.0.0.1:3306)/jev?multiStatements=true

ENV_FILE ?= .env
API_BIN ?= /tmp/jev-api
API_PID_FILE ?= /tmp/jev-api.pid

ifneq (,$(wildcard $(ENV_FILE)))
include $(ENV_FILE)
export DB_DSN
endif

.PHONY: up db-down db-logs migrate-down migrate-version api

up:
	docker compose up -d --wait mysql
	migrate -path db/migrations -database '$(MYSQL_DSN)' up

db-down:
	docker compose down

db-logs:
	docker compose logs -f mysql

migrate-down:
	migrate -path db/migrations -database '$(MYSQL_DSN)' down 1

migrate-version:
	migrate -path db/migrations -database '$(MYSQL_DSN)' version

api:
	@test -n "$(DB_DSN)" || (echo "DB_DSN is not set in $(ENV_FILE)" >&2; exit 1)
	@if [ -f "$(API_PID_FILE)" ]; then \
		pid="$$(cat "$(API_PID_FILE)")"; \
		if kill -0 "$$pid" 2>/dev/null; then \
			echo "stopping previous API server (PID $$pid)"; \
			kill "$$pid"; \
			while kill -0 "$$pid" 2>/dev/null; do sleep 0.1; done; \
		fi; \
		rm -f "$(API_PID_FILE)"; \
	fi
	@go build -o "$(API_BIN)" ./cmd/api
	@set -e; \
	"$(API_BIN)" & pid=$$!; \
	echo "$$pid" > "$(API_PID_FILE)"; \
	echo "API server started (PID $$pid)"; \
	trap 'kill "$$pid" 2>/dev/null || true; rm -f "$(API_PID_FILE)"' EXIT INT TERM; \
	wait "$$pid"
