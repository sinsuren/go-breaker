# Circuit Breaker for Go

## Overview
This repository provides a simple implementation of the Circuit Breaker pattern in Go. The Circuit Breaker helps improve the resilience of applications by preventing repeated execution of failing operations, allowing the system to recover and maintain stability.

## Features
- **Three States:** Closed, Open, and Half-Open
- **Sliding Window Support:** Count-based and Time-based
- **Failure and Slow Call Monitoring**
- **Configurable Thresholds**
- **Automatic State Transitions**

## Installation
```sh
go get github.com/sinsuren/go_breaker
```

## Usage

### Initializing the Circuit Breaker
```go
config := Config{
    Name: "ExampleService",
    FailureRateThreshold: 50.0,
    SlowCallRateThreshold: 50.0,
    SlowCallDurationThreshold: 2 * time.Second,
    MinimumNumberOfCalls: 5,
    SlidingWindowType: COUNT_BASED,
    SlidingWindowSize: 10,
    PermittedNumberOfCallsInHalfOpenState: 3,
    WaitDurationInOpenState: 5 * time.Second,
}

cb := NewCircuitBreaker(config)
```

### Executing Actions with Circuit Breaker
```go
err := cb.Execute(func() error {
    // Simulate a request or operation
    return errors.New("service unavailable")
})

if err != nil {
    fmt.Println("Request failed:", err)
}
```

## Circuit Breaker States
1. **Closed**: The system is functioning normally.
2. **Open**: The system has encountered excessive failures and stops sending requests.
3. **Half-Open**: The system allows a limited number of test requests to check if recovery is possible.

## Configuration Parameters
| Parameter | Description |
|-----------|-------------|
| `FailureRateThreshold` | Percentage of failures to trigger Open state |
| `SlowCallRateThreshold` | Percentage of slow calls to trigger Open state |
| `SlowCallDurationThreshold` | Time threshold to consider a call slow |
| `MinimumNumberOfCalls` | Minimum calls required before evaluation |
| `SlidingWindowType` | Count-based or Time-based window |
| `SlidingWindowSize` | Size of the sliding window |
| `PermittedNumberOfCallsInHalfOpenState` | Allowed calls in Half-Open state |
| `WaitDurationInOpenState` | Time to wait before transitioning to Half-Open |

## Contributions
Contributions are welcome! Please submit a pull request or open an issue for suggestions or improvements.

