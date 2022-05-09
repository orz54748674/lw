package apiDg

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"strconv"
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
	"vn/storage/apiDgStorage"
	"vn/storage/apiStorage"
	"vn/storage/gameStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

var Module = func() module.Module {
	this := new(Dg)
	return this
}

type Dg struct {
	basemodule.BaseModule
	room *room.Room
}

func (self *Dg) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.ApiDg)
}

func (self *Dg) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

var (
	ActionRpcEnter      = "/apiDg/enter"
	ApiType        int8 = 9
	LoginUrl       map[string]string
	ApiUrl         string
	Lang           string
	AgentName      string
	ApiKey         string
)

func (s *Dg) InitConf(env string) {
	AgentName = "DGTE010525"
	ApiKey = "cda3c01b38174967b18c34de68888bb1"
	if env != "release" {
		LoginUrl["0"] = "https://luckymobile.1win888.net/"
		LoginUrl["1"] = "https://lucky.1win888.net/"
		ApiUrl = "https://api.dg2.co"
		Lang = "zh-CN"
	} else {
		LoginUrl["0"] = "https://luckymobile.fts368.com/"
		LoginUrl["1"] = "https://lucky.fts368.com/"
		ApiUrl = "https://api.dg2.co"
		Lang = "vi-VN"
	}
}

func (s *Dg) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings)
	LoginUrl = make(map[string]string)
	apiStorage.InitApiConfig(s)
	s.GetServer().RegisterGO(ActionRpcEnter, s.enter)

	go s.GetReportTiming()
}

func (self *Dg) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())
	<-closeSig
	log.Info("%v模块已停止...", self.GetType())
}

func (self *Dg) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", self.GetType())
}

func (s *Dg) onDisconnect(uid string) (interface{}, error) {
	log.Info("onDisconnect serverId: %s, uid: %s", s.GetServerID(), uid)
	return nil, nil
}

func (s *Dg) enter(data map[string]interface{}) (resp string, err error) {
	params := &struct {
		Token     string `json:"token"`
		GameType  string `json:"gameType"`
		Action    string `json:"action"`
		DriveType string `json:"driveType"`
	}{}
	if err = mapstructure.Decode(data, params); err != nil {
		log.Error("rpc Dg enter err:%s", err.Error())
		return
	}
	token := userStorage.QueryToken(params.Token)
	if token == nil {
		err = fmt.Errorf("user not login")
		return
	}

	uid := token.Oid.Hex()
	userInfo, err := apiDgStorage.GetDgUserInfoByUid(uid)
	fmt.Println("gdere111111111")

	if err != nil && err != mongo.ErrNoDocuments {
		return "", err
	}

	if err == mongo.ErrNoDocuments {

		fmt.Println("r3rgsdgds", err)
		user := userStorage.QueryUserId(utils.ConvertOID(uid))
		tmpToken, tmpRandom := GetToken()

		paramMap := make(map[string]interface{})
		paramMap["token"] = tmpToken
		paramMap["random"] = tmpRandom
		paramMap["data"] = "A"
		memberMap := make(map[string]interface{})
		memberMap["username"] = user.Account
		memberMap["password"] = GetPassword()
		memberMap["currencyName"] = "VND2"
		memberMap["winLimit"] = 0
		paramMap["member"] = memberMap
		fmt.Println("paragegfe", paramMap)
		if err = UserRegister(paramMap); err != nil {
			return "", err
		}

		userInfo.Uid = uid
		userInfo.Username = user.Account
		userInfo.Data = "A"
		userInfo.Password = memberMap["password"].(string)
		userInfo.WinLimit = 0
		if err = apiDgStorage.UpsertDgUserInfo(userInfo); err != nil {
			log.Error("insert dg user info err:%s", err.Error())
			return
		}
	}

	loginParam := make(map[string]interface{})
	loginParam["token"], loginParam["random"] = GetToken()
	loginParam["lang"] = "en"
	loginParam["domains"] = "1"
	loginMemberParam := make(map[string]interface{})
	loginMemberParam["username"] = userInfo.Username
	loginMemberParam["password"] = userInfo.Password
	loginParam["member"] = loginMemberParam

	fmt.Println("ggeege", loginParam)
	url, err := Login(loginParam, 0, "en")
	if err != nil {
		return "", err
	}

	return url, nil
}

func (s *Dg) GetReportTiming() {
	for {
		tmpToken, tmpRandom := GetToken()
		paramMap := make(map[string]interface{})
		paramMap["token"] = tmpToken
		paramMap["random"] = tmpRandom

		recordMap, err := GetReport(paramMap)
		if err != nil {
			continue
		}
		if recordMap["list"] != nil {
			list := recordMap["list"].([]interface{})
			var idArr []int64
			for _, v := range list {
				data := v.(map[string]interface{})
				iCount := int(data["isRevocation"].(float64))
				if iCount == 1 {
					idArr = append(idArr, int64(data["id"].(float64)))
					var recordParams gameStorage.BetRecordParam
					userName := data["userName"].(string)
					userInfo, err := apiDgStorage.GetDgUserInfoByUsername(userName)
					if err != nil {
						continue
					}
					wallet := walletStorage.QueryWallet(utils.ConvertOID(userInfo.Uid))
					recordParams.Uid = userInfo.Uid
					recordParams.GameNo = strconv.Itoa(int(data["id"].(float64)))
					recordParams.BetAmount = int64(data["availableBet"].(float64) * 1000)
					recordParams.BotProfit = 0
					recordParams.SysProfit = 0
					recordParams.BetDetails = data["betDetail"].(string)
					recordParams.GameResult = data["result"].(string)
					recordParams.GameType = game.ApiDg
					recordParams.Income = int64(data["winOrLoss"].(float64)*1000) - recordParams.BetAmount
					recordParams.CurBalance = wallet.VndBalance
					recordParams.IsSettled = false
					if _, ok := data["GameId"].(float64); ok {
						recordParams.GameId = strconv.Itoa(int(data["GameId"].(float64)))
					} else {
						recordParams.GameId = strconv.Itoa(int(data["gameId"].(float64)))
					}

					gameStorage.InsertBetRecord(recordParams)
				}
			}
			tmpToken1, tmpRandom1 := GetToken()
			paramMap1 := make(map[string]interface{})
			paramMap1["token"] = tmpToken1
			paramMap1["random"] = tmpRandom1
			paramMap1["list"] = idArr
			MarkReport(paramMap1)
		}
		time.Sleep(20 * time.Second)
	}
}

func (s *Dg) InitApiCfg(cfg *apiStorage.ApiConfig) {
	cfg.ApiType = ApiType
	cfg.ApiTypeName = string(game.ApiDg)
	cfg.Env = s.App.GetSettings().Settings["env"].(string)
	cfg.Module = s.GetType()
	cfg.GameType = apiStorage.Sports
	cfg.GameTypeName = "Sports"
	cfg.Topic = ActionRpcEnter[1:]

	s.InitConf(cfg.Env)
}
