package util

import (
	"context"
	"errors"
	"time"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewCircuitBreaker creates a circuit breaker for a given client name with the shared configuration.
func NewCircuitBreaker(name string, maxRequests uint32) *gobreaker.CircuitBreaker {
	return gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        name,
		MaxRequests: maxRequests,      // Half-Open state allowed requests
		Interval:    5 * time.Second,  // Cyclic period of the Closed state
		Timeout:     10 * time.Second, // Timeout before transitioning from Open to Half-Open
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip if consecutive failures count > 5
			return counts.ConsecutiveFailures > 5
		},
		IsSuccessful: func(err error) bool {
			if err == nil {
				return true
			}
			st, ok := status.FromError(err)
			if !ok {
				// Non-gRPC errors or unknown errors count as failures
				return false
			}
			switch st.Code() {
			case codes.Internal, codes.Unavailable, codes.DeadlineExceeded:
				// Only system/transient errors count as failures that trip the breaker
				return false
			}
			// Client errors (InvalidArgument, NotFound, etc.) do NOT count as failures
			return true
		},
	})
}

// CircuitBreakerInterceptor returns a unary client interceptor with the given circuit breaker.
func CircuitBreakerInterceptor(cb *gobreaker.CircuitBreaker) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if cb == nil {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
		_, err := cb.Execute(func() (any, error) {
			err := invoker(ctx, method, req, reply, cc, opts...)
			return nil, err
		})
		if err != nil {
			if errors.Is(err, gobreaker.ErrOpenState) {
				return status.Errorf(codes.Unavailable, "circuit breaker %q is open", cb.Name())
			}
			return err
		}
		return nil
	}
}
