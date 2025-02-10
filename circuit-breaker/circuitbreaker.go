package circuit_breaker

import (
	"container/list"
	"errors"
	"log"
	"sync"
	"time"
)

type CircuitBreaker struct {
	config Config
	state  *CircuitBreakerState
	mu     sync.Mutex

	halfOpenCalls   int // Track permitted calls in half-open state
	lastFailureTime time.Time
	requests        *list.List
}

func NewCircuitBreaker(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		config:   config,
		state:    &CircuitBreakerState{state: Closed},
		requests: list.New(),
	}
}

// requestEntry stores whether a call was a failure or slow
type requestEntry struct {
	failed bool
	slow   bool
}

func (cb *CircuitBreaker) Execute(action func() error) error {
	cb.mu.Lock()

	if cb.state == nil {
		cb.mu.Unlock()
		return errors.New("circuit breaker state is not initialized")
	}

	// transition from Open → Half-Open if cooling time has passed
	if cb.state.IsOpen() && time.Since(cb.lastFailureTime) >= cb.config.WaitDurationInOpenState {
		log.Printf("%s: moving state to half-open", cb.config.Name)
		cb.state.SetState(HalfOpen)
		cb.halfOpenCalls = 0 // Reset allowed calls
	}

	if cb.state.IsOpen() {
		cb.mu.Unlock()
		return errors.New(cb.config.Name + " :circuit breaker is in opened state")
	}

	if cb.state.IsHalfOpen() {
		if cb.halfOpenCalls >= cb.config.PermittedNumberOfCallsInHalfOpenState {
			cb.mu.Unlock()
			return errors.New(cb.config.Name + " :circuit breaker half-open, no more calls allowed")
		}
		cb.halfOpenCalls++ // Count half-open calls
	}

	cb.mu.Unlock()

	start := time.Now()
	err := action()
	duration := time.Since(start)

	// Determine if call was slow
	isSlow := duration >= cb.config.SlowCallDurationThreshold

	// Record success/failure/slow call status
	cb.recordResult(err, isSlow)

	return err
}

func (cb *CircuitBreaker) recordResult(err error, isSlow bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.addRequest(err != nil, isSlow)

	failureRate := cb.getFailureRate()
	slowCallRate := cb.getSlowCallRate()

	switch cb.state.state {
	case Closed:
		if cb.requests.Len() >= cb.config.MinimumNumberOfCalls &&
			(failureRate >= cb.config.FailureRateThreshold || slowCallRate >= cb.config.SlowCallRateThreshold) {
			cb.state.SetState(Open)
			cb.lastFailureTime = time.Now()
			log.Printf("%s: moving state to open", cb.config.Name)
		}

	case Open:
		if time.Since(cb.lastFailureTime) >= cb.config.WaitDurationInOpenState {
			cb.state.SetState(HalfOpen)
			cb.halfOpenCalls = 0 // ✅ Reset half-open call count
			log.Printf("%s: moving state to half-open", cb.config.Name)
		}

	case HalfOpen:
		if err != nil || isSlow { //Failure or slow call moves back to Open
			cb.state.SetState(Open) // Move back to Open if a failure occurs
			cb.halfOpenCalls = 0    // Reset half-open call count
			cb.lastFailureTime = time.Now()
		} else {
			//cb.halfOpenCalls++ // Count half-open calls when calls are being made, during decision it's quite late.
			if cb.halfOpenCalls >= cb.config.PermittedNumberOfCallsInHalfOpenState {
				// ✅ Move to CLOSED if all calls pass in Half-Open state
				log.Printf("%s: moving state to closed", cb.config.Name)
				cb.state.SetState(Closed)
				cb.halfOpenCalls = 0 //Reset half open calls
				//cb.requests.Init()   // Let old failures naturally expire instead of wiping the history ✅
			}
		}
	}
}

// addRequest tracks recent failures in a sliding window
func (cb *CircuitBreaker) addRequest(failed, slow bool) {
	// Ensure cb.requests is initialized
	if cb.requests == nil {
		cb.requests = list.New()
	}

	// Remove the oldest request if the window size exceeds the limit
	if cb.requests.Len() >= cb.config.SlidingWindowSize {
		if front := cb.requests.Front(); front != nil {
			cb.requests.Remove(front)
		}
	}

	// Store boolean failure status in the sliding window
	cb.requests.PushBack(requestEntry{failed: failed, slow: slow})
}

// getFailureRate calculates the failure percentage
func (cb *CircuitBreaker) getFailureRate() float64 {
	if cb.requests.Len() == 0 {
		return 0.0
	}
	failures := 0
	for e := cb.requests.Front(); e != nil; e = e.Next() {
		if e.Value.(requestEntry).failed {
			failures++
		}
	}
	return (float64(failures) / float64(cb.requests.Len())) * 100
}

// getSlowCallRate calculates the slow call percentage
func (cb *CircuitBreaker) getSlowCallRate() float64 {
	if cb.requests.Len() == 0 {
		return 0.0
	}
	slowCalls := 0
	for e := cb.requests.Front(); e != nil; e = e.Next() {
		entry := e.Value.(requestEntry)
		//failed calls are not considered slow, as they are already counted in getFailureRate
		if entry.slow && !entry.failed {
			slowCalls++
		}
	}
	return (float64(slowCalls) / float64(cb.requests.Len())) * 100
}
