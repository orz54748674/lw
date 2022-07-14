package lobbyImpl

import (
	"encoding/json"
	"math/rand"
	"runtime"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	"vn/game"
	common2 "vn/game/common"
	"vn/gate"
	"vn/storage"
	"vn/storage/lobbyStorage"
)

type Broadcast struct {
	App        module.App
	Settings   *conf.ModuleSettings
	onlinePush *gate.OnlinePush
}

var ( //[游戏名称][用户名称][奖励类型][奖励金额]
	gameTypes = game.GameList
	//rewardTypes = []string{"赢得","奖池瓜分"}
	rewardTypes = []string{"win", "winJackpot"}
)

func (s *Broadcast) Init() {
	s.onlinePush = &gate.OnlinePush{
		App:       s.App,
		TraceSpan: log.CreateRootTrace(),
	}
	s.onlinePush.OnlinePushInit(nil, 64)
	s.onlinePush.Run(1 * time.Second)
}

func (s *Broadcast) Run() {
	defer func() {
		if r := recover(); r != nil {
			buff := make([]byte, 1024)
			runtime.Stack(buff, false)
			log.Error("lobby broadcast panic(%v)\n info:%s", r, string(buff))
		}
	}()
	for {
		time.Sleep(10 * time.Second)
		banners := getBanners()
		s.notify(banners)
		var ids []primitive.ObjectID
		for _, banner := range banners {
			ids = append(ids, banner.Oid)
		}
		lobbyStorage.RemoveLobbyBanner(ids)
		expire := storage.QueryConf(storage.KLobbyBannerUnUseExpire).(string)
		expireTime, _ := utils.ConvertInt(expire)
		lobbyStorage.RefreshLobbyBanner(expireTime)
	}
}
func getBanners() []lobbyStorage.LobbyBanner {
	banners := *lobbyStorage.QueryBanner(10)
	if len(banners) < 10 {
		difference := 10 - len(banners)
		for i := 0; i < difference; i++ {
			banners = append(banners, *randomOneLobbyBanner())
		}
	}
	return banners
}

func (s *Broadcast) notify(banners []lobbyStorage.LobbyBanner) {
	res := make(map[string]interface{}, 3)
	res["bannerArray"] = banners
	res["Action"] = "bannerArray"
	res["GameType"] = game.Lobby
	b, _ := json.Marshal(res)
	sessionBeans := gate.QuerySessionByPage("HallScene")
	if len(*sessionBeans) == 0 {
		return
	}
	var sessionIds []string
	for _, sessionBean := range *sessionBeans {
		sessionIds = append(sessionIds, sessionBean.SessionId)
	}
	_ = s.onlinePush.SendCallBackMsgNR(sessionIds, game.Push, b)
}

func randomOneLobbyBanner() *lobbyStorage.LobbyBanner {
Again:
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	gameRandom := utils.RandInt64(1, int64(len(gameTypes))+1, r)
	//x := float64(utils.RandInt64(1, 99,r))
	//y := (-4*math.Pow(10, -6))*math.Pow(x, 4) + 0.0005*math.Pow(x, 3) - 0.0204*math.Pow(x, 2) + 0.0644*x + 99.167
	//rewardRandom := 0
	//if int64(y) > 85 {
	//	rewardRandom = 1
	//}
	//dxConf := dxStorage.GetDxConf()
	//amount := dx.GetOneBetAmount(dxConf)
	lowestAmount, _ := utils.ConvertInt(storage.QueryConf(
		storage.KLobbyBannerLowestAmount).(string))
	tmp := lowestAmount / 100
	amount := utils.RandInt64(tmp, tmp*10, r) * 100
	bot := common2.RandBotN(1, r)
	gameType := string(gameTypes[gameRandom-1])
	isOpen := lobbyStorage.QueryLobbyGameLayoutByName(gameTypes[gameRandom-1]).Status
	if isOpen == 0 { //游戏未打开
		goto Again
	}
	banner := &lobbyStorage.LobbyBanner{
		Uid:      primitive.NewObjectID(),
		UserName: bot[0].NickName,
		GameType: common.I18str(gameType),
		WinType:  common.I18str(rewardTypes[0]),
		Amount:   amount,
	}
	return banner
}
