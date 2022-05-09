package apiAe

import (
	"strings"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
	"vn/storage/apiStorage"
	"vn/storage/userStorage"

	"github.com/mitchellh/mapstructure"
)

type Ae struct {
	basemodule.BaseModule
}

var (
	actionEnter      = "HD_enter"
	apiType     int8 = 2
)

var Module = func() module.Module {
	this := new(Ae)
	return this
}

func (self *Ae) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "apiAe"
}

func (self *Ae) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

func (self *Ae) OnInit(app module.App, settings *conf.ModuleSettings) {
	self.BaseModule.OnInit(self, app, settings)
	env := app.GetSettings().Settings["env"].(string)
	InitEnv(env)
	hook := game.NewHook(self.GetType())
	hook.RegisterAndCheckLogin(self.GetServer(), actionEnter, self.Enter)
}

func (self *Ae) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())
	<-closeSig
	log.Info("%v模块已停止...", self.GetType())
}

func (self *Ae) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", self.GetType())
}

func (self *Ae) Enter(session gate.Session, data map[string]interface{}) (map[string]interface{}, error) {
	params := &struct {
		Device   int8 `json:"Device"`
		PlayMode int8 `json:"PlayMode"`
	}{}
	if err := mapstructure.Decode(data, params); err != nil {
		return errCode.ErrParams.GetI18nMap(), err
	}
	uid := session.GetUserID()
	mApiUser := &apiStorage.ApiUser{}
	err := mApiUser.GetApiUser(uid, apiType)
	if err == mongo.ErrNoDocuments {
		user := userStorage.QueryUserId(utils.ConvertOID(uid))
		user.Account = strings.ToUpper(user.Account)
		if eCode, err := CreateUser(user.Account); err != nil {
			return errCode.ApiCreateUserErr.GetI18nMap(), err
		} else if eCode == UserExists || eCode == SuccessCode {
			mApiUser.Account = user.Account
			mApiUser.Type = apiType
			mApiUser.Uid = uid
			if err = mApiUser.Save(); err != nil {
				return errCode.ServerError.GetI18nMap(), err
			}
		} else {
			return errCode.ApiErr.GetI18nMap(), nil
		}
	} else if err != nil {
		return errCode.ServerError.GetI18nMap(), err
	}
	loginInfo, err := Login(mApiUser.Account, "AWS_1", params.PlayMode, params.Device)
	if err != nil {
		log.Error("ApiLoginErr err:%s", err.Error())
		return errCode.ApiLoginErr.GetI18nMap(), err
	}
	return errCode.Success(map[string]interface{}{"Data": loginInfo}).GetI18nMap(), nil
}
