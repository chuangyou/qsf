# 分布式追踪系统

分布式系统为什么需要 Tracing？
-------------------

  先介绍一个概念：分布式跟踪，或分布式追踪。
  电商平台由数以百计的分布式服务构成，每一个请求路由过来后，会经过多个业务系统并留下足迹，并产生对各种Cache或DB的访问，但是这些分散的数据对于问题排查，或是流程优化都帮助有限。对于这么一个跨进程/跨线程的场景，汇总收集并分析海量日志就显得尤为重要。要能做到追踪每个请求的完整调用链路，收集调用链路上每个服务的性能数据，计算性能数据和比对性能指标（SLA），甚至在更远的未来能够再反馈到服务治理中，那么这就是分布式跟踪的目标了。在业界，twitter 的 zipkin 和淘宝的鹰眼就是类似的系统，它们都起源于 Google Dapper 论文，就像历史上 Hadoop 发源于 Google Map/Reduce 论文，HBase 源自 Google BigTable 论文一样。
  好了，整理一下，Google叫Dapper，淘宝叫鹰眼，Twitter叫ZipKin，京东商城叫Hydra，eBay叫Centralized Activity Logging (CAL)，大众点评网叫CAT，我们叫Tracing。
  这样的系统通常有几个设计目标：
（1）低侵入性——作为非业务组件，应当尽可能少侵入或者无侵入其他业务系统，对于使用方透明，减少开发人员的负担；
（2）灵活的应用策略——可以（最好随时）决定所收集数据的范围和粒度；
（3）时效性——从数据的收集和产生，到数据计算和处理，再到最终展现，都要求尽可能快；
（4）决策支持——这些数据是否能在决策支持层面发挥作用，特别是从 DevOps 的角度；
（5）可视化才是王道。

接下来将演示如何在QSF中使用ZIPKIN
---------------------
client:

    package main
    import (
        zipkin "github.com/openzipkin/zipkin-go-opentracing"
        "github.com/opentracing/opentracing-go"
        "github.com/chuangyou/qsf/tracing"
    	"github.com/grpc-ecosystem/go-grpc-middleware"
    	...
    )
    func main(){
        ...
    	//zipkin
    	collector, err := zipkin.NewHTTPCollector("http://localhost:9411/api/v1/spans")
    	if err != nil {
    		panic(err)
    		return
    	}
    	defer collector.Close()
    
    	tracer, err := zipkin.NewTracer(
    		zipkin.NewRecorder(collector, false, "localhost:0", "Client.V1"),
    		zipkin.ClientServerSameSpan(true),
    		zipkin.TraceID128Bit(true),
    	)
    	if err != nil {
    		panic(err)
    	}
    	opentracing.InitGlobalTracer(tracer)
    	grpcDialOpts := []grpc.DialOption{
        	grpc.WithUnaryInterceptor(
        		grpc.WithInsecure(),
        		grpc_middleware.ChainUnaryClient(
        			otgrpc.OpenTracingClientInterceptor(tracer),
        		),
        	),
        }
        grpc.Dial("service addr",grpcDialOpts)
        ...
    }

server:

    package main
    import (
        zipkin "github.com/openzipkin/zipkin-go-opentracing"
        "github.com/opentracing/opentracing-go"
        "github.com/chuangyou/qsf/tracing"
    	"github.com/grpc-ecosystem/go-grpc-middleware"
    	...
    )
    func main(){
        ...
    	//zipkin
    	collector, err := zipkin.NewHTTPCollector("http://localhost:9411/api/v1/spans")
    	if err != nil {
    		panic(err)
    		return
    	}
    	defer collector.Close()
    
    	tracer, err := zipkin.NewTracer(
    		zipkin.NewRecorder(collector, false, "localhost:0", "Client.V1"),
    		zipkin.ClientServerSameSpan(true),
    		zipkin.TraceID128Bit(true),
    	)
    	if err != nil {
    		panic(err)
    	}
    	opentracing.InitGlobalTracer(tracer)
         s := grpc.NewServer(
            grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
    			grpc.UnaryServerInterceptor(otgrpc.OpenTracingServerInterceptor(tracer,otgrpc.LogPayloads())),
    		)),
        )
        //registe service
        ...
    }



