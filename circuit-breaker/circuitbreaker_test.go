package circuit_breaker

import (
	"errors"
	"testing"
	"time"
)

func TestHalfOpenToClosedTransition(t *testing.T) {
	config := Config{
		Name:                                  "test",
		SlidingWindowType:                     COUNT_BASED,
		FailureRateThreshold:                  50,
		MinimumNumberOfCalls:                  6,
		WaitDurationInOpenState:               2 * time.Second,
		PermittedNumberOfCallsInHalfOpenState: 3,
		SlidingWindowSize:                     10,
		SlowCallDurationThreshold:             200 * time.Millisecond,
		SlowCallRateThreshold:                 50.0,
	}
	breaker := NewCircuitBreaker(config)

	// Step 1: Fail enough calls to trigger Open state
	for i := 1; i <= 6; i++ {
		_ = breaker.Execute(func() error { return errors.New("failure") })
	}

	if breaker.state.state != Open {
		t.Errorf("Expected circuit breaker to be OPEN, got %v", breaker.state.state)
	}

	// Step 2: Wait for Open → Half-Open transition
	time.Sleep(3 * time.Second)

	//make success call to return to half open
	breaker.Execute(func() error { return nil })

	if breaker.state.state != HalfOpen {
		t.Errorf("Expected circuit breaker to be HALF-OPEN, got %v", breaker.state.state)
	}

	// Step 3: Make permitted successful calls
	breaker.Execute(func() error { return nil })
	breaker.Execute(func() error { return nil }) // Last successful call in Half-Open

	if breaker.state.state != Closed {
		t.Errorf("Expected circuit breaker to be CLOSED after successful Half-Open calls, got %v", breaker.state.state)
	}

	// Verify that metrics are reset
	if breaker.halfOpenCalls != 0 {
		t.Errorf("Expected reset metrics, got halfOpenCalls=%d", breaker.halfOpenCalls)
	}
}
