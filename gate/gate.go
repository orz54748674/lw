package gate

import (
	"strconv"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/storage/userStorage"
)

var Module = func() module.Module {
	gate := new(Gate)
	return gate
}

type Gate struct {
	basegate.Gate //继承
}

func (this *Gate) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "Gate"
}
func (this *Gate) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

func (this *Gate) OnInit(app module.App, settings *conf.ModuleSettings) {
	//注意这里一定要用 gate.Gate 而不是 module.BaseModule
	wssPort := int(app.GetSettings().Settings["wssPort"].(float64))
	tcpPort := int(app.GetSettings().Settings["tcpPort"].(float64))
	sessionExpireSecond := time.Duration(app.GetSettings().Settings["sessionExpireSecond"].(float64))
	sessionStorage := &SessionStorage{}
	sessionStorage.InitMongo(sessionExpireSecond * time.Second)
	RemoveAllSession()
	this.Gate.OnInit(this, app, settings,
		gate.WsAddr(":"+strconv.Itoa(wssPort)),
		gate.TCPAddr(":"+strconv.Itoa(tcpPort)),
		gate.SetSessionLearner(this),
		gate.ConcurrentTasks(100),
		gate.Heartbeat(3*time.Second),
		gate.MaxPackSize(65536),
		gate.OverTime(5*time.Second),
		gate.SetStorageHandler(sessionStorage),
		//gate.TLS(true),
		//gate.CertFile("xxx.cert"),
		//gate.KeyFile("xxx.key"),
	)

}

//当连接建立  并且MQTT协议握手成功
func (this *Gate) Connect(a gate.Session) {
	log.Info("Connect: %v", &a)
}

//当连接关闭	或者客户端主动发送MQTT DisConnect命令
func (this *Gate) DisConnect(a gate.Session) {
	log.Info("DisConnect: %s", &a)
	tokenObj := userStorage.QueryTokenBySession(a.GetSessionID())
	if tokenObj == nil {
		return
	} else {
		sessionBean := QuerySessionBean(tokenObj.Oid.Hex())
		if sessionBean != nil {
			onlineSec := int64(utils.Now().Sub(sessionBean.CreateAt) / time.Second)
			userStorage.IncUserSumOnlineSec(tokenObj.Oid, onlineSec)
		}
		tokenObj.SessionId = ""
		userStorage.UpsertToken(tokenObj)
	}
	if uid := a.GetUserID(); uid != "" {
		callListener(this.App, common.EventDisconnect, uid)
	}
	_ = this.GetStorageHandler().Delete(a)
}
func callListener(app module.App, event string, uid string) {
	listeners := common.QueryListener(event)
	for _, listener := range *listeners {
		server, err := app.GetServerByID(listener.ServerId)
		if err != nil {
			log.Error("callListener error: %s", err)
			return
		}
		if err := server.CallNR(listener.ServerRegister, uid); err != nil {
			log.Error(err.Error())
		}
	}
}
