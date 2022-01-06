package main

import (
	"flag"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/hashicorp/consul/api"
	"github.com/openzipkin/zipkin-go"
	zipkinhttpsvr "github.com/openzipkin/zipkin-go/middleware/http"
	//"github.com/openzipkin/zipkin-go/reporter"
	zipkinhttp "github.com/openzipkin/zipkin-go/reporter/http"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
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
		serviceName := "gateway-service"
		reporter := zipkinhttp.NewReporter(*zipkinURL)
		zEP, _ := zipkin.NewEndpoint(serviceName, hostPort)
		zipkinTracer, _ = zipkin.NewTracer(
			reporter, zipkin.WithLocalEndpoint(zEP), zipkin.WithNoopTracer(useNoopTracer),
		)
		fmt.Println("zipkin.Tracer in gateway 实例化一次")
		//zipkinTracer = new(Tracer)
	}
	return zipkinTracer
}
func main() {
	flag.Parse()

	//创建日志组件
	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	//var (
	//	err           error
	//	//hostPort      = "localhost:9292"
	//	//serviceName   = "gateway-service"
	//	useNoopTracer = (*zipkinURL == "")
	//	reporter      = zipkinhttp.NewReporter(*zipkinURL)
	//)
	//defer reporter.Close()
	//zEP, _ := zipkin.NewEndpoint(serviceName, hostPort)
	// 全局变量
	zipkinTracer = GetTracer()
	//fmt.Println(zipkinTracer.)

	//if err != nil {
	//	logger.Log("err", err)
	//	os.Exit(1)
	//}
	//if !useNoopTracer {
	//	logger.Log("tracer", "Zipkin", "type", "Native", "URL", *zipkinURL)
	//}

	// 创建consul api客户端
	consulConfig := api.DefaultConfig()
	consulConfig.Address = "http://" + *consulHost + ":" + *consulPort
	consulClient, err := api.NewClient(consulConfig)
	if err != nil {
		logger.Log("err", err)
		os.Exit(1)
	}

	//创建反向代理
	proxy := SmartReverseProxy(consulClient, zipkinTracer, logger)

	tags := map[string]string{
		"component": "gateway_server",
	}

	handler := zipkinhttpsvr.NewServerMiddleware(
		zipkinTracer,
		zipkinhttpsvr.SpanName("gateway"),
		zipkinhttpsvr.TagResponseSize(true),
		zipkinhttpsvr.ServerTags(tags),
	)(proxy)

	errc := make(chan error)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	//开始监听
	go func() {
		logger.Log("transport", "HTTP", "addr", "9292")
		errc <- http.ListenAndServe(":9292", handler)
	}()

	// 开始运行，等待结束
	logger.Log("exit", <-errc)
}

//SmartReverseProxy 创建反向代理处理方法
func SmartReverseProxy(client *api.Client, zikkinTracer *zipkin.Tracer, logger log.Logger) *httputil.ReverseProxy {

	//创建Director
	director := func(req *http.Request) {

		//查询原始请求路径，如：/string-service/op/10/5
		reqPath := req.URL.Path
		if reqPath == "" {
			return
		}
		//按照分隔符'/'对路径进行分解，获取服务名称serviceName
		pathArray := strings.Split(reqPath, "/")
		serviceName := pathArray[1]

		//调用consul api查询serviceName的服务实例列表
		result, _, err := client.Catalog().Service(serviceName, "", nil)
		if err != nil {
			logger.Log("ReverseProxy failed", "query service instace error", err.Error())
			return
		}

		if len(result) == 0 {
			logger.Log("ReverseProxy failed", "no such service instance", serviceName)
			return
		}

		//重新组织请求路径，去掉服务名称部分
		destPath := strings.Join(pathArray[2:], "/")

		//随机选择一个服务实例
		tgt := result[rand.Int()%len(result)]
		logger.Log("service id", tgt.ServiceID)

		//设置代理服务地址信息
		req.URL.Scheme = "http"
		req.URL.Host = fmt.Sprintf("%s:%d", tgt.ServiceAddress, tgt.ServicePort)
		req.URL.Path = "/" + destPath

	}

	// 为反向代理增加追踪逻辑，使用如下RoundTrip代替默认Transport
	roundTrip, _ := zipkinhttpsvr.NewTransport(zikkinTracer, zipkinhttpsvr.TransportTrace(true))

	return &httputil.ReverseProxy{
		Director:  director,
		Transport: roundTrip,
	}
}
