package ratelimit

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"strconv"

	"github.com/chuangyou/qsf/grpc_error"
)

func UnaryServerInterceptor(bucket *Bucket) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if bucket.TakeAvailable(1) > 0 {
			return handler(ctx, req)
		} else {
			return nil, grpc_error.ResourceExhausted("服务并发限制", "当前服务最大并发数为"+strconv.FormatInt(bucket.Capacity(), 10)+"，请稍后重试。")
		}
	}
}
func StreamServerInterceptor(bucket *Bucket) grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if bucket.TakeAvailable(1) > 0 {
			return handler(srv, stream)
		} else {
			return grpc_error.ResourceExhausted("服务并发限制", "当前服务最大并发数为"+strconv.FormatInt(bucket.Capacity(), 10)+"，请稍后重试。")
		}
	}
}
