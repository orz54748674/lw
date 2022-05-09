package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/framework/mqant"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/framework/mqant/registry"
	"vn/framework/mqant/registry/consul"
	"vn/game/activity"
	"vn/game/admin"
	"vn/game/api/spadeGame"
	"vn/game/apiAwc"
	"vn/game/apiCmd"
	"vn/game/apiCq"
	"vn/game/apiDg"
	"vn/game/apiSaBa"
	"vn/game/apiSbo"
	"vn/game/apiWm"
	"vn/game/apiXg"
	"vn/game/bjl"
	"vn/game/card/cardCatte"
	"vn/game/card/cardCddN"
	"vn/game/card/cardLhd"
	"vn/game/card/cardPhom"
	"vn/game/card/cardQzsg"
	"vn/game/card/cardSss"
	"vn/game/chat"
	common2 "vn/game/common"
	"vn/game/data"
	"vn/game/dx"
	"vn/game/fish"
	"vn/game/lobby"
	"vn/game/lottery"
	"vn/game/lottery/reptile"
	"vn/game/lottery/settle"
	"vn/game/mini/guessBigSmall"
	pk "vn/game/mini/poker"
	roshambo "vn/game/mini/roshambo"
	"vn/game/pay"
	"vn/game/sd"
	"vn/game/slot/slotCs"
	"vn/game/slot/slotDance"
	"vn/game/slot/slotLs"
	"vn/game/slot/slotSex"
	"vn/game/suoha"
	"vn/game/yxx"
	"vn/gate"
	"vn/http"
	"vn/sms"
	"vn/storage"

	"github.com/nats-io/nats.go"
)

var consulHost = flag.String("ch", "172.17.0.2", "consul Server IP")
var consulPort = flag.String("chp", "8500", "consul Server port")
var natHost = flag.String("nh", "192.168.72.128", "nats Server IP")
var natPort = flag.String("nhp", "4222", "nats Server port")
var wdPath = flag.String("wd", "./", "Server work directory")
var confPath = flag.String("conf", "./bin/conf/server.json", "Server configuration file path")
var ProcessID = flag.String("pid", "development", "Server ProcessID?")
var Logdir = flag.String("log", "./bin/logs", "Log file directory?")
var BIdir = flag.String("bi", "./bin", "bi file directory?")

func main() {
	flag.Parse()
	log.Debug("consulHost:%s ,natHost:%s", *consulHost, *natHost)
	rs := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{fmt.Sprintf("%s:%s", *consulHost, *consulPort)}
	})
	natUrl := fmt.Sprintf("nats://%s:%s", *natHost, *natPort)
	nc, err := nats.Connect(natUrl, nats.MaxReconnects(10000))
	if err != nil {
		log.Error("nats error %v", err)
		return
	}
	app := mqant.CreateApp(
		module.Parse(false),
		module.WorkDir(*wdPath),
		module.Configure(*confPath),
		module.ProcessID(*ProcessID),
		module.LogDir(*Logdir),
		module.BILogDir(*BIdir),

		module.KillWaitTTL(1*time.Minute),
		module.Debug(true),  //只有是在调试模式下才会在控制台打印日志, 非调试模式下只在日志文件中输出日志
		module.Nats(nc),     //指定nats rpc
		module.Registry(rs), //指定服务发现
		module.RegisterTTL(20*time.Second),
		module.RegisterInterval(10*time.Second),
	)
	_ = app.OnConfigurationLoaded(func(app module.App) {
		mongoConf := &common.DBConf{
			Host:     app.GetSettings().Settings["MongodbHost"].(string),
			DbName:   app.GetSettings().Settings["MongodbDB"].(string),
			User:     app.GetSettings().Settings["MongodbUser"].(string),
			Password: app.GetSettings().Settings["MongodbPassword"].(string),
			MysqlDns: app.GetSettings().Settings["MysqlDns"].(string),
		}
		common.MongoConfig = mongoConf
		//common.InitMongo(mongoConf)
		common.InitMongoDB(mongoConf)
		common.InitMysql(mongoConf)
		common.InitListener(app)
		common.CurLanguage = storage.QueryConf(storage.KCurLanguage).(string)
		common2.InitBots()
	})
	_ = app.SetProtocolMarshal(func(Trace string, Result interface{}, Error string) (module.ProtocolMarshal, string) {
		//log.Error("result: %v",Result)
		b, err := json.Marshal(Result)
		if err == nil {
			//解析得到[]byte后用NewProtocolMarshal封装为module.ProtocolMarshal
			return app.NewProtocolMarshal(b), ""
		} else {
			log.Error(err.Error())
			e, _ := errCode.ActionNotFound.Json()
			return app.NewProtocolMarshal(e), ""
		}
	})
	common.App = app
	err = app.Run( //模块都需要加到入口列表中传入框架
		//hello.Module(),
		data.Module(),
		admin.Module(),
		pay.Module(),
		chat.Module(),
		dx.Module(),
		lobby.Module(),
		yxx.Module(),
		sd.Module(),
		gate.Module(),
		http.HttpModule(),
		slotLs.Module(),
		slotCs.Module(),
		slotSex.Module(),
		slotDance.Module(),
		cardSss.Module(),
		//cardCddS.Module(),
		cardCddN.Module(),
		cardPhom.Module(),
		cardCatte.Module(),
		cardLhd.Module(),
		cardQzsg.Module(),
		reptile.Module(),
		settle.Module(),
		lottery.Module(),
		bjl.Module(),
		fish.Module(),
		pk.Module(),
		roshambo.Module(),
		guessBigSmall.Module(),
		apiXg.Module(),
		apiCq.Module(),
		apiCmd.Module(),
		apiDg.Module(),
		apiAwc.Module(),
		apiWm.Module(),
		apiSbo.Module(),
		apiSaBa.Module(),
		activity.Module(),
		suoha.Module(),
		sms.Module(),
		spadeGame.Module(),
	)

	if err != nil {
		log.Error(err.Error())
	}
}
