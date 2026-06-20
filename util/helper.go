package util

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var (
	// ULID regex: 26 chars, Base32 charset, first char in 0-7
	ulidRegex = regexp.MustCompile(`^[0-7][0-9A-HJKMNP-TV-Z]{25}$`)
	// UUID regex: standard 8-4-4-4-12 hex format
	uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	// GPath regex: starts with / and only contains lowercase letters, numbers, hyphens, underscores, and slashes
	gpathRegex = regexp.MustCompile(`^/[a-z0-9_-]+(/[a-z0-9_-]+)*$`)
)

func IsRunningInK8s() bool {
	_, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount")
	return err == nil
}

func GetNamespace() (string, error) {
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func GetEndpointAddress(envName, svcName, port string) string {
	endpoint := os.Getenv(envName)
	if endpoint == "" {
		namespace, err := GetNamespace()
		if err != nil {
			namespace = "default"
		}

		endpoint = fmt.Sprintf("%s.%s.svc.cluster.local:%s", svcName, namespace, port)
	}

	return endpoint
}

func GetGrpcClientTransportCreds() credentials.TransportCredentials {
	tlsDir := strings.TrimSpace(os.Getenv("TLS_DIR"))
	if tlsDir == "" {
		return insecure.NewCredentials()
	}

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	caFile := filepath.Join(tlsDir, "ca.crt")
	if caPEM, err := os.ReadFile(caFile); err == nil {
		pool := x509.NewCertPool()
		if ok := pool.AppendCertsFromPEM(caPEM); !ok {
			slog.Warn("gRPC: failed to append CA cert", "ca_file", caFile)
			return insecure.NewCredentials()
		}

		tlsConfig.RootCAs = pool
	} else {
		slog.Warn("gRPC: load ca faild", "ca_file", caFile, "error", err)
		return insecure.NewCredentials()
	}

	return credentials.NewTLS(tlsConfig)
}

func GetGrpcServerTransportCreds() credentials.TransportCredentials {
	tlsDir := os.Getenv("TLS_DIR")
	if tlsDir != "" {
		certFile := filepath.Join(tlsDir, "tls.crt")
		keyFile := filepath.Join(tlsDir, "tls.key")
		tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			slog.Error("failed to load TLS cert/key", "tls_dir", tlsDir, "cert_file", certFile, "key_file", keyFile, "error", err)
			return nil
		}

		tlsConfig := &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{tlsCert},
		}
		return credentials.NewTLS(tlsConfig)
	}

	return nil
}

// NewULID generates a ULID with the current UTC timestamp.
func NewULID() string {
	ms := ulid.Timestamp(time.Now().UTC())
	id, err := ulid.New(ms, rand.Reader)
	if err != nil {
		panic("NewULID: " + err.Error())
	}
	return id.String()
}

func IsValidULID(s string) bool {
	return ulidRegex.MatchString(s)
}

func IsValidUUID(s string) bool {
	return uuidRegex.MatchString(s)
}

func IsValidGPath(s string) bool {
	return gpathRegex.MatchString(s)
}

func WrapGrpcError(serviceName string, err error) error {
	if err == nil {
		return nil
	}
	if st, ok := status.FromError(err); ok {
		return status.Errorf(st.Code(), "%s: %s", serviceName, st.Message())
	}
	return status.Errorf(codes.Internal, "%s: %v", serviceName, err)
}

func LoggingRecoveryUnaryServerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	startedAt := time.Now().UTC()
	method := ""
	if info != nil {
		method = info.FullMethod
	}

	slog.InfoContext(ctx, "gRPC request started", "method", method)
	defer func() {
		durationMs := time.Since(startedAt).Milliseconds()
		if r := recover(); r != nil {
			slog.ErrorContext(ctx, "gRPC server panic recovered",
				"method", method,
				"code", codes.Internal.String(),
				"duration_ms", durationMs,
				"error", r,
				"stack", string(debug.Stack()),
			)
			err = status.Errorf(codes.Internal, "panic recovered: %v", r)
			return
		}

		code := status.Code(err)
		attrs := []any{
			"method", method,
			"code", code.String(),
			"duration_ms", durationMs,
		}
		if err != nil {
			attrs = append(attrs, "error", err)
			if isClientOrExpectedGrpcCode(code) {
				slog.WarnContext(ctx, "gRPC request failed", attrs...)
			} else {
				slog.ErrorContext(ctx, "gRPC request failed", attrs...)
			}
			return
		}

		slog.InfoContext(ctx, "gRPC request completed", attrs...)
	}()

	return handler(ctx, req)
}

func LoggingUnaryClientInterceptor(ctx context.Context, method string, req any, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
	startedAt := time.Now().UTC()
	target := ""
	if cc != nil {
		target = cc.Target()
	}

	err := invoker(ctx, method, req, reply, cc, opts...)
	durationMs := time.Since(startedAt).Milliseconds()
	code := status.Code(err)
	attrs := []any{
		"target", target,
		"method", method,
		"code", code.String(),
		"duration_ms", durationMs,
	}
	if err != nil {
		attrs = append(attrs, "error", err)
		if isClientOrExpectedGrpcCode(code) {
			slog.WarnContext(ctx, "gRPC call-out failed", attrs...)
		} else {
			slog.ErrorContext(ctx, "gRPC call-out failed", attrs...)
		}
		return err
	}

	slog.InfoContext(ctx, "gRPC call-out completed", attrs...)
	return nil
}

func isClientOrExpectedGrpcCode(code codes.Code) bool {
	switch code {
	case codes.Canceled,
		codes.InvalidArgument,
		codes.NotFound,
		codes.AlreadyExists,
		codes.PermissionDenied,
		codes.Unauthenticated,
		codes.FailedPrecondition,
		codes.OutOfRange,
		codes.DeadlineExceeded,
		codes.Aborted:
		return true
	default:
		return false
	}
}
