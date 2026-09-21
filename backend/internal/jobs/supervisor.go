package jobs

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

var ErrMutationBusy = errors.New("another library mutation job is already running")
var ErrSupervisorStopping = errors.New("job supervisor is stopping")

type State struct {
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
	Error      string    `json:"error,omitempty"`
}

type Supervisor struct {
	ctx context.Context

	mu       sync.RWMutex
	stopping bool
	states   map[string]State

	mutation chan struct{}
	wg       sync.WaitGroup
}

func New(ctx context.Context) *Supervisor {
	return &Supervisor{
		ctx:      ctx,
		states:   make(map[string]State),
		mutation: make(chan struct{}, 1),
	}
}

func (s *Supervisor) AcquireMutation(ctx context.Context, name string) (func(error), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := s.ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopping {
		return nil, ErrSupervisorStopping
	}

	select {
	case s.mutation <- struct{}{}:
	default:
		return nil, ErrMutationBusy
	}

	s.wg.Add(1)
	s.states[name] = State{
		Name:      name,
		Status:    "running",
		StartedAt: time.Now().UTC(),
	}

	var once sync.Once
	finish := func(jobErr error) {
		once.Do(func() {
			s.mu.Lock()
			state := s.states[name]
			state.Status = "completed"
			state.FinishedAt = time.Now().UTC()
			if jobErr != nil {
				state.Status = "failed"
				state.Error = jobErr.Error()
			}
			s.states[name] = state
			s.mu.Unlock()

			<-s.mutation
			s.wg.Done()
		})
	}
	return finish, nil
}

func (s *Supervisor) GoMutation(name string, fn func(context.Context) error) bool {
	finish, err := s.AcquireMutation(s.ctx, name)
	if err != nil {
		return false
	}
	go func() {
		err := fn(s.ctx)
		finish(err)
	}()
	return true
}

func (s *Supervisor) Snapshot() []State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]State, 0, len(s.states))
	for _, state := range s.states {
		out = append(out, state)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartedAt.Equal(out[j].StartedAt) {
			return out[i].Name < out[j].Name
		}
		return out[i].StartedAt.Before(out[j].StartedAt)
	})
	return out
}

func (s *Supervisor) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	s.stopping = true
	s.mu.Unlock()

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
