FROM golang:1.22-bookworm AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/simulator ./simulators/cmd/simulator

FROM debian:bookworm-slim
WORKDIR /app

RUN useradd -u 10001 -m appuser
COPY --from=build /out/simulator /app/simulator

USER appuser
ENV RABBITMQ_URL=""
# Флаги можно переопределить в compose: command: ["-train", "X", "-interval", "500ms"]
ENTRYPOINT ["/app/simulator"]
CMD ["-interval", "1s", "-train", "LOC-DEMO-001"]
