package client

import (
	"errors"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/chuangyou/qsf/constant"
	"github.com/chuangyou/qsf/plugin/breaker"
	registry "github.com/chuangyou/qsf/plugin/loadbalance/registry/etcd"
	"github.com/chuangyou/qsf/plugin/prometheus"
	"github.com/chuangyou/qsf/plugin/tracing"
	etcd "github.com/coreos/etcd/clientv3"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/naming"
)

type Config struct {
	Name            string              //服务名
	AccessToken     string              //服务密钥
	AccessTokenFunc ServiceCredentialer //授权方法
	RegistryAddrs   []string            //服务注册地址
	Breaker         *breaker.Breaker    //熔断器
	Tracer          opentracing.Tracer  //服务tracer
	GrpcMetrics     *grpc_prometheus.ClientMetrics
}
type Client struct {
	GrpcConn *grpc.ClientConn
	GrpcOpts []grpc.DialOption
}

func NewClient(config *Config, isGateway bool) (client *Client, err error) {
	var (
		r                        naming.Resolver
		b                        grpc.Balancer
		grpcOpts                 []grpc.DialOption
		unaryClientInterceptors  []grpc.UnaryClientInterceptor
		streamClientInterceptors []grpc.StreamClientInterceptor
	)
	if config.Name == "" || len(config.RegistryAddrs) == 0 {
		err = errors.New("service config data error")
		return
	}
	client = new(Client)
	grpcOpts = append(grpcOpts, grpc.WithInsecure())
	grpcOpts = append(grpcOpts, grpc.WithInitialWindowSize(constant.InitialWindowSize))
	grpcOpts = append(grpcOpts, grpc.WithInitialConnWindowSize(constant.InitialConnWindowSize))
	grpcOpts = append(grpcOpts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(constant.MaxCallMsgSize),
		grpc.MaxCallSendMsgSize(constant.MaxSendMsgSize),
	))
	if config.AccessToken != "" {
		if config.AccessTokenFunc == nil {
			err = errors.New("service token config error")
			return
		}
		//setToken
		config.AccessTokenFunc.SetServiceToken(config.AccessToken)
		grpcOpts = append(grpcOpts, grpc.WithPerRPCCredentials(config.AccessTokenFunc))
	}

	//service discovery
	r = registry.NewResolver(
		constant.DEFAULT_ETCD_PATH,
		config.Name,
		etcd.Config{
			Endpoints: config.RegistryAddrs,
		},
	)
	//loadbalance
	b = grpc.RoundRobin(r)
	grpcOpts = append(grpcOpts, grpc.WithBalancer(b))

	if config.Breaker != nil {
		unaryClientInterceptors = append(unaryClientInterceptors, breaker.UnaryClientInterceptor(config.Breaker))

	}
	if config.Tracer != nil {
		unaryClientInterceptors = append(unaryClientInterceptors, otgrpc.OpenTracingClientInterceptor(config.Tracer))
		streamClientInterceptors = append(streamClientInterceptors, otgrpc.OpenTracingStreamClientInterceptor(config.Tracer))
	}
	if config.GrpcMetrics != nil {
		unaryClientInterceptors = append(unaryClientInterceptors, config.GrpcMetrics.UnaryClientInterceptor())
		streamClientInterceptors = append(streamClientInterceptors, config.GrpcMetrics.StreamClientInterceptor())
	}
	if len(unaryClientInterceptors) > 0 && len(streamClientInterceptors) > 0 {
		grpcOpts = append(grpcOpts, grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(unaryClientInterceptors...)))
		grpcOpts = append(grpcOpts, grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(streamClientInterceptors...)))
	} else if len(unaryClientInterceptors) > 0 {
		grpcOpts = append(grpcOpts, grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(unaryClientInterceptors...)))
	} else if len(streamClientInterceptors) > 0 {
		grpcOpts = append(grpcOpts, grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(streamClientInterceptors...)))
	}
	if isGateway {
		client.GrpcOpts = grpcOpts
	} else {
		client.GrpcConn, err = grpc.Dial("", grpcOpts...)
	}

	return
}

func HandleSignal(httpServer http.Server) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		systemSignal := <-c
		log.Println("server get a signal ", systemSignal.String())
		switch systemSignal {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			signal.Stop(c)
			httpServer.Shutdown(nil)
			return
		case syscall.SIGHUP:
			signal.Stop(c)
			httpServer.Shutdown(nil)
			cmd := exec.Command(os.Args[0], os.Args[1:]...)
			err := cmd.Start()
			if err != nil {
				log.Println("cmd.Start fail: ", err)
				return
			}
			log.Println("forked new pid : ", cmd.Process.Pid)
			return
		default:
			signal.Stop(c)
			httpServer.Shutdown(nil)
			return
		}
	}
}

type ServiceCredentialer interface {
	SetServiceToken(string)
	GetRequestMetadata(context.Context, ...string) (map[string]string, error)
	RequireTransportSecurity() bool
}
type ServiceCredential struct {
	serviceToken string
}

func (c *ServiceCredential) SetServiceToken(token string) {
	c.serviceToken = token
}
func (c *ServiceCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "basic " + c.serviceToken,
	}, nil
}
func (c *ServiceCredential) RequireTransportSecurity() bool {
	return false
}
