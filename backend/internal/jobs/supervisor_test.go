package jobs

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMutationJobsAreExclusive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s := New(ctx)

	finish, err := s.AcquireMutation(ctx, "scan")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.AcquireMutation(ctx, "autotagger"); !errors.Is(err, ErrMutationBusy) {
		t.Fatalf("expected ErrMutationBusy, got %v", err)
	}
	finish(nil)

	finish2, err := s.AcquireMutation(ctx, "autotagger")
	if err != nil {
		t.Fatal(err)
	}
	finish2(errors.New("boom"))

	states := s.Snapshot()
	if len(states) != 2 {
		t.Fatalf("states=%d", len(states))
	}
	if states[1].Status != "failed" || states[1].Error != "boom" {
		t.Fatalf("unexpected state: %+v", states[1])
	}
}

func TestShutdownWaitsForRunningJobAndBlocksNewOnes(t *testing.T) {
	appCtx, cancel := context.WithCancel(context.Background())
	s := New(appCtx)

	started := make(chan struct{})
	release := make(chan struct{})
	if !s.GoMutation("autotagger", func(ctx context.Context) error {
		close(started)
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}) {
		t.Fatal("job was not started")
	}
	<-started

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Second)
	defer shutdownCancel()
	done := make(chan error, 1)
	go func() { done <- s.Shutdown(shutdownCtx) }()

	time.Sleep(10 * time.Millisecond)
	if _, err := s.AcquireMutation(context.Background(), "scan"); !errors.Is(err, ErrSupervisorStopping) {
		t.Fatalf("expected supervisor stopping, got %v", err)
	}

	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	cancel()
}
