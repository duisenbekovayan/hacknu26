package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	amqp091 "github.com/rabbitmq/amqp091-go"

	"hacknu/pkg/rabbitmq"
	"hacknu/simulators/synth"
)

func main() {
	amqpURL := flag.String("amqp", "", "RabbitMQ (по умолчанию $RABBITMQ_URL или amqp://hacknu:hacknu@127.0.0.1:5672/)")
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

	u := strings.TrimSpace(*amqpURL)
	if u == "" {
		u = strings.TrimSpace(os.Getenv("RABBITMQ_URL"))
	}
	if u == "" {
		u = "amqp://hacknu:hacknu@127.0.0.1:5672/"
	}
	log.Printf("simulator -> RabbitMQ %s each %s (train=%s)", u, *interval, *train)
	runAMQPPublish(u, *interval, *train, g)
}

func runAMQPPublish(amqpURL string, interval time.Duration, train string, g *synth.Synthesizer) {
	for {
		conn, err := amqp091.Dial(amqpURL)
		if err != nil {
			log.Println("amqp dial:", err, "- retry in 1s")
			time.Sleep(time.Second)
			continue
		}
		ch, err := conn.Channel()
		if err != nil {
			_ = conn.Close()
			log.Println("amqp channel:", err, "- retry in 1s")
			time.Sleep(time.Second)
			continue
		}
		if err := rabbitmq.DeclareTelemetryTopology(ch); err != nil {
			_ = ch.Close()
			_ = conn.Close()
			log.Println("amqp topology:", err, "- retry in 1s")
			time.Sleep(time.Second)
			continue
		}
		log.Println("rabbitmq publisher connected")

		send := func() error {
			s := g.NextSample(train)
			body, err := json.Marshal(s)
			if err != nil {
				return err
			}
			return rabbitmq.PublishSample(ch, body)
		}

		if err := send(); err != nil {
			log.Println("amqp publish:", err)
			_ = ch.Close()
			_ = conn.Close()
			time.Sleep(time.Second)
			continue
		}
		log.Println("first sample published OK")

		tick := time.NewTicker(interval)
		for range tick.C {
			if err := send(); err != nil {
				log.Println("amqp publish:", err)
				tick.Stop()
				_ = ch.Close()
				_ = conn.Close()
				break
			}
		}
		tick.Stop()
		_ = ch.Close()
		_ = conn.Close()
		log.Println("reconnecting…")
	}
}
