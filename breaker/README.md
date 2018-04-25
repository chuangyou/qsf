背景
--

伴随着业务复杂性的提高，系统的不断拆分，一个面向用户端的API，其内部的RPC调用层层嵌套，调用链条可能会非常长。这会造成以下几个问题：

API接口可用性降低
----------

引用Hystrix官方的一个例子，假设tomcat对外提供的一个application，其内部依赖了30个服务，每个服务的可用性都很高，为99.99%。那整个applicatiion的可用性就是：99.99%的30次方 ＝ 99.7%，即0.3%的失败率。

这也就意味着，每1亿个请求，有30万个失败；按时间来算，就是每个月的故障时间超过2小时。

系统被block
--------

假设一个请求的调用链上面有10个服务，只要这10个服务中有1个超时，就会导致这个请求超时。 
更严重的，如果该请求的并发数很高，所有该请求在短时间内都被block（等待超时），tomcat的所有线程都block在此请求上，导致其他请求没办法及时响应。

服务熔断
----

为了解决上述问题，服务熔断的思想被提出来。类似现实世界中的“保险丝”，当某个异常条件被触发，直接熔断整个服务，而不是一直等到此服务超时。 
熔断的触发条件可以依据不同的场景有所不同，比如统计一个时间窗口内失败的调用次数。

使用方法：
-----
    package main
    import (
        "github.com/chuangyou/qsf/breaker"
    	"github.com/grpc-ecosystem/go-grpc-middleware"
    	...
    )
    func main(){
        ...
        //最后100个调用结果采样，95%失败启动熔断器
        breakerInterceptor := breaker.NewRateBreaker(0.95, 100) 
        grpcDialOpts := []grpc.DialOption{
        	grpc.WithUnaryInterceptor(
        		grpc.WithInsecure(),
        		grpc_middleware.ChainUnaryClient(
        			breaker.UnaryClientInterceptor(breakerInterceptor),
        		),
        	),
        }
        grpc.Dial("service addr",grpcDialOpts)
        ...
    }




