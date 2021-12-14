package transport

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gauge2009/micro-golang/ch6-discovery/gipkin-service/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/transport"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	ErrorBadRequest = errors.New("invalid request parameter")
)

// MakeHttpHandler make http handler use mux
func MakeHttpHandler(ctx context.Context, endpoints endpoint.StringEndpoints, logger log.Logger) http.Handler {
	r := mux.NewRouter()

	options := []kithttp.ServerOption{
		kithttp.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
		kithttp.ServerErrorEncoder(encodeError),
	}
	///http://127.0.0.1:10085/op/Diff/break/Bread
	//r.Methods("POST").Path("/op/{type}/{a}/{b}").Handler(kithttp.NewServer(
	//	endpoints.StringEndpoint,
	//	decodeStringRequest,
	//	encodeStringResponse,
	//	options...,
	//))
	//"KeyID":"gauge212112141242",
	//	"SpanID":"RabbitMQ_To_Worker",
	//	"TraceID":"12345678-415d-40e1-987a-17a24f83f47c",
	//	"BizCode":"ATSINSPECT-ATSINSPECT",
	//	"ParentID":"AtsTaskService",
	//	"Level":"Debug",
	//	"ClassName":"HRLink.BackendService.AtsInspectService",
	//	"MethodName":"AtsInspectExecute",
	//	"LocationDesc":"考勤审查持久化完成"
	r.Methods("POST").Path("/op/{type}/{KeyID}/{SpanID}/{TraceID}/{BizCode}/{ParentID}/{Level}/{ClassName}/{MethodName}/{LocationDesc}").Handler(kithttp.NewServer(
		endpoints.StringEndpoint,
		decodeStringRequest,
		encodeStringResponse,
		options...,
	))
	r.Path("/metrics").Handler(promhttp.Handler())

	// create health check handler
	r.Methods("GET").Path("/health").Handler(kithttp.NewServer(
		endpoints.HealthCheckEndpoint,
		decodeHealthCheckRequest,
		encodeStringResponse,
		options...,
	))

	return r
}

// decodeStringRequest decode request params to struct
func decodeStringRequest(_ context.Context, r *http.Request) (interface{}, error) {

	vars := mux.Vars(r)
	requestType, ok := vars["type"]
	if !ok {
		return nil, ErrorBadRequest
	}

	pKeyID, ok := vars["KeyID"]
	if !ok {
		return nil, ErrorBadRequest
	}

	pSpanID, ok := vars["SpanID"]
	if !ok {
		return nil, ErrorBadRequest
	}
	pTraceID, ok := vars["TraceID"]
	if !ok {
		return nil, ErrorBadRequest
	}
	pBizCode, ok := vars["BizCode"]
	if !ok {
		return nil, ErrorBadRequest
	}
	pParentID, ok := vars["ParentID"]
	if !ok {
		return nil, ErrorBadRequest
	}
	pLevel, ok := vars["Level"]
	if !ok {
		return nil, ErrorBadRequest
	}
	pClassName, ok := vars["ClassName"]
	if !ok {
		return nil, ErrorBadRequest
	}
	pMethodName, ok := vars["MethodName"]
	if !ok {
		return nil, ErrorBadRequest
	}
	pLocationDesc, ok := vars["LocationDesc"]
	if !ok {
		return nil, ErrorBadRequest
	}
	return endpoint.StringRequest{
		RequestType:  requestType,
		KeyID:        pKeyID,
		SpanID:       pSpanID,
		TraceID:      pTraceID,
		BizCode:      pBizCode,
		ParentID:     pParentID,
		Level:        pLevel,
		ClassName:    pClassName,
		MethodName:   pMethodName,
		LocationDesc: pLocationDesc,
	}, nil
}

// encodeStringResponse encode response to return
func encodeStringResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json;charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

// decodeHealthCheckRequest decode request
func decodeHealthCheckRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	return endpoint.HealthRequest{}, nil
}

func encodeError(_ context.Context, err error, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	switch err {
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": err.Error(),
	})
}
