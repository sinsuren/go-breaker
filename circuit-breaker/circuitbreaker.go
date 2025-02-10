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

	failureCount  int
	slowCallCount int
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
	failed        bool
	slow          bool
	executionTime time.Time
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

	// Remove old requests based on the configured strategy
	cb.cleanupOldRequests()

	// Store boolean failure status in the sliding window
	cb.requests.PushBack(requestEntry{failed: failed, slow: slow, executionTime: time.Now()})

	if failed {
		cb.failureCount++
	} else if slow {
		cb.slowCallCount++
	}
}

// cleanupOldRequests removes outdated requests based on the sliding window strategy
func (cb *CircuitBreaker) cleanupOldRequests() {
	switch cb.config.SlidingWindowType {
	case COUNT_BASED:
		cb.enforceCountBasedWindow()
	case TIME_BASED:
		cb.enforceTimeBasedWindow()
	}
}

// enforceCountBasedWindow removes oldest entries if count exceeds SlidingWindowSize
func (cb *CircuitBreaker) enforceCountBasedWindow() {
	for cb.requests.Len() >= cb.config.SlidingWindowSize {
		if front := cb.requests.Front(); front != nil {
			cb.removeFrontRequest()
		}
	}
}

// enforceTimeBasedWindow removes entries older than SlidingWindowTime
func (cb *CircuitBreaker) enforceTimeBasedWindow() {
	expirationTime := time.Now().Add(-time.Duration(cb.config.SlidingWindowSize))

	for cb.requests.Len() > 0 {
		front := cb.requests.Front()
		if front == nil {
			break
		}

		entry := front.Value.(requestEntry)
		if !entry.executionTime.Before(expirationTime) {
			break // Stop removing when the first valid entry is found
		}

		cb.removeFrontRequest()
	}
}

// removeFrontRequest removes the front request and updates counters
func (cb *CircuitBreaker) removeFrontRequest() {
	front := cb.requests.Front()
	if front != nil {
		entry := front.Value.(requestEntry)

		// Decrement counters accordingly
		if entry.failed {
			cb.failureCount--
		} else if entry.slow {
			cb.slowCallCount--
		}

		cb.requests.Remove(front)
	}
}

// getFailureRate returns the failure percentage in O(1) time
func (cb *CircuitBreaker) getFailureRate() float64 {
	if cb.requests.Len() == 0 {
		return 0.0
	}
	return (float64(cb.failureCount) / float64(cb.requests.Len())) * 100
}

// getSlowCallRate returns the slow call percentage in O(1) time
func (cb *CircuitBreaker) getSlowCallRate() float64 {
	if cb.requests.Len() == 0 {
		return 0.0
	}
	return (float64(cb.slowCallCount) / float64(cb.requests.Len())) * 100
}
