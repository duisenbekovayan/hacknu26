.PHONY: test docker-test up

up:
	docker compose up -d

test:
	go test ./... -count=1

# Если локально нет Go: поднять `make up`, затем (macOS/Windows Docker Desktop):
docker-test:
	docker run --rm -v "$$(pwd)":/src -w /src \
		-e DATABASE_URL="postgres://hacknu:hacknu@host.docker.internal:5432/locomotive?sslmode=disable" \
		golang:1.22-bookworm go test ./... -count=1
