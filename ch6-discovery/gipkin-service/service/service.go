package service

import (
	"errors"
	"flag"
	"fmt"
	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"strings"
	"time"
	//"../gedis-benchmark/cacheClient"
	"github.com/gauge2009/micro-golang/gedis-benchmark/cacheClient"
)

// Service constants
const (
	StrMaxSize = 1024
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

func (s GipkinService) DoTrace(KeyID string, SpanID string, TraceID string, BizCode string, ParentID string, Level string, ClassName string, MethodName string, LocationDesc string) (string, error) {
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
	result := db.Create(&Task_link_trace{Key_id: KeyID, Span_id: SpanID,
		Trace_id: TraceID, Biz_code: BizCode, Parent_id: ParentID, Operate_dt: t2, Operate_by: "sys",
		Level: Level, Node_id: "worker_0x",
		Thread_id:  1,
		Class_name: ClassName, Method_name: MethodName, Location_desc: LocationDesc,
	})
	//result :=  db.Create(&Task_link_trace{Key_id: "1111-22-33333", Span_id: SpanID })
	if result != nil {
		if result.Error != nil {
			fmt.Println("result.Error ! = %v+\n", result.Error)
		}
		fmt.Println("RowsAffected = %v+\n", result.RowsAffected)
	}

	// Read
	var task_link_trace Task_link_trace

	db.First(&task_link_trace, "key_id = ?", "1fe7c255-84ae-4224-acbd-c2b116430b9e")

	//Gedis
	gedis_client_set("gauge2009_demo_key1", t2.String())

	return "success", nil
}

func gedis_client_set(gedis_key, gedis_val string) {
	server := flag.String("h", "localhost", "cache server address")
	op := flag.String("c", "set", "command, could be get/set/del")
	key := flag.String("k", gedis_key, "key")
	value := flag.String("v", gedis_val, "value")
	flag.Parse()
	client := cacheClient.New("tcp", *server)
	cmd := &cacheClient.Cmd{*op, *key, *value, nil}
	//./client -c set -k info1 -v  '{  \"result\": \"success\",   \"error\": null }'
	//./client -c get -k info1
	client.Run(cmd)
	if cmd.Error != nil {
		fmt.Println("error:", cmd.Error)
	} else {
		fmt.Println(cmd.Value)
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
