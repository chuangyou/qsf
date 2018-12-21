package breaker

import (
	"github.com/chuangyou/qsf2/grpc_error"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func UnaryClientInterceptor(breaker *Breaker) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		err := breaker.Call(func() error {
			return invoker(ctx, method, req, reply, cc, opts...)
		}, 0)

		if err == ErrBreakerOpen {
			//service fallback
			return grpc_error.Unavailable()
		}
		return err
	}
}
func StreamServerInterceptor(breaker *Breaker) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := breaker.Call(func() error {
			return handler(srv, stream)
		}, 0)
		if err == ErrBreakerOpen {
			//service fallback
			return grpc_error.Unavailable()
		}
		return err
	}
}
