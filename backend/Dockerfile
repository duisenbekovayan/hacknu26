FROM golang:1.22-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/backend-service ./backend/cmd/server

FROM debian:bookworm-slim
WORKDIR /app

RUN useradd -u 10001 -m appuser
COPY --from=build /out/backend-service /app/backend-service
COPY frontend /app/frontend

ENV FRONTEND_DIR=/app/frontend
EXPOSE 8080
USER appuser

ENTRYPOINT ["/app/backend-service"]
