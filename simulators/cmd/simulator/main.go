package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"hacknu/simulators/synth"
)

func main() {
	base := flag.String("url", "http://127.0.0.1:8080", "Базовый URL API (http/https → ws/wss для /ws/ingest)")
	interval := flag.Duration("interval", time.Second, "Интервал отправки")
	train := flag.String("train", "LOC-DEMO-001", "Идентификатор поезда")
	seed := flag.Int64("seed", 0, "Seed PRNG (0 — от времени, для повторяемого демо задайте число)")
	flag.Parse()

	var rng *rand.Rand
	if *seed == 0 {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	} else {
		rng = rand.New(rand.NewSource(*seed))
	}
	g := synth.NewSynthesizer(rng)

	wsURL := ingestWebSocketURL(*base)
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		Proxy:            http.ProxyFromEnvironment,
	}

	log.Printf("simulator -> WebSocket ingest %s each %s (train=%s)", wsURL, *interval, *train)

	for {
		conn, _, err := dialer.Dial(wsURL, nil)
		if err != nil {
			log.Println("ws dial:", err, "- retry in 1s")
			time.Sleep(time.Second)
			continue
		}
		log.Println("ws ingest connected")

		send := func() error {
			s := g.NextSample(*train)
			body, err := json.Marshal(s)
			if err != nil {
				return err
			}
			return conn.WriteMessage(websocket.TextMessage, body)
		}

		if err := send(); err != nil {
			log.Println("ws write:", err)
			_ = conn.Close()
			continue
		}
		log.Println("first sample sent OK")

		tick := time.NewTicker(*interval)
		for range tick.C {
			if err := send(); err != nil {
				log.Println("ws write:", err)
				tick.Stop()
				_ = conn.Close()
				break
			}
		}
		tick.Stop()
		_ = conn.Close()
		log.Println("reconnecting…")
	}
}

func ingestWebSocketURL(httpBase string) string {
	u, err := url.Parse(strings.TrimSpace(httpBase))
	if err != nil || u.Host == "" {
		return "ws://127.0.0.1:8080/ws/ingest"
	}
	switch u.Scheme {
	case "https":
		u.Scheme = "wss"
	case "http":
		u.Scheme = "ws"
	default:
		u.Scheme = "ws"
	}
	u.Path = "/ws/ingest"
	u.RawQuery = ""
	return u.String()
}
