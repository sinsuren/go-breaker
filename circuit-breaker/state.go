package circuit_breaker

import "sync"

type State string

const (
	Closed   State = "CLOSED"
	Open     State = "OPEN"
	HalfOpen State = "HALF_OPEN"
)

type CircuitBreakerState struct {
	state State
	mu    sync.Mutex
}

func (s *CircuitBreakerState) SetState(newState State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state = newState
}

func (s *CircuitBreakerState) IsOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == Open
}

func (s *CircuitBreakerState) IsHalfOpen() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == HalfOpen
}

func (s *CircuitBreakerState) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state == Closed
}
