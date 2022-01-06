package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gauge2009/micro-golang/ch12-trace/zipkin-kit/pb"
	endpointing "github.com/gauge2009/micro-golang/ch12-trace/zipkin-kit/string-service/endpoint"
	"github.com/gauge2009/micro-golang/ch12-trace/zipkin-kit/string-service/plugins"
	"github.com/gauge2009/micro-golang/ch12-trace/zipkin-kit/string-service/service"
	"github.com/go-kit/kit/log"
	kitzipkin "github.com/go-kit/kit/tracing/zipkin"
	"github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var zipkinTracer *zipkin.Tracer // 全局变量
// 创建环境变量
var (
	consulHost = flag.String("consul.host", "127.0.0.1", "consul server ip address")
	consulPort = flag.String("consul.port", "8500", "consul server port")
	zipkinURL  = flag.String("zipkin.url", "http://127.0.0.1:9411/api/v2/spans", "Zipkin server url")
)

//获取单例对象的方法，引用传递返回
func GetTracer() *zipkin.Tracer {
	if zipkinTracer == nil {
		useNoopTracer := (*zipkinURL == "")
		hostPort := "localhost:9292"
		serviceName := "string-service"
		reporter := zipkinhttp.NewReporter(*zipkinURL)
		zEP, _ := zipkin.NewEndpoint(serviceName, hostPort)
		zipkinTracer, _ = zipkin.NewTracer(
			reporter, zipkin.WithLocalEndpoint(zEP), zipkin.WithNoopTracer(useNoopTracer),
		)
		fmt.Println("zipkin.Tracer in service 实例化一次")
		//zipkinTracer = new(Tracer)
	}
	return zipkinTracer
}

func main() {

	var (
		//consulHost  = flag.String("consul.host", "127.0.0.1", "consul ip address")
		//consulPort  = flag.String("consul.port", "8500", "consul port")
		serviceHost = flag.String("service.host", "localhost", "service ip address")
		servicePort = flag.String("service.port", "9009", "service port")
		//zipkinURL   = flag.String("zipkin.url", "http://127.0.0.1:9411/api/v2/spans", "Zipkin server url")
		grpcAddr = flag.String("grpc", ":9008", "gRPC listen address.")
	)

	flag.Parse()

	ctx := context.Background()
	errChan := make(chan error)

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	// 全局变量
	zipkinTracer = GetTracer()

	var svc service.Service
	svc = service.StringService{}

	// add logging middleware to service
	svc = plugins.LoggingMiddleware(logger)(svc)

	endpoint := endpointing.MakeStringEndpoint(ctx, svc)
	endpoint = kitzipkin.TraceEndpoint(zipkinTracer, "string-endpoint")(endpoint)

	//创建健康检查的Endpoint
	healthEndpoint := endpointing.MakeHealthCheckEndpoint(svc)
	healthEndpoint = kitzipkin.TraceEndpoint(zipkinTracer, "health-endpoint")(healthEndpoint)

	//把算术运算Endpoint和健康检查Endpoint封装至StringEndpoints
	endpts := endpointing.StringEndpoints{
		StringEndpoint:      endpoint,
		HealthCheckEndpoint: healthEndpoint,
	}

	//创建http.Handler
	r := plugins.MakeHttpHandler(ctx, endpts, zipkinTracer, logger)

	//创建注册对象
	registar := plugins.Register(*consulHost, *consulPort, *serviceHost, *servicePort, logger)

	go func() {
		fmt.Println("Http Server start at port:" + *servicePort)
		//启动前执行注册
		registar.Register()
		handler := r
		errChan <- http.ListenAndServe(":"+*servicePort, handler)
	}()
	//grpc server
	go func() {
		fmt.Println("grpc Server start at port" + *grpcAddr)
		listener, err := net.Listen("tcp", *grpcAddr)
		if err != nil {
			errChan <- err
			return
		}
		serverTracer := kitzipkin.GRPCServerTrace(zipkinTracer, kitzipkin.Name("string-grpc-transport"))

		handler := plugins.NewGRPCServer(ctx, endpts, serverTracer)
		gRPCServer := grpc.NewServer()
		pb.RegisterStringServiceServer(gRPCServer, handler)
		errChan <- gRPCServer.Serve(listener)
	}()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()

	error := <-errChan
	//服务退出取消注册
	registar.Deregister()
	fmt.Println(error)
}
