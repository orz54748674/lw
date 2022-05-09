package utils

import (
	"context"
	"time"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	mqrpc "vn/framework/mqant/rpc"
	"vn/framework/mqant/server"
)

type Module interface {
	GetType() string
	GetServer() server.Server
	GetApp() module.App
}

type rpcHeadler func(map[string]interface{}) (map[string]interface{}, error)

func RegisterRpcToHttp(m Module, routerPath, topic string, rpcFun rpcHeadler, registerLock chan bool) {
	data := map[string]interface{}{
		"routerPath": routerPath,
		"topic":      topic,
		"moduleType": m.GetType(),
	}
	tk := time.NewTicker(3 * time.Second)
	defer tk.Stop()
	count := 0
	for {
		if count > 10 {
			break
		}
		<-tk.C
		ctx, _ := context.WithTimeout(context.TODO(), time.Second*3)
		code, err := mqrpc.Int64(
			m.GetApp().Call(
				ctx,
				"http_gate",
				"/http_gate/rpcRegister",
				mqrpc.Param(data),
			),
		)
		count++
		if err != nil {
			log.Error("Sbo registerRpcToHttp code:%d err:%s", code, err.Error())
			continue
		}
		if code == 0 {
			log.Debug("Sbo registerRpcToHttp success code:%d", code)
			registerLock <- true
			m.GetServer().RegisterGO(topic, rpcFun)
			<-registerLock
			break
		}
	}
}
