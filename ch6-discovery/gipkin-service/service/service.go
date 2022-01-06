package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/opentracing/opentracing-go"
	zipkin "github.com/openzipkin-contrib/zipkin-go-opentracing"
	"go.etcd.io/etcd/clientv3"
	"os"

	//"github.com/coreos/etcd/clientv3"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"log"
	"strings"
	"time"
	//"../gedis-benchmark/cacheClient"
	"github.com/gauge2009/micro-golang/gedis-benchmark/cacheClient"
)

// Service errors
var (
	ErrMaxSize = errors.New("maximum size of 1024 bytes exceeded")

	ErrStrValue = errors.New("maximum size of 1024 bytes exceeded")
)

// Service Define a service interface
type Service interface {
	/// <param name="SiteCode"></param>
	/// <param name="SpanID">链路ID：如：SpanIDEnum.RabbitMQ_To_Worker</param>
	/// <param name="TraceID">批次ID或会话ID， 从头到尾要唯一</param>
	/// <param name="BizCode">业务代码 如：  TaskSchedulerBizCodeEnum.ATSINSPECT</param>
	/// <param name="ParentID">上游服务</param>
	/// <param name="Level">级别</param>
	/// <param name="ClassName">类名 如：this.GetType().FullName</param>
	/// <param name="MethodName">方法名 如：MethodInfo.GetCurrentMethod().Name</param>
	/// <param name="LocationDesc">描述信息【可空】</param>
	DoTrace(KeyID string, SpanID string, TraceID string, BizCode string, ParentID string, Level string, ClassName string, MethodName string, LocationDesc string) (string, error)

	// Concat a and b
	Concat(a, b string) (string, error)

	// a,b pkg string value
	Diff(a, b string) (string, error)

	// HealthCheck check service health status
	HealthCheck() bool
}

/*

CREATE table dbo.task_link_trace (
   key_id               nvarchar(36)          ,
   span_id              nvarchar(50)       ,
   trace_id             nvarchar(50)        ,
   biz_code             nvarchar(50)        ,
   parent_id            nvarchar(50)        ,
   operate_dt           datetime            ,
   operate_by           nvarchar(50)         null,
   level              nvarchar(10)         null,
   node_id              nvarchar(50)         null,
   thread_id            int       null,
   class_name              nvarchar(100)         null,
   method_name              nvarchar(100)         null,
  location_desc            nvarchar(200)  null,
   constraint PK__task_link_trace primary key (key_id)
         on "PRIMARY"
)
on "PRIMARY"
GO
*/

type Task_link_trace struct {
	//gorm.Model
	Key_id        string    `gorm:"size:36"`
	Span_id       string    `gorm:"size:50"`
	Trace_id      string    `gorm:"size:50"`
	Biz_code      string    `gorm:"size:50"`
	Parent_id     string    `gorm:"size:50"`
	Operate_dt    time.Time `gorm:"type:datetime"`
	Operate_by    string    `gorm:"size:50"`
	Level         string    `gorm:"size:10"`
	Node_id       string    `gorm:"size:50"`
	Thread_id     int       `gorm:"int"`
	Class_name    string    `gorm:"size:100"`
	Method_name   string    `gorm:"size:100"`
	Location_desc string    `gorm:"size:200"`
}

func (this *Task_link_trace) TableName() string {
	return "task_link_trace"
}

//ArithmeticService implement Service interface
type GipkinService struct {
}

///██████████████████████████████████████████████████████████████████
//http://127.0.0.1:21212/op/DoTrace/gauge202112301916/RabbitMQ_To_Worker/12345678-415d-40e1-987a-17a24f83f47c/ATSINSPECT/AtsTaskService/Debug/HRLink.BackendService.AtsInspectService/AtsInspectExecute/持久化完成
func (s GipkinService) DoTrace(KeyID string, SpanID string, TraceID string, BizCode string, ParentID string, Level string, ClassName string, MethodName string, LocationDesc string) (string, error) {
	//Trace(BizCode)
	decimal.DivisionPrecision = 4 // 保留4位小数，如有更多位，则进行四舍五入保留两位小数
	// github.com/denisenkom/go-mssqldb
	dsn := "sqlserver://sa:sparksubmit666@localhost/hive?database=ai_cop"
	db, err := gorm.Open(sqlserver.Open(dsn), &gorm.Config{})
	//db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	// Migrate the schema
	db.AutoMigrate(&Task_link_trace{})
	now := time.Now().Format("2006-01-02 15:04:05") //go语言的诞生时间
	fmt.Println(now)
	t2, err := time.ParseInLocation("2006-01-02 15:04:05", now, time.Local)

	//var num1 float64 = 3.14
	//rh := decimal.NewFromFloat(num1)

	// Create  uuid.NewV4().String()
	entity := BuildEntity(KeyID, SpanID, TraceID, BizCode, ParentID, t2, Level, ClassName, MethodName, LocationDesc)
	//result := db.Create(&Task_link_trace{Key_id: KeyID, Span_id: SpanID,
	//	Trace_id: TraceID, Biz_code: BizCode, Parent_id: ParentID, Operate_dt: t2, Operate_by: "sys",
	//	Level: Level, Node_id: "worker_0x",
	//	Thread_id:  1,
	//	Class_name: ClassName, Method_name: MethodName, Location_desc: LocationDesc,
	//})
	result := db.Create(entity)
	//result :=  db.Create(&Task_link_trace{Key_id: "1111-22-33333", Span_id: SpanID })
	if result != nil {
		if result.Error != nil {
			fmt.Println("result.Error ! = %v+\n", result.Error)
		}
		fmt.Println("RowsAffected = %v+\n", result.RowsAffected)
	}

	/// 调用zipkin
	//TraceByZipkin(BizCode)
	svc_name := BizCode + "@" + TraceID
	TraceByZipkin(svc_name, SpanID, BizCode, ParentID, ClassName, MethodName, LocationDesc)

	// Read
	var task_link_trace Task_link_trace
	db.First(&task_link_trace, "key_id = ?", "1fe7c255-84ae-4224-acbd-c2b116430b9e")

	//Gedis
	input := BuildEntity(KeyID, SpanID, TraceID, BizCode, ParentID, t2, Level, ClassName, MethodName, LocationDesc)
	inputbytes, err0 := json.Marshal(input)
	if err0 != nil {
		log.Panicln("decode  failed:", string(inputbytes), err0)
	}
	fmt.Println(input)
	fmt.Println(string(inputbytes))
	gedis_client_set("gipkin:"+KeyID, string(inputbytes))
	getcd_client_set("gipkin:"+KeyID, string(inputbytes))

	return "success", nil
}

/// 用于链路跟踪：
const (
	// Our service name.
	//serviceName = "DoTrace_Service"

	// Host + port of our service.
	hostPort = "0.0.0.0:0"

	// Endpoint to send Zipkin spans to.
	zipkinHTTPEndpoint = "http://localhost:9411/api/v1/spans"

	// Debug mode.
	debug = false

	// Base endpoint of our SVC1 service.
	svc1Endpoint = "http://localhost:61001"

	// same span can be set to true for RPC style spans (Zipkin V1) vs Node style (OpenTracing)
	sameSpan = true

	// make Tracer generate 128 bit traceID's for root spans.
	traceID128Bit = true
)

// Service constants
const (
	StrMaxSize = 1024
)

var tracer opentracing.Tracer // 全局变量
// 创建环境变量
var (
//consulHost = flag.String("consul.host", "127.0.0.1", "consul server ip address")
//consulPort = flag.String("consul.port", "8500", "consul server port")
//zipkinURL  = flag.String("zipkin.url", "http://127.0.0.1:9411/api/v2/spans", "Zipkin server url")
)

// █ █ █ █ █ 获取单例对象的方法，引用传递返回
// █ █ █ █ █ serviceName 可以把多行trace串起来的唯一标识——如考勤审查批次ID
func GetTracer(serviceName string) opentracing.Tracer {
	if tracer == nil {
		// Create our HTTP collector.
		collector, err := zipkin.NewHTTPCollector(zipkinHTTPEndpoint)
		if err != nil {
			fmt.Printf("unable to create Zipkin HTTP collector: %+v\n", err)
			os.Exit(-1)
		}
		// Create our recorder.
		//recorder := zipkin.NewRecorder(collector, debug, hostPort, serviceName)
		recorder := zipkin.NewRecorder(collector, debug, hostPort, serviceName)
		// Create our tracer.
		tracer, err = zipkin.NewTracer(
			recorder,
			zipkin.ClientServerSameSpan(sameSpan),
			zipkin.TraceID128Bit(traceID128Bit),
		)
		if err != nil {
			fmt.Printf("unable to create Zipkin tracer: %+v\n", err)
			os.Exit(-1)
		}

		// Explicitly set our tracer to be the default tracer.
		opentracing.InitGlobalTracer(tracer)

		fmt.Println("zipkin.Tracer in gateway 实例化一次")
		//zipkinTracer = new(Tracer)
	}
	return tracer
}

// █ █ █ █ █ 链路跟踪主方法 █ █ █ █ █
func TraceByZipkin(serviceName string, SpanID string, BizCode string, ParentID string, ClassName string, MethodName string, LocationDesc string) {
	//// Create our HTTP collector.
	//collector, err := zipkin.NewHTTPCollector(zipkinHTTPEndpoint)
	//if err != nil {
	//	fmt.Printf("unable to create Zipkin HTTP collector: %+v\n", err)
	//	os.Exit(-1)
	//}
	//// Create our recorder.
	////recorder := zipkin.NewRecorder(collector, debug, hostPort, serviceName)
	//recorder := zipkin.NewRecorder(collector, debug, hostPort, serviceName)
	//// Create our tracer.
	//tracer, err := zipkin.NewTracer(
	//	recorder,
	//	zipkin.ClientServerSameSpan(sameSpan),
	//	zipkin.TraceID128Bit(traceID128Bit),
	//)
	//if err != nil {
	//	fmt.Printf("unable to create Zipkin tracer: %+v\n", err)
	//	os.Exit(-1)
	//}
	//
	//// Explicitly set our tracer to be the default tracer.
	tracer = GetTracer(serviceName)

	//opentracing.InitGlobalTracer(tracer)

	//// Create Client to svc1 Service
	//client := svc1.NewHTTPClient(tracer, svc1Endpoint)

	// Create Root Span for duration of the interaction with svc1
	span := opentracing.StartSpan(SpanID)

	// Put root span in context so it will be used in our calls to the client.
	ctx := opentracing.ContextWithSpan(context.Background(), span)

	//// Call the Concat Method
	span.LogEvent("Trace id =" + serviceName + "; SpanID =" + SpanID + "; BizCode =" + BizCode + "; ParentID =" + ParentID + "; ClassName =" + ClassName + "; MethodName =" + MethodName + "; LocationDesc =" + LocationDesc)
	//res1, err := client.Concat(ctx, "Hello", " World!")
	//fmt.Printf("Concat: %s Err: %+v\n", res1, err)
	span = opentracing.SpanFromContext(ctx)
	span.SetTag("BizCode", BizCode)

	//
	//// Call the Sum Method
	span.LogEvent("备注：Trace id就如审查批次ID | SpanID 对应长时任务在当前分布式系统中执行到了那个阶段 ")
	//res2, err := client.Sum(ctx, 10, 20)
	//fmt.Printf("Sum: %d Err: %+v\n", res2, err)
	span = opentracing.SpanFromContext(ctx)
	span.SetTag("本阶段名称", SpanID)
	span.SetTag("上个阶段名称", ParentID)
	span.SetTag("程序类", ClassName)
	span.SetTag("所在方法或函数名", MethodName)
	span.SetTag("补充信息", LocationDesc)

	// Finish our CLI span
	span.Finish()

	// Close collector to ensure spans are sent before exiting.
	//collector.Close()
}

func BuildEntity(KeyID string, SpanID string, TraceID string, BizCode string, ParentID string, CreateDatetime time.Time, Level string, ClassName string, MethodName string, LocationDesc string) *Task_link_trace {
	return &Task_link_trace{Key_id: KeyID, Span_id: SpanID,
		Trace_id: TraceID, Biz_code: BizCode, Parent_id: ParentID, Operate_dt: CreateDatetime, Operate_by: "sys",
		Level: Level, Node_id: "worker_0x",
		Thread_id:  1,
		Class_name: ClassName, Method_name: MethodName, Location_desc: LocationDesc,
	}
}

func gedis_client_set(gedis_key, gedis_val string) {
	//server := flag.String("h", "localhost", "cache server address")
	//op := flag.String("c", "set", "command, could be get/set/del")
	//key := flag.String("k", gedis_key, "key")
	//value := flag.String("v", gedis_val, "value")
	//flag.Parse()
	//client := cacheClient.New("tcp", *server)
	client := cacheClient.New("tcp", "localhost")
	//cmd := &cacheClient.Cmd{*op, *key, *value, nil}
	cmd := &cacheClient.Cmd{"set", gedis_key, gedis_val, nil}
	//./client -c set -k info1 -v  '{  \"result\": \"success\",   \"error\": null }'
	//./client -c get -k info1
	client.Run(cmd)
	if cmd.Error != nil {
		fmt.Println("error:", cmd.Error)
	} else {
		fmt.Println(cmd.Value)
	}
}

func getcd_client_set(gedis_key, gedis_val string) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		// handle error!
		fmt.Printf("connect to etcd failed, err:%v\n", err)
		return
	}
	fmt.Println("connect to etcd success")
	defer cli.Close()
	// put
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	//_, err = cli.Put(ctx, "name", "gaugest2009")
	_, err = cli.Put(ctx, gedis_key, gedis_val)
	cancel()
	if err != nil {
		fmt.Printf("put to getcd failed, err:%v\n", err)
		return
	}
	// get
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	resp, err := cli.Get(ctx, gedis_key)
	cancel()
	if err != nil {
		fmt.Printf("get from getcd failed, err:%v\n", err)
		return
	}
	for _, ev := range resp.Kvs {
		fmt.Printf("%s:%s\n", ev.Key, ev.Value)
	}
}

func (s GipkinService) Concat(a, b string) (string, error) {
	// test for length overflow
	if len(a)+len(b) > StrMaxSize {
		return "", ErrMaxSize
	}
	return a + b, nil
}

func (s GipkinService) Diff(a, b string) (string, error) {
	if len(a) < 1 || len(b) < 1 {
		return "", nil
	}
	res := ""
	if len(a) >= len(b) {
		for _, char := range b {
			if strings.Contains(a, string(char)) {
				res = res + string(char)
			}
		}
	} else {
		for _, char := range a {
			if strings.Contains(b, string(char)) {
				res = res + string(char)
			}
		}
	}
	return res, nil
}

// HealthCheck implement Service method
// 用于检查服务的健康状态，这里仅仅返回true。
func (s GipkinService) HealthCheck() bool {
	return true
}

// ServiceMiddleware define service middleware
type ServiceMiddleware func(Service) Service
