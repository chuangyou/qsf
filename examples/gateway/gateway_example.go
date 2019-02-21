package main

import (
	"log"
	"net/http"
	"time"

	"github.com/chuangyou/qsf/client"
	spb "github.com/chuangyou/qsf/examples/pb"
	"github.com/chuangyou/qsf/grpc_error"
	"github.com/chuangyou/qsf/plugin/breaker"
	"github.com/chuangyou/qsf/plugin/gateway/runtime"
	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"golang.org/x/net/context"
	"golang.org/x/net/http2"
	"google.golang.org/grpc/metadata"
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
	var (
		mux         *runtime.ServeMux
		ctx         context.Context
		cancel      context.CancelFunc
		http2Server http.Server
	)
	ctx = context.Background()
	ctx, cancel = context.WithCancel(ctx)
	defer cancel()
	mux = runtime.NewServeMux(
		runtime.WithForwardResponseOption(ForwardResponseFilter), //响应过滤器
		runtime.WithMetadata(MetaDataJoin),                       //custom meta
	)
	runtime.HTTPError = grpc_error.CustomHTTPError              //自定义HTTP错误
	runtime.OtherErrorHandler = grpc_error.CustomOtherHTTPError //自定义HTTP错误
	runtime.DefaultContextTimeout = time.Second * 10            //默认超时

	//配置zipkin（可选）
	collector, err := zipkin.NewHTTPCollector(ZIPKIN_HTTP_ENDPOINT)
	if err != nil {
		log.Fatalf("zipkin.NewHTTPCollector err: %v", err)
	}
	recorder := zipkin.NewRecorder(collector, true, ZIPKIN_RECORDER_HOST_PORT, "QSF.Api-Gateway")
	tracer, err := zipkin.NewTracer(
		recorder, zipkin.ClientServerSameSpan(false),
	)
	if err != nil {
		log.Fatalf("zipkin.NewTracer err: %v", err)
	}

	//配置zipkin（可选）
	//配置熔断器（可选）
	breakerP := breaker.NewRateBreaker(BreakerRate, BreakMinSamples)

	initExampleService(ctx, mux, breakerP, tracer)

	http2Server.Handler = AuthHandle(mux) //自定义请求过滤器
	http2Server.Addr = "0.0.0.0:8082"
	http2.ConfigureServer(&http2Server, &http2.Server{})
	go func() {
		err = http2Server.ListenAndServeTLS("./server.pem", "./server.key")
		if err != nil {
			log.Fatalf("ListenAndServeTLS err: %v", err)
		}
	}()
	client.HandleSignal(http2Server)
}

func initExampleService(ctx context.Context, mux *runtime.ServeMux, breaker *breaker.Breaker, tracer opentracing.Tracer) {
	config := new(client.Config)
	config.Name = "example"
	config.AccessToken = "123456"                            //服务密钥
	config.AccessTokenFunc = new(client.ServiceCredential)   //授权方法
	config.RegistryAddrs = []string{"http://127.0.0.1:2379"} //etcd 注册中心
	config.Breaker = breaker                                 //熔断器
	config.Tracer = tracer
	c, err := client.NewClient(config, true)
	if err == nil {
		err = spb.RegisterExampleServiceHandlerFromEndpoint(ctx, mux, "", c.GrpcOpts)
	} else {
		log.Fatalf("initExampleService err: %v", err)
	}

}
func MetaDataJoin(ctx context.Context, r *http.Request) metadata.MD {
	return metadata.New(map[string]string{"qsf-userid": "uid"})
}
func ForwardResponseFilter(ctx context.Context, w http.ResponseWriter, resp proto.Message) error {
	w.Header().Del("Grpc-Metadata-Content-Type")
	return nil
}
func AuthHandle(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//TODO your codes
		h.ServeHTTP(w, r)
	})
}
