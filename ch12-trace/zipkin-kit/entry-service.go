package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/gauge2009/micro-golang/ch12-trace/zipkin-kit/client"

	"github.com/go-kit/kit/log"
	kitzipkin "github.com/go-kit/kit/tracing/zipkin"
	"github.com/openzipkin/zipkin-go"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"google.golang.org/grpc"

	"os"

	//"testing"
	"time"
)

func main() {
	var (
		grpcAddr    = flag.String("addr", ":9008", "gRPC address")
		serviceHost = flag.String("service.host", "localhost", "service ip address")
		servicePort = flag.String("service.port", "8009", "service port")
		zipkinURL   = flag.String("zipkin.url", "http://127.0.0.1:9411/api/v2/spans", "Zipkin server url")
	)
	flag.Parse()
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}
	var zipkinTracer *zipkin.Tracer
	{
		var (
			err           error
			hostPort      = *serviceHost + ":" + *servicePort
			serviceName   = "entry-service"
			useNoopTracer = (*zipkinURL == "")

			reporter = zipkinhttp.NewReporter(*zipkinURL) // 报
		)
		defer reporter.Close()
		zEP, _ := zipkin.NewEndpoint(serviceName, hostPort)
		zipkinTracer, err = zipkin.NewTracer(
			reporter, zipkin.WithLocalEndpoint(zEP), zipkin.WithNoopTracer(useNoopTracer),
		) // 跨（跨度模型）
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
		if !useNoopTracer {
			logger.Log("tracer", "Zipkin", "type", "Native", "URL", *zipkinURL)
		}
	}
	tr := zipkinTracer                       // 跨
	parentSpan := tr.StartSpan("entry-test") // 跨（依赖服务）
	defer parentSpan.Flush()

	ctx := zipkin.NewContext(context.Background(), parentSpan) // 跨（跨度上下文）

	clientTracer := kitzipkin.GRPCClientTrace(tr) // 传 （跨度）
	conn, err := grpc.Dial(*grpcAddr, grpc.WithInsecure(), grpc.WithTimeout(1*time.Second))
	if err != nil {
		fmt.Println("gRPC dial err:", err)
	}
	defer conn.Close()

	svr := client.StringDiff(conn, clientTracer)
	result, err := svr.Diff(ctx, "Add", "ppsdd") // 传（跨度上下文）
	if err != nil {
		fmt.Println("Diff error", err.Error())

	}

	fmt.Println("result =", result)
}
