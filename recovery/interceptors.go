package recovery

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryHandlerFunc is a function that recovers from a panic.
type RecoveryHandlerFunc func(ctx context.Context, p interface{}) (err error)

type wrappedStream struct {
	grpc.ServerStream
	recoveryHandler RecoveryHandlerFunc
}

func (w *wrappedStream) RecvMsg(m interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = recoverFrom(w.Context(), r, w.recoveryHandler)
		}
	}()
	return w.ServerStream.RecvMsg(m)
}

func (w *wrappedStream) SendMsg(m interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = recoverFrom(w.Context(), r, w.recoveryHandler)
		}
	}()
	return w.ServerStream.SendMsg(m)
}

// StreamServerInterceptor returns a new streaming server interceptor that recovers from panics.
func StreamServerInterceptor(handler RecoveryHandlerFunc) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handlerFunc grpc.StreamHandler) (err error) {
		ws := &wrappedStream{
			ServerStream:    ss,
			recoveryHandler: handler,
		}
		defer func() {
			if r := recover(); r != nil {
				err = recoverFrom(ss.Context(), r, handler)
			}
		}()
		return handlerFunc(srv, ws)
	}
}

// UnaryServerInterceptor returns a new unary server interceptor that recovers from panics.
func UnaryServerInterceptor(handler RecoveryHandlerFunc) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handlerFunc grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = recoverFrom(ctx, r, handler)
			}
		}()
		return handlerFunc(ctx, req)
	}
}

func recoverFrom(ctx context.Context, r interface{}, handler RecoveryHandlerFunc) error {
	if handler != nil {
		return handler(ctx, r)
		// Note: if the handler returns nil, we still want to return a status error to prevent hanging
	}
	return status.Errorf(codes.Internal, "%v", r)
}
