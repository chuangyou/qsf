package main

import (
	"log"
	"time"

	"github.com/chuangyou/qsf2/client"
	spb "github.com/chuangyou/qsf2/examples/pb"
	etcd "github.com/coreos/etcd/clientv3"
	"github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func main() {
	//使用GRPC客户端调用
	config := new(client.Config)
	config.Name = "example"
	config.Path = "/qsf.service.v1"
	config.EtcdConfig = etcd.Config{
		Endpoints: []string{
			"http://127.0.0.1:2379",
		},
	}
	config.Token = "123456"
	config.TokenFunc = new(ExampleServiceCredential)
	//初始化zipkin
	collector, err := zipkin.NewHTTPCollector("http://127.0.0.1:9411/api/v1/spans")
	if err != nil {
		panic(err)

	}
	defer collector.Close()
	tracer, err := zipkin.NewTracer(
		zipkin.NewRecorder(collector, false, "127.0.0.1:0", "Example.Client.V1"),
		zipkin.ClientServerSameSpan(true),
		zipkin.TraceID128Bit(true),
	)
	if err != nil {
		panic(err)
	}
	opentracing.InitGlobalTracer(tracer)
	config.ZipkinEnable = true
	config.ZipkinTrance = tracer
	//初始化zipkin

	//开启熔断器
	config.BreakerEnable = true
	c, err := client.NewClient(config)
	if err == nil {
		conn, err := grpc.Dial("", c.GrpcOpts...)
		if err == nil {
			defer conn.Close()
			pbc := spb.NewExampleServiceClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			r, err := pbc.GetExample(ctx, &spb.GetExampleRequest{Value: ""})
			log.Println(r.GetValue(), err)
		}
	}

}

type ExampleServiceCredential struct {
	serviceToken string
}

func (c *ExampleServiceCredential) SetServiceToken(token string) {
	c.serviceToken = token
}
func (c *ExampleServiceCredential) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "basic " + c.serviceToken,
	}, nil
}
func (c *ExampleServiceCredential) RequireTransportSecurity() bool {
	return false
}
