package main

import (
	"log"
	"time"

	"github.com/chuangyou/qsf/client"
	spb "github.com/chuangyou/qsf/examples/pb"
	"github.com/chuangyou/qsf/plugin/breaker"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"golang.org/x/net/context"
)

var (
	BreakerRate     = float64(0.95)
	BreakMinSamples = int64(100)
)

const (
	ZIPKIN_HTTP_ENDPOINT      = "http://127.0.0.1:9411/api/v1/spans"
	ZIPKIN_RECORDER_HOST_PORT = "127.0.0.1:0"
)

func main() {
	config := new(client.Config)
	config.Name = "example"
	config.AccessToken = "123456"                                         //服务密钥
	config.AccessTokenFunc = new(client.ServiceCredential)                //授权方法
	config.RegistryAddrs = []string{"http://127.0.0.1:2379"}              //etcd 注册中心
	config.Breaker = breaker.NewRateBreaker(BreakerRate, BreakMinSamples) //熔断器

	//配置zipkin（可选）
	collector, err := zipkin.NewHTTPCollector(ZIPKIN_HTTP_ENDPOINT)
	if err != nil {
		log.Fatalf("zipkin.NewHTTPCollector err: %v", err)
	}
	recorder := zipkin.NewRecorder(collector, true, ZIPKIN_RECORDER_HOST_PORT, config.Name+".Client")
	tracer, err := zipkin.NewTracer(
		recorder, zipkin.ClientServerSameSpan(false),
	)
	if err != nil {
		log.Fatalf("zipkin.NewTracer err: %v", err)
	}
	config.Tracer = tracer
	//配置zipkin（可选）

	c, err := client.NewClient(config, false) //网关使用第二个参数为true
	if err == nil {
		pbc := spb.NewExampleServiceClient(c.GrpcConn)
		defer c.GrpcConn.Close()
		time.Sleep(time.Second * 1)
		for {
			r, err := pbc.GetExample(context.Background(), &spb.GetExampleRequest{Value: "ddd"})
			log.Println(r.GetValue(), err)
			time.Sleep(time.Second * 1)
		}
	}
}
