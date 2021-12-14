package endpoint

import (
	"context"
	"errors"
	//"github.com/gauge2009/micro-golang/ch10-resiliency/gipkin-service/service"
	"github.com/gauge2009/micro-golang/ch6-discovery/gipkin-service/service"
	"github.com/go-kit/kit/endpoint"
	"strings"
)

// StringEndpoint define endpoint
type StringEndpoints struct {
	StringEndpoint      endpoint.Endpoint
	HealthCheckEndpoint endpoint.Endpoint
}

var (
	ErrInvalidRequestType = errors.New("RequestType has only two type: Concat, Diff")
)

// StringRequest define request struct
type StringRequest struct {
	RequestType  string `json:"request_type"`
	KeyID        string `json:"KeyID"`
	SpanID       string `json:"SpanID"`
	TraceID      string `json:"TraceID"`
	BizCode      string `json:"BizCode"`
	ParentID     string `json:"ParentID"`
	Level        string `json:"Level"`
	ClassName    string `json:"ClassName"`
	MethodName   string `json:"MethodName"`
	LocationDesc string `json:"LocationDesc"`
}

// StringResponse define response struct
type StringResponse struct {
	Result string `json:"result"`
	Error  error  `json:"error"`
}

// MakeStringEndpoint make endpoint
//	DoTrace(KeyID string, SpanID string, TraceID string, BizCode string, ParentID string, Level string, ClassName string, MethodName string, LocationDesc string) (string, error)
func MakeStringEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(StringRequest)

		var (
			res, KeyID, SpanID, TraceID, BizCode, ParentID, Level, ClassName, MethodName, LocationDesc string
			opError                                                                                    error
		)

		KeyID = req.KeyID
		SpanID = req.SpanID
		TraceID = req.TraceID
		BizCode = req.BizCode
		ParentID = req.ParentID
		Level = req.Level
		ClassName = req.ClassName
		MethodName = req.MethodName
		LocationDesc = req.LocationDesc

		// 根据请求操作类型请求具体的操作方法
		//if strings.EqualFold(req.RequestType, "Concat") {
		//	res, _ = svc.Concat(a, b)
		//} else if strings.EqualFold(req.RequestType, "Diff") {
		//	res, _ = svc.Diff(a, b)
		//} else
		if strings.EqualFold(req.RequestType, "DoTrace") {
			res, _ = svc.DoTrace(KeyID, SpanID, TraceID, BizCode, ParentID, Level, ClassName, MethodName, LocationDesc)
		} else {
			return nil, ErrInvalidRequestType
		}

		return StringResponse{Result: res, Error: opError}, nil
	}
}

// HealthRequest 健康检查请求结构
type HealthRequest struct{}

// HealthResponse 健康检查响应结构
type HealthResponse struct {
	Status bool `json:"status"`
}

// MakeHealthCheckEndpoint 创建健康检查Endpoint
func MakeHealthCheckEndpoint(svc service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		status := svc.HealthCheck()
		return HealthResponse{status}, nil
	}
}
