package main

import (
	"flag"
	"log"
	"strconv"

	"github.com/chuangyou/rsa"

	spb "github.com/chuangyou/qsf/examples/pb"
	"github.com/chuangyou/qsf/grpc_error"
	"github.com/chuangyou/qsf/plugin/ratelimit"
	"github.com/chuangyou/qsf/server"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"github.com/zheng-ji/goSnowFlake"
	"golang.org/x/net/context"
)

const (
	ZIPKIN_HTTP_ENDPOINT      = "http://127.0.0.1:9411/api/v1/spans"
	ZIPKIN_RECORDER_HOST_PORT = "127.0.0.1:0"
)

var nodeID = flag.String("node", "node1", "node ID")
var addr = flag.String("addr", "0.0.0.0:28544", "listening addr")
var monitorListenAddr = flag.String("monitoraddr", "0.0.0.0:9094", "monitor listen addr")

func main() {
	flag.Parse()
	config := new(server.Config)
	config.Name = "example" //服务名称
	config.Addr = *addr     //服务地址
	config.NodeId = *nodeID //服务节点
	config.AccessToken = "123456"
	config.RegistryAddrs = []string{"http://127.0.0.1:2379"} //etcd 注册中心
	//配置限流器（可选）
	rateLimit := int64(10000)
	config.RateLimter = ratelimit.NewBucketWithRate(float64(rateLimit), rateLimit)
	//配置限流器（可选）

	//config.MonitorListenAddr = *monitorListenAddr //配置prometheus采集地址（可选）

	//配置zipkin（可选）
	collector, err := zipkin.NewHTTPCollector(ZIPKIN_HTTP_ENDPOINT)
	if err != nil {
		log.Fatalf("zipkin.NewHTTPCollector err: %v", err)
	}
	recorder := zipkin.NewRecorder(collector, true, ZIPKIN_RECORDER_HOST_PORT, config.Name+".Server")
	tracer, err := zipkin.NewTracer(
		recorder, zipkin.ClientServerSameSpan(false),
	)
	if err != nil {
		log.Fatalf("zipkin.NewTracer err: %v", err)
	}
	config.Tracer = tracer
	//配置zipkin（可选）

	service, err := server.NewSevice(config)
	if err != nil {
		log.Fatalf("server.NewSevice err: %v", err)
	}
	example := new(ExampleService)
	idWordker, err := goSnowFlake.NewIdWorker(1)
	if err != nil {
		log.Fatalf("SnowFlakerver.NewIdWorker err: %v", err)
	}
	example.IdWorker = idWordker
	spb.RegisterExampleServiceServer(service.GrpcServer, example) //注册服务
	service.Run()                                                 //运行
}

type ExampleService struct {
	IdWorker *goSnowFlake.IdWorker
}

func (s *ExampleService) GetExample(ctx context.Context, request *spb.GetExampleRequest) (response *spb.Example, err error) {
	var (
		appSecret []byte
	)
	if request.Value == "" {
		response = nil
		err = grpc_error.InvalidArgument("value", "输入的值不能为空！")
	} else {
		id, _ := s.IdWorker.NextId()
		log.Println("request-id:" + strconv.FormatInt(id, 10) + ",service-node:" + *nodeID)
		appSecret, err = rsa.RsaEncrypt([]byte("605f81a5fd7c623a22fb9b2ee33ad49f"), rsa.Base64Decode(request.Value))
		if err != nil {
			response = nil
			err = grpc_error.InvalidArgument("value", "RSA公钥不正确！")
		} else {
			response = &spb.Example{
				Value: rsa.Base64Encode(appSecret),
			}
			err = nil

		}
	}
	return
}
