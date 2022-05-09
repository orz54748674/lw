package http

import (
	"vn/framework/mqant/conf"
	go_api "vn/framework/mqant/httpgateway/proto"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	mqrpc "vn/framework/mqant/rpc"
	rpcpb "vn/framework/mqant/rpc/pb"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
)

type HttpRouter struct {
	app module.App
	settings *conf.ModuleSettings
}

func (self *HttpRouter) show404(request *go_api.Request) (*go_api.Response,error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/http_gate/topic", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`show404`))
	})

	req, err := http.NewRequest(request.Method, request.Url, strings.NewReader(request.Body))
	if err != nil {
		return nil,err
	}
	for _,v:=range request.Header{
		req.Header.Set(v.Key, strings.Join(v.Values,","))
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	resp := &go_api.Response{
		StatusCode:  int32(rr.Code),
		Body: rr.Body.String(),
		Header: make(map[string]*go_api.Pair),
	}
	for key, vals := range rr.Header() {
		header, ok := resp.Header[key]
		if !ok {
			header = &go_api.Pair{
				Key: key,
			}
			resp.Header[key] = header
		}
		header.Values = vals
	}
	return    resp,nil
}
/**
NoFoundFunction 当未找到请求的handler时会触发该方法
*FunctionInfo  选择可执行的handler
return error
*/
func (s *HttpRouter)NoFoundFunction(fn string) (*mqrpc.FunctionInfo, error){
	log.Info("fn:" + fn)
	return &mqrpc.FunctionInfo{
		Function:reflect.ValueOf(s.show404),
		Goroutine:true,
	},nil
}
/**
BeforeHandle会对请求做一些前置处理，如：检查当前玩家是否已登录，打印统计日志等。
@session  可能为nil
return error  当error不为nil时将直接返回改错误信息而不会再执行后续调用
*/
func (s *HttpRouter)BeforeHandle(fn string, callInfo *mqrpc.CallInfo) error{
	return nil
}

func (s *HttpRouter)OnTimeOut(fn string, Expired int64) {}

func (s *HttpRouter)OnError(fn string, callInfo *mqrpc.CallInfo, err error) {}
/**
fn 		方法名
params		参数
result		执行结果
exec_time 	方法执行时间 单位为 Nano 纳秒  1000000纳秒等于1毫秒
*/
func (s *HttpRouter)OnComplete(fn string, callInfo *mqrpc.CallInfo, result *rpcpb.ResultInfo, execTime int64){}
