package server

import (
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/chuangyou/qsf/constant"
	"github.com/chuangyou/qsf/grpc_error"
	etcd_registry "github.com/chuangyou/qsf/plugin/loadbalance/registry/etcd"
	"github.com/chuangyou/qsf/plugin/prometheus"
	"github.com/chuangyou/qsf/plugin/ratelimit"
	"github.com/chuangyou/qsf/plugin/tracing"
	etcd "github.com/coreos/etcd/clientv3"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type Config struct {
	Name              string                 //服务名称
	Addr              string                 //服务地址
	NodeId            string                 //服务节点
	RegistryAddrs     []string               //服务注册地址
	AccessToken       string                 //服务密钥
	RateLimter        *ratelimit.RateLimiter //服务限流器
	MonitorListenAddr string                 //服务监控地址
	Tracer            opentracing.Tracer     //服务tracer
}
type Service struct {
	Addr              string       //服务地址
	AccessToken       string       //服务密钥
	GrpcServer        *grpc.Server //grpc实例
	monitorHttpServer *http.Server
	grpcMetrics       *grpc_prometheus.ServerMetrics
}

func NewSevice(config *Config) (service *Service, err error) {
	var (
		unaryServerInterceptors  []grpc.UnaryServerInterceptor
		streamServerInterceptors []grpc.StreamServerInterceptor
	)
	if config.Name == "" || config.Addr == "" || config.NodeId == "" || len(config.RegistryAddrs) == 0 {
		err = errors.New("service config data error")
		return
	}
	service = new(Service)
	service.Addr = config.Addr
	//register a service to etcd
	err = service.registry(config.RegistryAddrs, config.Name, config.Addr, config.NodeId)
	if err != nil {
		return
	}
	//service accessToken
	service.AccessToken = config.AccessToken
	if service.AccessToken != "" {
		unaryServerInterceptors = append(unaryServerInterceptors, grpc_auth.UnaryServerInterceptor(service.AuthFunc))
		streamServerInterceptors = append(streamServerInterceptors, grpc_auth.StreamServerInterceptor(service.AuthFunc))
	}
	//enable service ratelimit
	if config.RateLimter != nil {
		unaryServerInterceptors = append(unaryServerInterceptors, ratelimit.UnaryServerInterceptor(config.RateLimter))
		streamServerInterceptors = append(streamServerInterceptors, ratelimit.StreamServerInterceptor(config.RateLimter))
	}
	//enable service monitor
	if config.MonitorListenAddr != "" {
		prometheusRegistry := prometheus.NewRegistry()
		service.grpcMetrics = grpc_prometheus.NewServerMetrics()
		service.grpcMetrics.EnableHandlingTimeHistogram()
		prometheusRegistry.MustRegister(service.grpcMetrics)
		service.monitorHttpServer = &http.Server{Handler: promhttp.HandlerFor(prometheusRegistry, promhttp.HandlerOpts{}), Addr: config.MonitorListenAddr}
		unaryServerInterceptors = append(unaryServerInterceptors, grpc.UnaryServerInterceptor(service.grpcMetrics.UnaryServerInterceptor()))
		streamServerInterceptors = append(streamServerInterceptors, grpc.StreamServerInterceptor(service.grpcMetrics.StreamServerInterceptor()))
	}
	//enable the tracer
	if config.Tracer != nil {
		unaryServerInterceptors = append(unaryServerInterceptors, grpc.UnaryServerInterceptor(otgrpc.OpenTracingServerInterceptor(config.Tracer, otgrpc.LogPayloads())))
		streamServerInterceptors = append(streamServerInterceptors, grpc.StreamServerInterceptor(otgrpc.OpenTracingStreamServerInterceptor(config.Tracer, otgrpc.LogPayloads())))
	}
	if len(unaryServerInterceptors) > 0 && len(streamServerInterceptors) > 0 {
		service.GrpcServer = grpc.NewServer(
			grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryServerInterceptors...)),
			grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamServerInterceptors...)),
		)
	} else if len(unaryServerInterceptors) > 0 {
		service.GrpcServer = grpc.NewServer(
			grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryServerInterceptors...)),
		)
	} else if len(streamServerInterceptors) > 0 {
		service.GrpcServer = grpc.NewServer(
			grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamServerInterceptors...)),
		)
	} else {
		service.GrpcServer = grpc.NewServer()
	}

	return

}
func (s *Service) registry(registryAddrs []string, serviceName, serviceAddr, serviceNodeId string) (err error) {
	var (
		registry *etcd_registry.EtcdReigistry
	)
	//register a service to etcd
	registry, err = etcd_registry.NewRegistry(
		etcd_registry.Option{
			EtcdConfig: etcd.Config{
				Endpoints: registryAddrs,
			},
			RegistryDir: constant.DEFAULT_ETCD_PATH,
			ServiceName: serviceName,
			NodeID:      serviceNodeId,
			NData: etcd_registry.NodeData{
				Addr: serviceAddr,
				//Metadata: map[string]string{"service_version": serviceVersion},
			},
			Ttl: 10 * time.Second,
		})
	if err == nil {
		go func() {
			registry.Register()
		}()
	}
	return

}

func (s *Service) Run() {
	///enable service monitor server
	if s.monitorHttpServer != nil && s.grpcMetrics != nil {
		s.grpcMetrics.InitializeMetrics(s.GrpcServer)
		go func() {
			if err := s.monitorHttpServer.ListenAndServe(); err != nil {
				log.Fatalf("Unable to start a monitor http server.")
			}
		}()
	}
	go func() {
		lis, err := net.Listen("tcp", s.Addr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		if err := s.GrpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()
	s.handleSignal()

}
func (s *Service) handleSignal() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		systemSignal := <-c
		log.Println("server get a signal ", systemSignal.String())
		switch systemSignal {
		case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			signal.Stop(c)
			s.GrpcServer.GracefulStop()
			return
		case syscall.SIGHUP:
			signal.Stop(c)
			s.GrpcServer.GracefulStop()
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
			s.GrpcServer.GracefulStop()
			return
		}
	}
}
func (s *Service) AuthFunc(ctx context.Context) (context.Context, error) {
	accessToken, err := grpc_auth.AuthFromMD(ctx, "Basic")
	if err != nil {
		return nil, err
	}
	if accessToken == "" {
		return nil, grpc_error.Internal()
	}
	if accessToken != s.AccessToken {
		return nil, grpc_error.Internal()
	}
	return ctx, nil
}
