package apiStorage

import (
	"time"
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/log"
)

type ApiCfg interface {
	InitApiCfg(cfg *ApiConfig)
}

var (
	cApiConfig = "ApiConfig"
)

func InitApiConfig(a ApiCfg) {
	InitApiUser()
	cfg := &ApiConfig{}
	a.InitApiCfg(cfg)
	if err := cfg.init(); err != nil {
		log.Error("%s init apiConfig err:%s", cfg.Module, err.Error())
	}
}

func AddApiConfig(cfgs ...*ApiConfig) {
	for _, cfg := range cfgs {
		if err := cfg.init(); err != nil {
			log.Error("%s init apiConfig err:%s", cfg.Module, err.Error())
		}
	}
}

/**
 *  @title	SetName
 *	@description	获取集合名
 *	@return	 setName	string	集合名
 */
func (m *ApiConfig) SetName() string {
	return cApiConfig
}

func (m *ApiConfig) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *ApiConfig) init() error {
	find := bson.M{"Env": m.Env, "Module": m.Module, "GameTypeName": m.GameTypeName}
	res := &ApiConfig{}
	err := m.C().Find(find).One(res)
	if err == mongo.ErrNoDocuments {
		m.CreateAt = time.Now()
		m.UpdateAt = time.Now()
		err = m.C().Insert(m)
	} else if err == nil {
		*m = *res
	}
	return err
}

func (m *ApiConfig) GetApis(gameTypeName, env string) (res []*ApiConfig, err error) {
	find := bson.M{"GameTypeName": gameTypeName, "Env": env, "Status": 1}
	res = []*ApiConfig{}
	err = m.C().Find(find).All(&res)
	return
}

func (m *ApiConfig) GetApi(gameTypeName, module, env string) (res *ApiConfig, err error) {
	find := bson.M{"GameTypeName": gameTypeName, "Env": env, "Status": 1, "Module": module}
	res = &ApiConfig{}
	err = m.C().Find(find).All(res)
	return
}
