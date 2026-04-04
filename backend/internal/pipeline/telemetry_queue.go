package pipeline

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"hacknu/pkg/telemetry"
)

var ErrQueueFull = errors.New("telemetry queue is full")

type storeWriter interface {
	Insert(ctx context.Context, s *telemetry.Sample) error
}

type broadcaster interface {
	Broadcast(v any)
}

type QueueOptions struct {
	Buffer  int
	Workers int
}

type TelemetryQueue struct {
	log         *slog.Logger
	writer      storeWriter
	broadcast   broadcaster
	jobs        chan queueJob
	shutdownOne sync.Once
	workersWG   sync.WaitGroup
}

type queueJob struct {
	ctx    context.Context
	sample telemetry.Sample
	done   chan error
}

func NewTelemetryQueue(log *slog.Logger, writer storeWriter, broadcast broadcaster, opts QueueOptions) *TelemetryQueue {
	if log == nil {
		log = slog.Default()
	}
	if opts.Buffer <= 0 {
		opts.Buffer = 1024
	}
	if opts.Workers <= 0 {
		opts.Workers = 4
	}

	q := &TelemetryQueue{
		log:       log,
		writer:    writer,
		broadcast: broadcast,
		jobs:      make(chan queueJob, opts.Buffer),
	}

	for i := 0; i < opts.Workers; i++ {
		q.workersWG.Add(1)
		go q.worker()
	}

	q.log.Info("telemetry queue started", "buffer", opts.Buffer, "workers", opts.Workers)
	return q
}

func (q *TelemetryQueue) Ingest(ctx context.Context, s *telemetry.Sample) error {
	if s == nil {
		return errors.New("nil sample")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	job := queueJob{
		ctx:    ctx,
		sample: *s,
		done:   make(chan error, 1),
	}

	select {
	case q.jobs <- job:
	default:
		return ErrQueueFull
	}

	select {
	case err := <-job.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *TelemetryQueue) Shutdown(ctx context.Context) error {
	q.shutdownOne.Do(func() {
		close(q.jobs)
	})

	done := make(chan struct{})
	go func() {
		q.workersWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		q.log.Info("telemetry queue stopped")
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *TelemetryQueue) worker() {
	defer q.workersWG.Done()
	for job := range q.jobs {
		err := q.writer.Insert(job.ctx, &job.sample)
		if err == nil {
			q.broadcast.Broadcast(&job.sample)
		}
		job.done <- err
		close(job.done)
	}
}
