# QSF - 服务治理框架

------

QSF是基于GRPC生态圈打造的一个简单、易用、功能强大的服务治理框架。

> * 简单易用
        易于入门，易于开发，易于集成,，易于发布，易于监控。
> * 高性能
        基于GRPC（HTTP2、Protocol Buffer）高性能传输。
> * 跨平台、多语言
        由于采用GRPC，即可很容易部署在Windows/Linux/MacOS等平台，同时也支持各种编程语言的调用。
> * 服务发现
        除了直连外，目前支持ETCD注册中心。
> * 服务治理
        目前支持随机、轮询、权重等负载均衡算法，支持限流、熔断、降级等服务保护手段，支持基于prometheus实现的服务监控（可用grafana展示以及做告警），支持基于opentracing实现的服务调用链追踪。
> * API网关
        目前接入GRPC生态圈的GRPC-GATEWAY（对其进行了适当的二次开发，更适用于本框架）。
# 快速开始
1、安装ProtocolBuffers

    mkdir tmp
    cd tmp
    git clone https://github.com/google/protobuf
    cd protobuf
    ./autogen.sh
    ./configure
    make
    make check
    sudo make install
    
2、安装proto,protoc-gen-go

    go get -u github.com/golang/protobuf/{proto,protoc-gen-go}

3、安装GRPC-API-GATEWAY（需要网关就必须安装此依赖）

    go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
    go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

4、安装QSF

    go get github.com/chuangyou/qsf
5、hellowolrd（GRPC服务例子）

proto:
```proto
syntax = "proto3";

option java_multiple_files = true;
option java_package = "io.grpc.examples.helloworld";
option java_outer_classname = "HelloWorldProto";

package helloworld;

// The greeting service definition.
service Greeter {
  // Sends a greeting
  rpc SayHello (HelloRequest) returns (HelloReply) {}
}

// The request message containing the user's name.
message HelloRequest {
  string name = 1;
}

// The response message containing the greetings
message HelloReply {
  string message = 1;
}
```
server:
```go
package main

import (
	"log"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":50051"
)

// server is used to implement helloworld.GreeterServer.
type server struct{}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	return &pb.HelloReply{Message: "Hello " + in.Name}, nil
}

func main() {
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterGreeterServer(s, &server{})
	// Register reflection service on gRPC server.
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
```
client:

```go
package main

import (
	"log"
	"os"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
)

const (
	address     = "localhost:50051"
	defaultName = "world"
)

func main() {
	// Set up a connection to the server.
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)

	// Contact the server and print out its response.
	name := defaultName
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r.Message)
}
```

6、QSF和GRPC结合的例子（后续将陆续更新）

 - [熔断器][1]
 - [API网关][2]
 - [服务发现][3]
 - [负载均衡][4]
 - [服务监控][5]
 - [服务限流][6]
 - [分布式链路追踪][7]


  [1]: https://github.com/chuangyou/qsf/tree/master/breaker
  [2]: https://github.com/chuangyou/qsf/tree/master/gateway
  [3]: https://github.com/chuangyou/qsf/tree/master/loadbalance
  [4]: https://github.com/chuangyou/qsf/tree/master/loadbalance
  [5]: https://github.com/chuangyou/qsf/tree/master/prometheus
  [6]: https://github.com/chuangyou/qsf/tree/master/ratelimit
  [7]: https://github.com/chuangyou/qsf/tree/master/tracing