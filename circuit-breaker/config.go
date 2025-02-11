package circuit_breaker

import (
	"time"
)

type SlidingWindowType string

const (
	COUNT_BASED SlidingWindowType = "COUNT_BASED"
	TIME_BASED  SlidingWindowType = "TIME_BASED"
)

type Config struct {
	Name                                  string
	FailureRateThreshold                  float64
	MinimumNumberOfCalls                  int
	PermittedNumberOfCallsInHalfOpenState int32
	WaitDurationInOpenState               time.Duration
	SlidingWindowSize                     int           // int for COUNT_BASED,
	SlidingWindowTime                     time.Duration //time window for TIME_BASED
	SlidingWindowType                     SlidingWindowType
	SlowCallDurationThreshold             time.Duration
	SlowCallRateThreshold                 float64
}
