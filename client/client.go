package client

import (
	"errors"

	"github.com/chuangyou/qsf2/plugin/breaker"
	grpclb "github.com/chuangyou/qsf2/plugin/loadbalance"
	registry "github.com/chuangyou/qsf2/plugin/loadbalance/registry/etcd"
	"github.com/chuangyou/qsf2/plugin/tracing"
	etcd "github.com/coreos/etcd/clientv3"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/naming"
)

const (
	Version               = "V1"
	InitialWindowSize     = 1 << 30
	InitialConnWindowSize = 1 << 30
	MaxSendMsgSize        = 1<<31 - 1
	MaxCallMsgSize        = 1<<31 - 1
	BreakerRate           = float64(0.95)
	BreakMinSamples       = int64(100)
)

type ServiceCredential interface {
	SetServiceToken(string)
	GetRequestMetadata(context.Context, ...string) (map[string]string, error)
	RequireTransportSecurity() bool
}
type Config struct {
	Name          string
	Path          string
	Token         string
	TokenFunc     ServiceCredential
	EtcdConfig    etcd.Config
	LoadBalancer  string
	ZipkinEnable  bool
	ZipkinTrance  opentracing.Tracer
	BreakerEnable bool
}
type Client struct {
	GrpcOpts []grpc.DialOption
}

func NewClient(config *Config) (client *Client, err error) {
	var (
		r       naming.Resolver
		b       grpc.Balancer
		grpcOpt grpc.DialOption
	)
	if config.Name == "" ||
		config.Path == "" ||
		len(config.EtcdConfig.Endpoints) == 0 {
		err = errors.New("service config data error")
		return
	}
	client = new(Client)
	client.GrpcOpts = append(client.GrpcOpts, grpc.WithInsecure())
	client.GrpcOpts = append(client.GrpcOpts, grpc.WithInitialWindowSize(InitialWindowSize))
	client.GrpcOpts = append(client.GrpcOpts, grpc.WithInitialConnWindowSize(InitialConnWindowSize))
	client.GrpcOpts = append(client.GrpcOpts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(MaxCallMsgSize),
		grpc.MaxCallSendMsgSize(MaxSendMsgSize),
	))
	if config.Token != "" {
		if config.TokenFunc == nil {
			err = errors.New("service token config error")
			return
		}
		//setToken
		config.TokenFunc.SetServiceToken(config.Token)
		client.GrpcOpts = append(client.GrpcOpts, grpc.WithPerRPCCredentials(config.TokenFunc))
	}
	//service discovery
	r = registry.NewResolver(
		config.Path,
		config.Name,
		config.EtcdConfig,
	)
	//loadbalance
	if config.LoadBalancer == "roundrobin" {
		b = grpclb.NewBalancer(r, grpclb.NewRoundRobinSelector())
	} else if config.LoadBalancer == "random" {
		b = grpclb.NewBalancer(r, grpclb.NewRandomSelector())
	} else {
		b = grpclb.NewBalancer(r, nil)
	}
	client.GrpcOpts = append(client.GrpcOpts, grpc.WithBalancer(b))
	//breaker
	if config.BreakerEnable {
		grpcOpt = grpc.WithUnaryInterceptor(
			grpc_middleware.ChainUnaryClient(
				breaker.UnaryClientInterceptor(breaker.NewRateBreaker(BreakerRate, BreakMinSamples)),
			),
		)
		client.GrpcOpts = append(client.GrpcOpts, grpcOpt)
	}
	//zipkin
	if config.ZipkinEnable {
		if config.ZipkinTrance != nil {
			grpcOpt = grpc.WithUnaryInterceptor(
				grpc_middleware.ChainUnaryClient(
					otgrpc.OpenTracingClientInterceptor(config.ZipkinTrance),
				),
			)
			client.GrpcOpts = append(client.GrpcOpts, grpcOpt)
		}
	}
	return
}
