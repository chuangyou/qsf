package main

import (
	"log"
	"net"

	spb "github.com/chuangyou/qsf2/examples/pb"
	"github.com/chuangyou/qsf2/grpc_error"
	"github.com/chuangyou/qsf2/server"
	etcd "github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
)

func main() {
	config := new(server.Config)
	config.Version = "V1"           //服务版本
	config.Name = "example"         //服务名称
	config.Path = "/qsf.service.v1" //服务根路径
	config.Addr = "0.0.0.0:7145"    //服务地址
	config.Token = "123456"         //服务授权token（此项不为空时客户端需要配置对应的token才有权限访问服务）
	config.RateLimit = 0            //服务QPS控制（0为不限）
	//etcd end_points
	config.EtcdConfig = etcd.Config{
		Endpoints: []string{
			"http://127.0.0.1:2379",
		},
	}
	config.Monitor = true                                             //是否开启服务监控（依托prometheus）
	config.MonitorListenAddr = "0.0.0.0:9094"                         //监控对外开放的采集接口
	config.ZipkinEnable = true                                        //开启zipkin
	config.ZipkinCollectorAddr = "http://127.0.0.1:9411/api/v1/spans" //目前只支持Http Collector（将逐步开放其他Collector）
	config.ZipkinRecorderAddr = "127.0.0.1:0"                         //zipkin recorder addr
	s, err := server.NewService(config)                               //新建服务
	if err != nil {
		log.Fatalf("new a service error", err)
	}
	spb.RegisterExampleServiceServer(s.Server, &exampleService{}) //注册服务
	if s.ZipkinCollector != nil {
		//只有开启zipkin时ZipkinCollector的值不为nil
		defer s.ZipkinCollector.Close()
	}
	if s.GrpcMetrics != nil {
		//初始化服务监控（只有config.Monitor = true时GrpcMetrics的值不为nil）
		s.GrpcMetrics.InitializeMetrics(s.Server)
		go func() {
			if err := s.MonitorHttpServer.ListenAndServe(); err != nil {
				log.Fatalf("Unable to start a http server.")
			}
		}()
	}
	lis, err := net.Listen("tcp", config.Addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	if err := s.Server.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}

type exampleService struct{}

func (s *exampleService) GetExample(ctx context.Context, request *spb.GetExampleRequest) (response *spb.Example, err error) {
	if request.Value == "" {
		response = nil
		err = grpc_error.InvalidArgument("value", "输入的值不能为空！")
	} else {
		response = &spb.Example{
			Value: "您输入的值是：" + request.Value,
		}
		err = nil
	}
	return
}
