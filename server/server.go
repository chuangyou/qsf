package server

import (
	"time"

	"errors"
	"log"
	"net/http"

	"github.com/chuangyou/qsf2/grpc_error"
	etcd_registry "github.com/chuangyou/qsf2/plugin/loadbalance/registry/etcd"
	"github.com/chuangyou/qsf2/plugin/prometheus"
	"github.com/chuangyou/qsf2/plugin/ratelimit"
	"github.com/chuangyou/qsf2/plugin/tracing"
	etcd "github.com/coreos/etcd/clientv3"
	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	opentracing "github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	DEFAULT_ETCD_NODEID = "node1"
)

type Config struct {
	Version             string
	Name                string
	Path                string
	Addr                string
	Token               string
	EtcdConfig          etcd.Config
	Monitor             bool
	MonitorListenAddr   string
	RateLimit           int64
	ZipkinEnable        bool
	ZipkinCollectorAddr string
	ZipkinRecorderAddr  string
}
type Service struct {
	token             string
	monitor           bool
	Registry          *etcd_registry.EtcdReigistry
	ZipkinCollector   zipkin.Collector
	Server            *grpc.Server
	GrpcMetrics       *grpc_prometheus.ServerMetrics
	MonitorHttpServer *http.Server
}

func NewService(config *Config) (service *Service, err error) {

	var (
		unaryServerInterceptors  []grpc.UnaryServerInterceptor
		streamServerInterceptors []grpc.StreamServerInterceptor
		registry                 *etcd_registry.EtcdReigistry
		prometheusRegistry       *prometheus.Registry
		zipkinTrance             opentracing.Tracer
	)
	if config.Version == "" ||
		config.Name == "" ||
		config.Path == "" ||
		config.Addr == "" ||
		len(config.EtcdConfig.Endpoints) == 0 {
		err = errors.New("service config data error")
		return
	}
	service = new(Service)
	service.token = config.Token
	//register a service to etcd
	registry, err = etcd_registry.NewRegistry(
		etcd_registry.Option{
			EtcdConfig:  config.EtcdConfig,
			RegistryDir: config.Path,
			ServiceName: config.Name,
			NodeID:      DEFAULT_ETCD_NODEID,
			NData: etcd_registry.NodeData{
				Addr: config.Addr,
			},
			Ttl: 60 * time.Second,
		})
	if err != nil {
		return
	}
	//register a service to etcd

	if service.token != "" {
		unaryServerInterceptors = append(unaryServerInterceptors, grpc_auth.UnaryServerInterceptor(service.AuthFunc))
		streamServerInterceptors = append(streamServerInterceptors, grpc_auth.StreamServerInterceptor(service.AuthFunc))
	}
	//enable service monitor
	if config.Monitor && config.Addr != "" {
		service.monitor = true
		prometheusRegistry = prometheus.NewRegistry()
		service.GrpcMetrics = grpc_prometheus.NewServerMetrics()
		service.GrpcMetrics.EnableHandlingTimeHistogram()
		prometheusRegistry.MustRegister(service.GrpcMetrics)
		service.MonitorHttpServer = &http.Server{Handler: promhttp.HandlerFor(prometheusRegistry, promhttp.HandlerOpts{}), Addr: config.MonitorListenAddr}
		unaryServerInterceptors = append(unaryServerInterceptors, grpc.UnaryServerInterceptor(service.GrpcMetrics.UnaryServerInterceptor()))
		streamServerInterceptors = append(streamServerInterceptors, grpc.StreamServerInterceptor(service.GrpcMetrics.StreamServerInterceptor()))
	}
	//enable service monitor

	//enable service ratelimit
	if config.RateLimit > 0 {
		rateLimter := ratelimit.NewBucketWithRate(float64(config.RateLimit), config.RateLimit)
		unaryServerInterceptors = append(unaryServerInterceptors, ratelimit.UnaryServerInterceptor(rateLimter))
		streamServerInterceptors = append(streamServerInterceptors, ratelimit.StreamServerInterceptor(rateLimter))
	}
	//enable service monitor

	//enable the zipkin
	if config.ZipkinEnable && config.ZipkinCollectorAddr != "" && config.ZipkinRecorderAddr != "" {
		service.ZipkinCollector, err = zipkin.NewHTTPCollector(config.ZipkinCollectorAddr)
		if err != nil {
			return
		}
		zipkinTrance, err = zipkin.NewTracer(
			zipkin.NewRecorder(service.ZipkinCollector, false, config.ZipkinRecorderAddr, config.Name+"."+config.Version+".Server"),
			zipkin.ClientServerSameSpan(true),
			zipkin.TraceID128Bit(true),
		)
		if err != nil {
			return
		}
		opentracing.InitGlobalTracer(zipkinTrance)
		unaryServerInterceptors = append(unaryServerInterceptors, grpc.UnaryServerInterceptor(otgrpc.OpenTracingServerInterceptor(zipkinTrance, otgrpc.LogPayloads())))
	}
	//enable the zipkin

	if len(unaryServerInterceptors) > 0 && len(streamServerInterceptors) > 0 {
		service.Server = grpc.NewServer(
			grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryServerInterceptors...)),
			grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamServerInterceptors...)),
		)
	} else if len(unaryServerInterceptors) > 0 {
		service.Server = grpc.NewServer(
			grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(unaryServerInterceptors...)),
		)
	} else if len(streamServerInterceptors) > 0 {
		service.Server = grpc.NewServer(
			grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(streamServerInterceptors...)),
		)
	} else {
		service.Server = grpc.NewServer()
	}

	go func() {
		if err := registry.Register(); err != nil {
			log.Fatalf("service registe failed: %v", err)
		} else {
			log.Println("service registe")
		}
	}()

	return

}
func (s *Service) AuthFunc(ctx context.Context) (context.Context, error) {
	token, err := grpc_auth.AuthFromMD(ctx, "Basic")
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, grpc_error.Unauthenticated()
	}
	if token != s.token {
		return nil, grpc_error.PermissionDenied("服务授权token错误！")
	}
	return ctx, nil
}
