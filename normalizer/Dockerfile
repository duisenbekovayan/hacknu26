FROM golang:1.22-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/normalizer-service ./normalizer/cmd/normalizer

FROM debian:bookworm-slim
WORKDIR /app

RUN useradd -u 10001 -m appuser
COPY --from=build /out/normalizer-service /app/normalizer-service

USER appuser
ENTRYPOINT ["/app/normalizer-service"]
