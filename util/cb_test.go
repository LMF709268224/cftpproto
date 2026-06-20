package util

import (
	"context"
	"testing"

	"github.com/sony/gobreaker"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCircuitBreakerInterceptor(t *testing.T) {
	t.Run("by-pass when cb is nil", func(t *testing.T) {
		interceptor := CircuitBreakerInterceptor(nil)
		called := false
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			called = true
			return status.Error(codes.InvalidArgument, "business error")
		}

		err := interceptor(context.Background(), "/test/Method", nil, nil, nil, invoker)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !called {
			t.Fatal("expected invoker to be called")
		}
		if status.Code(err) != codes.InvalidArgument {
			t.Fatalf("expected InvalidArgument, got: %v", err)
		}
	})

	t.Run("business errors do not trip the breaker", func(t *testing.T) {
		cb := NewCircuitBreaker("test-cb-business", 100)
		interceptor := CircuitBreakerInterceptor(cb)

		// Call 10 times with NotFound (which should NOT count as failure)
		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return status.Error(codes.NotFound, "not found")
		}

		for i := 0; i < 10; i++ {
			err := interceptor(context.Background(), "/test/Method", nil, nil, nil, invoker)
			if status.Code(err) != codes.NotFound {
				t.Fatalf("expected NotFound, got: %v", err)
			}
		}

		// Breaker should still be Closed
		if cb.State() != gobreaker.StateClosed {
			t.Fatalf("expected state Closed, got: %v", cb.State())
		}
	})

	t.Run("transient errors trip the breaker after 5 consecutive failures", func(t *testing.T) {
		cb := NewCircuitBreaker("test-cb-transient", 100)
		interceptor := CircuitBreakerInterceptor(cb)

		invoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return status.Error(codes.Unavailable, "service unavailable")
		}

		// First 5 requests fail but don't trip yet (consecutive <= 5)
		for i := 0; i < 5; i++ {
			err := interceptor(context.Background(), "/test/Method", nil, nil, nil, invoker)
			if status.Code(err) != codes.Unavailable {
				t.Fatalf("expected Unavailable, got: %v", err)
			}
			if cb.State() != gobreaker.StateClosed {
				t.Fatalf("expected state Closed on iteration %d, got: %v", i, cb.State())
			}
		}

		// 6th failure should trip the breaker (ConsecutiveFailures > 5)
		err := interceptor(context.Background(), "/test/Method", nil, nil, nil, invoker)
		if status.Code(err) != codes.Unavailable {
			t.Fatalf("expected Unavailable on 6th call, got: %v", err)
		}

		if cb.State() != gobreaker.StateOpen {
			t.Fatalf("expected state Open, got: %v", cb.State())
		}

		// 7th call should fail immediately with breaker open error (without calling invoker)
		invokerCalled := false
		dummyInvoker := func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			invokerCalled = true
			return nil
		}

		err = interceptor(context.Background(), "/test/Method", nil, nil, nil, dummyInvoker)
		if invokerCalled {
			t.Fatal("expected invoker NOT to be called when breaker is open")
		}
		if status.Code(err) != codes.Unavailable {
			t.Fatalf("expected Unavailable due to open circuit breaker, got: %v", err)
		}
	})
}
