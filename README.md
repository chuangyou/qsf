# QSF2.0 - 服务治理框架

------

QSF2.0是基于GRPC生态圈打造的一个简单、易用、功能强大的服务治理框架。

> * 简单易用
        易于入门，易于开发，易于集成,，易于发布，易于监控。
> * 高性能
        基于GRPC（HTTP2、Protocol Buffer）高性能传输。
> * 跨平台、多语言
        由于采用GRPC，即可很容易部署在Windows/Linux/MacOS等平台，同时也支持各种编程语言的调用。
> * 服务发现
        除了直连外，目前支持ETCD注册中心。
> * 服务治理
        目前支持随机、轮询、权重等负载均衡算法，支持限流、熔断、降级等服务保护手段，支持基于prometheus+alertmanager实现的服务监控以及告警（可用grafana展示），支持基于opentracing实现的服务调用链追踪。
> * API网关
        目前接入GRPC-GATEWAY。
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

3、安装GRPC-API-GATEWAY（用于生成网关代码，具体请参考grpc-gateway生成方式）

    go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
    go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger

4、安装QSF

    go get github.com/chuangyou/qsf
    
5、QSF2.0与QSF的区别

 - 封装底层实现
 - 调用简单
 - 所有组件可以一键配置
 - 具体请参考  [examples][1]

  [1]: https://github.com/chuangyou/qsf/tree/master/examples