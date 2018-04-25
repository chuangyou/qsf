# gRPC服务发现及负载均衡实现

gRPC开源组件官方并未直接提供服务注册与发现的功能实现，但其设计文档已提供实现的思路，并在不同语言的gRPC代码API中已提供了命名解析和负载均衡接口供扩展。
![结构图][1]

其基本实现原理
-------
服务启动后gRPC客户端向命名服务器发出名称解析请求，名称将解析为一个或多个IP地址，每个IP地址标示它是服务器地址还是负载均衡器地址，以及标示要使用那个客户端负载均衡策略或服务配置。

客户端实例化负载均衡策略，如果解析返回的地址是负载均衡器地址，则客户端将使用grpclb策略，否则客户端使用服务配置请求的负载均衡策略。

负载均衡策略为每个服务器地址创建一个子通道（channel）。

当有rpc请求时，负载均衡策略决定那个子通道即grpc服务器将接收请求，当可用服务器为空时客户端的请求将被阻塞。

根据gRPC官方提供的设计思路，基于进程内LB方案（阿里开源的服务框架 Dubbo 也是采用类似机制），结合分布式一致的组件（如Zookeeper、Consul、Etcd），可找到gRPC服务发现和负载均衡的可行解决方案。

在QSF中使用ETCD进行服务发现（含负载均衡）
------------------------
server:

```go
package main
import (
	etcd "github.com/coreos/etcd/clientv3"
	registry "github.com/chuangyou/qsf/loadbalance/registry/etcd"
	"time"
	...
)
func main(){
    ...
	etcdConfg := etcd.Config{
		Endpoints: []string{
			"http://127.0.0.1:2379",
		},
	}
	registry, err := registry.NewRegistry(
		registry.Option{
			EtcdConfig:  etcdConfg,
			RegistryDir: "/exampleservice.v1",
			ServiceName: "example",
			NodeID:      "node3",
			NData: registry.NodeData{
				Addr: "127.0.0.1:500053",
			},
			Ttl: 60 * time.Second,
		})
	if err != nil {
	    panic(err)
    }
    s := grpc.NewServer()
    //registe service
    go func() {
		if err := registry.Register(); err != nil {
			log.Fatalf("service registe failed: %v", err)
		} else {
			log.Println("service registe")
		}
	}()
    ...
}
```
client:

```go
package main
import (
	grpclb "github.com/chuangyou/qsf/loadbalance"
   	registry "github.com/chuangyou/qsf/loadbalance/registry/etcd"
	etcd "github.com/coreos/etcd/clientv3"
	...
)
func main(){
    ...
	etcdConfg := etcd.Config{
		Endpoints: []string{
			"http://127.0.0.1:2379",
		},
	}
	r := registry.NewResolver("/exampleservice.v1", "example", etcdConfg)
	b := grpclb.NewBalancer(r, grpclb.NewRoundRobinSelector())
	grpcDialOpts := []grpc.DialOption{
    	grpc.WithUnaryInterceptor(
    		grpc.WithInsecure(),
		    grpc.WithBalancer(b),
    	),
    }
    grpc.Dial("service addr",grpcDialOpts)
    ...
}
```

  [1]: https://segmentfault.com/img/bVKyoo?w=554&h=243