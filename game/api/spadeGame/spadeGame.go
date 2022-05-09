package spadeGame

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
	"vn/common/utils"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/game"
	"vn/storage/apiStorage"
	"vn/storage/userStorage"

	"github.com/goinggo/mapstructure"
)

var (
	actionRpcEnter = "/apiSpadeGame/enter"
	spadeGameEnter = "/spadegaming/"
)

type SpadeGame struct {
	basemodule.BaseModule
	rpcServer *SpadeRpc
}

var Module = func() module.Module {
	this := new(SpadeGame)
	return this
}

func (m *SpadeGame) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return "apiSpadeGame"
}

func (m *SpadeGame) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}

func (m *SpadeGame) OnInit(app module.App, settings *conf.ModuleSettings) {
	m.BaseModule.OnInit(m, app, settings)

	apiStorage.InitApiConfig(m)
	m.GetServer().RegisterGO(actionRpcEnter, m.enter)

	registerLock := make(chan bool, 1)
	go utils.RegisterRpcToHttp(m, spadeGameEnter, spadeGameEnter, m.rpcServer.Enter, registerLock)

	go m.loadGameList()
}

func (m *SpadeGame) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", m.GetType())
	<-closeSig
	log.Info("%v模块已停止...", m.GetType())
}

func (m *SpadeGame) OnDestroy() {
	//一定别忘了继承
	m.BaseModule.OnDestroy()
	log.Info("%v模块已回收...", m.GetType())
}

func (m *SpadeGame) enter(data map[string]interface{}) (resp string, err error) {
	log.Debug("SpadeGame enter time:%v", time.Now().Unix())

	req := &struct {
		Token  string     `json:"token"`
		Uid    string     `json:"uid"`
		Params url.Values `json:"params"`
	}{}
	if err = mapstructure.Decode(data, req); err != nil {
		log.Error("rpc awc enter err:%s", err.Error())
		return
	}
	log.Debug("SpadeGame enter req:%v", req)
	user := userStorage.QueryUserId(utils.ConvertOID(req.Uid))
	btToken, err := utils.AesEncrypt([]byte(req.Uid), []byte(tokenSecret), []byte(iv))
	if err != nil {
		log.Error("rpc SpadeGame enter AesEncrypt err:%s", err.Error())
		return
	}
	req.Params.Set("token", string(btToken))
	req.Params.Set("acctId", fmt.Sprintf("%s%s", accountPrefix, user.Account))
	req.Params.Set("language", language)

	return fmt.Sprintf("%s%s", gameUrl, req.Params.Encode()), nil
}

func (m *SpadeGame) InitApiCfg(cfg *apiStorage.ApiConfig) {
	cfg.ApiType = apiStorage.SpadeGameType
	cfg.ApiTypeName = string(game.ApiSpadeGame)
	cfg.Env = m.App.GetSettings().Settings["env"].(string)
	cfg.Module = m.GetType()
	cfg.GameType = apiStorage.Ae
	cfg.GameTypeName = apiStorage.AeName
	cfg.Topic = actionRpcEnter[1:]
	cfg.ProductType = apiStorage.AeName

	apiStorage.AddApiConfig(cfg)
	InitEnv(cfg.Env)
}

func (m *SpadeGame) loadGameList() {
	tk := time.NewTicker(time.Hour * 1)
	for {
		gameList, err := getGameList()
		if err != nil {
			log.Error("SpadeGame loadGameList getGameList err:%s", err.Error())
			return
		}
		for _, item := range gameList {
			if len(item.Thumbnail) > 0 {
				ph, err := m.dowmloadImage(imgHost + item.Thumbnail)
				if err != nil {
					log.Error("SpadeGame loadGameList dowmloadImage Thumbnail err", err.Error())
				} else {
					item.Thumbnail = ph
				}
			}
			gm := &apiStorage.AeGame{
				Screenshot: item.Screenshot,
				Thumbnail:  item.Thumbnail,
				Mthumbnail: item.Mthumbnail,
				GameCode:   item.GameCode,
				GameName:   item.GameName,
				Jackpot:    item.Jackpot,
				ApiType:    apiStorage.SpadeGameType,
				Extends:    fmt.Sprintf(`{"jackpotCode":"%s","jackpotName":"%s"}`, item.JackpotCode, item.JackpotName),
			}
			gm.Add()
		}
		<-tk.C
	}

}

func (m *SpadeGame) dowmloadImage(url string) (ph string, err error) {
	log.Debug("SpadeGame dowmloadImage url:%v", url)
	paths := strings.Split(url, "/")
	log.Debug("SpadeGame dowmloadImage paths:%v", paths)
	pathLen := len(paths)
	if pathLen == 0 {
		return "", fmt.Errorf("SpadeGame dowmloadImage url:%s", url)
	}
	imgInfo := strings.Split(paths[pathLen-1], ".")
	imgInfoLen := len(imgInfo)
	if imgInfoLen != 2 {
		return "", fmt.Errorf("SpadeGame dowmloadImage img format:%s", url)
	}
	name := fmt.Sprintf("%s%s%d.%s", imagePath, imgInfo[0], time.Now().Unix(), imgInfo[1])

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	out, err := os.Create(name)
	if err != nil {
		return "", err
	}
	log.Debug("SpadeGame dowmloadImage img out:%v", out)
	_, err = io.Copy(out, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	return name, nil
}
