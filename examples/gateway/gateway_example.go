package main

import (
	"net/http"
	"time"

	"github.com/chuangyou/qsf2/client"
	spb "github.com/chuangyou/qsf2/examples/pb"
	"github.com/chuangyou/qsf2/grpc_error"
	"github.com/chuangyou/qsf2/plugin/gateway/runtime"
	etcd "github.com/coreos/etcd/clientv3"
	"github.com/golang/protobuf/proto"
	"github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin/zipkin-go-opentracing"
	"golang.org/x/net/context"
	"golang.org/x/net/http2"
	"google.golang.org/grpc/metadata"
)

func main() {
	//使用网关
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

	config_gw := new(client.Config)
	config_gw.Name = "example"
	config_gw.Path = "/qsf.service.v1"
	config_gw.EtcdConfig = etcd.Config{
		Endpoints: []string{
			"http://127.0.0.1:2379",
		},
	}
	config_gw.Token = "123456"
	config_gw.TokenFunc = new(ExampleServiceCredential)
	//初始化zipkin
	collector_gw, err := zipkin.NewHTTPCollector("http://127.0.0.1:9411/api/v1/spans")
	if err != nil {
		panic(err)

	}
	defer collector_gw.Close()
	tracer_gw, err := zipkin.NewTracer(
		zipkin.NewRecorder(collector_gw, false, "127.0.0.1:0", "Gateway.V1"),
		zipkin.ClientServerSameSpan(true),
		zipkin.TraceID128Bit(true),
	)
	if err != nil {
		panic(err)
	}
	opentracing.InitGlobalTracer(tracer_gw)
	config_gw.ZipkinEnable = true
	config_gw.ZipkinTrance = tracer_gw
	//初始化zipkin

	//开启熔断器
	config_gw.BreakerEnable = true

	c_gw, err := client.NewClient(config_gw)
	if err == nil {
		err = spb.RegisterExampleServiceHandlerFromEndpoint(ctx, mux, "", c_gw.GrpcOpts)
		if err != nil {
			panic(err)
		}
	} else {
		panic(err)
	}
	//启动APP2服务器（curl https://127.0.0.1:8082/v1/examples/2222 -k）
	http2Server.Handler = AuthHandle(mux) //自定义请求过滤器
	http2Server.Addr = "0.0.0.0:8082"
	http2.ConfigureServer(&http2Server, &http2.Server{})
	err = http2Server.ListenAndServeTLS("./server.pem", "./server.key")
	if err != nil {
		panic(err)
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
