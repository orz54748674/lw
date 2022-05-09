package http

import (
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"net/http"
	"vn/common"
)

type BaseController struct {
	App module.App
	Settings *conf.ModuleSettings
}


func (s *BaseController)response(writer http.ResponseWriter,response *common.Err)  {
	byte,err := response.Json()
	if err != nil {
		log.Error("json format is err in your response")
	}
	if _,err := writer.Write(byte);err != nil{
		log.Error(err.Error())
	}
}
