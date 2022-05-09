package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"time"
	"vn/framework/mqant/conf"
	go_api "vn/framework/mqant/httpgateway/proto"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game/apiAwc"
	"vn/game/apiCmd"
	"vn/game/apiCq"
	"vn/game/apiDg"
	"vn/game/apiWm"
	"vn/game/apiXg"
	"vn/storage/userStorage"
	"vn/storage/versionStorage"

	"github.com/mitchellh/mapstructure"
)

var HttpModule = func() module.Module {
	this := new(httpgate)
	return this
}

type httpgate struct {
	basemodule.BaseModule
	httpPort          int
	route             *route
	userController    *UserController
	versionController *VersionController
	dataController    *DataController
	chargeNotify      *chargeNotify
	backendController *BackendController
	xgHttp            *apiXg.XgHttp
	cqHttp            *apiCq.CqHttp
	cmdHttp           *apiCmd.CmdHttp
	awcHttp           *apiAwc.AwcHttp
	wmHttp            *apiWm.WmHttp
	dgHttp            *apiDg.DgHttp
}

func (self *httpgate) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "http_gate"
}
func (self *httpgate) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

func (self *httpgate) OnInit(app module.App, settings *conf.ModuleSettings) {
	self.BaseModule.OnInit(self, app, settings)
	self.httpPort = int(app.GetSettings().Settings["httpPort"].(float64))
	self.route = &route{rpcMap: make(map[string]*rpcInfo), rpcMapLock: new(sync.RWMutex)}
	//httpRouter := &HttpRouter{app: app,settings: settings}
	//self.SetListener(httpRouter)
	self.userController = &UserController{
		BaseController{App: app, Settings: settings}}
	self.versionController = &VersionController{
		BaseController{App: app, Settings: settings}}
	self.backendController = &BackendController{
		BaseController{App: app, Settings: settings}}
	self.dataController = &DataController{
		BaseController{App: app, Settings: settings}}
	self.xgHttp = &apiXg.XgHttp{}
	env := app.GetSettings().Settings["env"].(string)
	apiXg.InitHost(env)
	self.chargeNotify = &chargeNotify{}
	self.chargeNotify.init()
	//self.GetServer().RegisterGO("/http_gate/user/register",self.httpgateway)
	//self.GetServer().RegisterGO("/http_gate/user/login",self.httpgateway)
	self.initMongo()
	self.GetServer().RegisterGO("/http_gate/rpcRegister", self.rpcRegister)
}

func (s *httpgate) rpcRegister(data map[string]interface{}) (code int64, err error) {
	params := &struct {
		RouterPath string `json:"routerPath"`
		Topic      string `json:"topic"`
		ModuleType string `json:"moduleType"`
	}{}
	if err = mapstructure.Decode(data, params); err != nil {
		code = 4
		log.Error("httpgate rpcRegister err:%s", err.Error())
		return
	}
	for _, pathInfo := range s.route.routePath {
		if pathInfo.pattern == params.RouterPath {
			code = 3
			err = fmt.Errorf("httpgate rpcRegister RouterPath:%s existed", params.RouterPath)
			return
		}
	}
	if _, ok := s.route.getRpcInfo(params.RouterPath); ok {
		code = 2
		err = fmt.Errorf("httpgate rpcRegister RouterPath:%s existed", params.RouterPath)
	} else {
		s.route.setRpcInfo(params.RouterPath, &rpcInfo{
			Topic:      params.Topic,
			ModuleType: params.ModuleType,
		})
	}
	return
}

func (s *httpgate) setupRouter(mux *http.ServeMux) {
	//mux.HandleFunc("/http_gate/user/login", s.userController.login)
	//mux.HandleFunc("/http_gate/user/register", s.userController.register)

}

func (s *httpgate) startHttpServer() *http.Server {
	//r := mux.NewRouter()
	srv := &http.Server{
		Addr: ":" + strconv.Itoa(s.httpPort),
		//Handler:httpgateway.NewHandler(s.App),
		//Handler: r,
	}

	//s.handleFunc(r,"/user/login",s.userController.login)
	//s.handleFunc(r,"/user/register",s.userController.register)
	//s.handleFunc(r,"/sms/send",s.userController.smsSend)
	//s.handleFunc(r,"/version/get",s.versionController.Get)
	//s.handleFunc(r,"/charge/",s.chargeNotify.Dispatch)
	routeInfo := []routeInfo{
		{pattern: "/test", f: s.versionController.test},
		{pattern: "/user/token_bind", f: s.userController.tokenBind},
		{pattern: "/user/login", f: s.userController.login},
		{pattern: "/user/register", f: s.userController.register},
		{pattern: "/sms/send", f: s.userController.smsSend},
		{pattern: "/data/start", f: s.dataController.start},
		{pattern: "/version/get", f: s.versionController.Get},
		{pattern: "/version/jackpot", f: s.versionController.jackpot},
		{pattern: "/version/conf", f: s.versionController.conf},
		{pattern: "/version/download", f: s.versionController.ToDownload},
		{pattern: "/version/get_app_url", f: s.versionController.GetAppDownloadUrl},
		{pattern: "/charge/*", f: s.chargeNotify.Dispatch},

		// XG接口
		{pattern: "/user/balance", f: s.xgHttp.GetUserBalance},
		{pattern: "/transaction/bet", f: s.xgHttp.AddBet},
		{pattern: "/transaction/settle", f: s.xgHttp.Settle},
		{pattern: "/transaction/rollback", f: s.xgHttp.Rollback},

		//cq接口
		{pattern: "/transaction/record/*", f: s.cqHttp.Record},
		{pattern: "/transaction/balance/*", f: s.cqHttp.Balance},
		{pattern: "/player/check/*", f: s.cqHttp.CheckPlayer},
		{pattern: "/transaction/game/bets", f: s.cqHttp.BatchBets},
		{pattern: "/transaction/game/refunds", f: s.cqHttp.Refunds},
		{pattern: "/transaction/game/cancel", f: s.cqHttp.Cancel},
		{pattern: "/transaction/game/amend", f: s.cqHttp.Amend},
		{pattern: "/transaction/game/wins", f: s.cqHttp.Wins},
		{pattern: "/transaction/game/amends", f: s.cqHttp.Amends},

		//cmd接口
		{pattern: "/cmd/verifyToken", f: s.cmdHttp.VerifyToken},
		{pattern: "/cmd/getBalance", f: s.cmdHttp.GetBalance},
		{pattern: "/cmd/deductBalance", f: s.cmdHttp.DeductBalance},
		{pattern: "/cmd/updateBalance", f: s.cmdHttp.UpdateBalance},

		//dg接口
		{pattern: "/user/getBalance/DGTE010525", f: s.dgHttp.GetBalance},
		{pattern: "/account/transfer/DGTE010525", f: s.dgHttp.Transfer},
		{pattern: "/account/checkTransfer/DGTE010525", f: s.dgHttp.CheckTransfer},
		{pattern: "/account/inform/DGTE010525", f: s.dgHttp.Inform},
		{pattern: "/account/order/DGTE010525", f: s.dgHttp.Order},
		{pattern: "/account/unsettle/DGTE010525", f: s.dgHttp.Unsettle},

		// Awc接口
		{pattern: "/awc", f: s.awcHttp.Entrance},
		// Wm 接口
		{pattern: "/wm", f: s.wmHttp.Entrance},

		{pattern: "/apiLogin", f: s.userController.apiLogin},
		{pattern: "/apiLoginV1", f: s.userController.apiLoginV1},
	}
	s.route.init(routeInfo, s.App)
	//r.Use(mux.CORSMethodMiddleware(r))
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			log.Info("Httpserver: ListenAndServe() error: %s", err)
		}
	}()
	// returning reference so caller can call Shutdown()
	return srv
}

func (self *httpgate) Run(closeSig chan bool) {
	log.Info("httpgate: starting HTTP server :" + strconv.Itoa(self.httpPort))
	srv := self.startHttpServer()
	<-closeSig
	log.Info("httpgate: stopping HTTP server")
	// now close the server gracefully ("shutdown")
	// timeout could be given instead of nil as a https://golang.org/pkg/context/
	if err := srv.Shutdown(context.Background()); err != nil {
		panic(err) // failure/timeout shutting down the server gracefully
	}
	log.Info("httpgate: done. exiting")
}

func (self *httpgate) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
}

func (self *httpgate) httpgateway(request *go_api.Request) (*go_api.Response, error) {
	mux := http.NewServeMux()
	self.setupRouter(mux)

	req, err := http.NewRequest(request.Method, request.Url, strings.NewReader(request.Body))
	if err != nil {
		return nil, err
	}
	for _, v := range request.Header {
		req.Header.Set(v.Key, strings.Join(v.Values, ","))
	}
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	header := make(map[string]*go_api.Pair)
	header["Access-Control-Allow-Origin"] = &go_api.Pair{Key: "Access-Control-Allow-Origin", Values: []string{"*"}}
	header["Access-Control-Allow-Methods"] = &go_api.Pair{Key: "Access-Control-Allow-Methods", Values: []string{"*"}}
	header["Access-Control-Allow-Headers"] = &go_api.Pair{Key: "Access-Control-Allow-Headers", Values: []string{"*"}}
	resp := &go_api.Response{
		StatusCode: int32(rr.Code),
		Body:       rr.Body.String(),
		Header:     header,
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
	return resp, nil
}
func (self *httpgate) initMongo() {
	loginTokenExpire := time.Duration(self.App.GetSettings().Settings["loginTokenExpire"].(float64))
	userStorage.InitUserMongo(loginTokenExpire * 24 * time.Hour)
	versionStorage.InitVersion()
}

//func (s *httpgate)mailSend(w http.ResponseWriter, r *http.Request){
//	_ = r.ParseForm()
//	p := r.Form
//	if _,ok := utils.CheckParams(p, []string{"title","contentTitle","content"});ok != nil{
//		return
//	}
//	mailType := gameStorage.Group
//	account := ""
//	if p["account"] != nil{
//		mailType = gameStorage.Private
//		account = p["account"][0]
//	}
//	mail := make(map[string]interface{})
//	mail["Type"] = mailType
//	mail["Title"] = p["title"][0]
//	mail["ContentTitle"] = p["contentTitle"][0]
//	mail["Content"] = p["content"][0]
//	mail["Account"] = account
//	ctx, _ := context.WithTimeout(context.TODO(), time.Second*3)
//	protoBean := new(rpcpb.ResultInfo)
//	mqrpc.Proto(protoBean, func() (reply interface{}, errstr interface{}) {
//		return s.Call(
//			ctx,
//			"lobby",     //要访问的moduleType
//			"HD_MailSend", //访问模块中handler路径
//			mqrpc.Param(mail),
//		)
//	})
//}
