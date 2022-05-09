package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
	"vn/common"
	"vn/framework/mqant"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
)

func main() {
	for idx, args := range os.Args {
		fmt.Println("参数"+strconv.Itoa(idx)+":", args)
	}

	//if len(os.Args) >= 2 {
	//	path := os.Args[1]
	//	Init(path)
	//	event := os.Args[2]
	//	switch event {
	//	case "order":
	//		//orderId := os.Args[3]
	//		//successVgPayOrder(orderId)
	//	}
	//}
	//if len(os.Args) >= 3 {
	//	path := os.Args[2]
	//	Init(path)
	//	fixActivity()
	//}

	if len(os.Args) >= 3 {
		path := os.Args[2]
		Init(path)
		fixGiftCode()
	}
	return
}

func Init(path string) {
	app := mqant.CreateApp(
		module.KillWaitTTL(1 * time.Minute),
	)
	f, err := os.Open(path)
	if err != nil {
		//文件不存在
		panic(fmt.Sprintf("config path error %v", err))
	}
	var cof conf.Config
	fmt.Println("Server configuration path :", path)
	conf.LoadConfig(f.Name()) //加载配置文件
	cof = conf.Conf
	log.Info("conf: %v", cof)
	app.Configure(cof)
	mongoConf := &common.DBConf{
		Host:     app.GetSettings().Settings["MongodbHost"].(string),
		DbName:   app.GetSettings().Settings["MongodbDB"].(string),
		User:     app.GetSettings().Settings["MongodbUser"].(string),
		Password: app.GetSettings().Settings["MongodbPassword"].(string),
		MysqlDns: app.GetSettings().Settings["MysqlDns"].(string),
	}
	common.MongoConfig = mongoConf
	common.InitMongoDB(mongoConf)
	//common.InitMongoDB(mongoConf)
	common.InitMysql(mongoConf)
}
