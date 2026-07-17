package recovery

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

func (m *mockServerStream) SendMsg(msg interface{}) error {
	panic("panic in send")
}

func (m *mockServerStream) RecvMsg(msg interface{}) error {
	panic("panic in recv")
}

func TestStreamServerInterceptor_PanicInHandler(t *testing.T) {
	var called bool
	handler := func(ctx context.Context, p interface{}) error {
		called = true
		return status.Errorf(codes.Internal, "custom panic: %v", p)
	}

	interceptor := StreamServerInterceptor(handler)
	stream := &mockServerStream{ctx: context.Background()}

	err := interceptor(nil, stream, nil, func(srv interface{}, ss grpc.ServerStream) error {
		panic("panic in handler")
	})

	if !called {
		t.Error("expected recovery handler to be called")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal || st.Message() != "custom panic: panic in handler" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestStreamServerInterceptor_PanicInSend(t *testing.T) {
	var called bool
	handler := func(ctx context.Context, p interface{}) error {
		called = true
		return status.Errorf(codes.Internal, "custom panic: %v", p)
	}

	interceptor := StreamServerInterceptor(handler)
	stream := &mockServerStream{ctx: context.Background()}

	err := interceptor(nil, stream, nil, func(srv interface{}, ss grpc.ServerStream) error {
		return ss.SendMsg("test")
	})

	if !called {
		t.Error("expected recovery handler to be called")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal || st.Message() != "custom panic: panic in send" {
		t.Errorf("unexpected error: %v", err)
	}
}
