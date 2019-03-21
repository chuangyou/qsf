package ratelimit

import (
	"strconv"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/chuangyou/qsf/grpc_error"
)

func UnaryServerInterceptor(rl *RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !rl.Limit() {
			return handler(ctx, req)
		} else {
			return nil, grpc_error.ResourceExhausted("服务并发限制", "当前服务最大并发数为"+strconv.FormatUint(rl.rate, 10)+"，请稍后重试。", 30)
		}
	}
}
func StreamServerInterceptor(rl *RateLimiter) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !rl.Limit() {
			return handler(srv, stream)
		} else {
			return grpc_error.ResourceExhausted("服务并发限制", "当前服务最大并发数为"+strconv.FormatUint(rl.rate, 10)+"，请稍后重试。", 30)
		}
	}
}
