# QSF - 服务治理框架

------

QSF是福建创游网络基于GRPC生态圈打造的一个简单、易用、功能强大的服务治理框架。

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
5、例子（后续将陆续更新）