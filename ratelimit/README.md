服务限流
----

限流在日常生活中也很常见，比如节假日你去一个旅游景点，为了不把景点撑爆，管理部门通常会在外面设置拦截，限制景点的进入人数（等有人出来之后，再放新的人进去）。

对应到计算机中，比如要搞活动，秒杀等，通常都会限流。

说到限流，有个关键问题就是：你根据什么策略进行限制？？

比如在Hystrix中，如果是线程隔离，可以通过线程数 + 队列大小限制；如果是信号量隔离，可以设置最大并发请求数。

另外一个常见的策略就是根据QPS限制，比如我知道我调用的一个db服务，qps是3000，那如果不限制，超过3000，db就可能被打爆。这个时候，我可用在服务端做这个限流逻辑，也可以在客户端做。

现在一般成熟的RPC框架，都有参数直接设置这个。

还有一些场景下，可用限制总数：比如连接数，业务层面限制“库存“总量等等。。

限流的技术原理 －令牌桶算法
--------------

关于限流的原理，相信很多人都听说过令牌桶算法，Guava的RateLimiter也已经有成熟做法，这个自己去搜索之。

此处想强调的是，令牌桶算法针对的是限制“速率“。至于其他限制策略，比如限制总数，限制某个业务量的count值，则要具体业务场景具体分析。

使用方法
----
```go
    package main
    import (
        "github.com/chuangyou/qsf/ratelimit"
    	"github.com/grpc-ecosystem/go-grpc-middleware"
    	...
    )
    func main(){
        ...
        //限制1万并发
        rateLimter := ratelimit.NewBucketWithRate(10000, 10000)
        s := grpc.NewServer(
            grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
    			ratelimit.UnaryServerInterceptor(rateLimter),
    		)),
        )
        //registe service
        ...
    }
```