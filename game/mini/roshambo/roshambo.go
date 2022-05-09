package roshambo

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"time"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/framework/mqant/server"
	"vn/game"
	"vn/game/activity"
	"vn/storage"
	"vn/storage/gameStorage"
	"vn/storage/rsbStorage"
	"vn/storage/walletStorage"
)

var Module = func() module.Module {
	this := new(roshambo)
	return this
}

type roshambo struct {
	basemodule.BaseModule
	room *room.Room

	playInfo   map[string]*playerGameInfo
	botBalance int64
	coinConf   []int64
}

type playerGameInfo struct {
	Bets        []int64
	BOpen       bool
	Score       int64
	CurTotalBet int64
	redRes      int
	blueRes     int
}

const (
	actionGetRecord  = "HD_getRecord"
	actionBetAndOpen = "HD_betAndOpen"
	actionGetOpenRes = "HD_getOpenRes"
)

var resStr = []string{"石头", "剪刀", "布"}
var posStr = []string{"红方", "蓝方", "平局"}

func (s *roshambo) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.Roshambo)
}
func (s *roshambo) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}
func (s *roshambo) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings,
		server.RegisterInterval(15*time.Second),
		server.RegisterTTL(30*time.Second),
	)
	s.room = room.NewRoom(s.App)

	hook := game.NewHook(s.GetType())
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetRecord, s.GetRecord)
	hook.RegisterAndCheckLogin(s.GetServer(), actionBetAndOpen, s.BetAndOpen)
	hook.RegisterAndCheckLogin(s.GetServer(), actionGetOpenRes, s.GetOpenRes)
}

func (s *roshambo) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", s.GetType())
	<-closeSig
}

func (s *roshambo) OnDestroy() {
	//一定别忘了继承
	s.BaseModule.OnDestroy()
}

func (s *roshambo) GetRecord(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	if uid == "" {
		return errCode.Illegal.GetI18nMap(), nil
	}
	param1, ok1 := msg["Offset"].(float64)
	offset := int(param1)
	if !ok1 {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	param2, ok2 := msg["Limit"].(float64)
	if !ok2 {
		return errCode.ErrParams.GetI18nMap(), nil

	}
	limit := int(param2)

	records := rsbStorage.GetRsbRecord(uid, offset, limit)

	return errCode.Success(records).GetI18nMap(), nil
}

func (s *roshambo) GetOpenRes(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	if uid == "" {
		return errCode.Illegal.GetI18nMap(), nil
	}

	openRes := rsbStorage.GetOpenRes(uid, 20)
	return errCode.Success(openRes).GetI18nMap(), nil
}

func (s *roshambo) BetAndOpen(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	param1, ok1 := msg["pos"].(float64)
	pos := int(param1)
	if !ok1 {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	param2, ok2 := msg["coin"].(float64)
	if !ok2 {
		return errCode.ErrParams.GetI18nMap(), nil
	}
	coin := int64(param2)

	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	gData := gameStorage.QueryGameCommonData(uid)
	if (wallet.VndBalance - gData.InRoomNeedVnd) < coin {
		return errCode.BalanceNotEnough.GetI18nMap(), nil
	}

	winPos := -1
	redRes := -1
	blueRes := -1
	var score, mingProfit, anProfit int64
	botBalance, mingPercent, anPercent := rsbStorage.GetRsbConf()
	for i := 0; i < 10000; i++ {
		mingProfit = 0
		redRes = rand.Intn(3)
		blueRes = rand.Intn(3)
		if redRes == blueRes && rand.Intn(100) < 50 {
			continue
		}

		if redRes == blueRes {
			winPos = 2
		} else if redRes == 0 {
			if blueRes == 1 {
				winPos = 0
			}
			if blueRes == 2 {
				winPos = 1
			}
		} else if redRes == 1 {
			if blueRes == 0 {
				winPos = 1
			}
			if blueRes == 2 {
				winPos = 0
			}
		} else if redRes == 2 {
			if blueRes == 0 {
				winPos = 0
			}
			if blueRes == 1 {
				winPos = 1
			}
		}

		if pos == 2 && winPos == 2 {
			mingProfit = coin * 3 * mingPercent / 1000
			anProfit = coin * 3 * anPercent / 1000
			score = coin * 3 - mingProfit
		} else if winPos == 2 {
			score = 0
		} else if winPos == pos {
			mingProfit = coin * mingPercent / 1000
			anProfit = coin * anPercent / 1000
			score = coin - mingProfit
		} else {
			anProfit = coin * anPercent / 1000
			score = -coin
		}

		if botBalance - score > 0 {
			break
		}
	}

	var record rsbStorage.RsbRecord
	record.Uid = uid
	record.Score = score
	record.WinPos = winPos
	record.BetCount = coin
	record.Res = []int{redRes, blueRes}
	record.UpdateTime = time.Now().Unix()
	record.GameNo = strconv.Itoa(int(storage.NewGlobalId("roshamboNo") + 100000))

	rsbStorage.InsertRecord(record)

	billType := walletStorage.TypeIncome
	if score < 0 {
		billType = walletStorage.TypeExpenses
	}
	bill := walletStorage.NewBill(uid, billType, walletStorage.EventGameRoshambo, record.GameNo, score)
	walletStorage.OperateVndBalance(bill)
	wallet = walletStorage.QueryWallet(utils.ConvertOID(uid))

	gameStorage.IncProfit(uid, game.Roshambo, mingProfit, -(score+anProfit), anProfit)

	go func() {
		time.Sleep(5 * time.Second)
		walletStorage.NotifyUserWallet(uid)
	}()

	betInfo := struct {
		Pos string
		Coin int64
	}{
		Pos: posStr[pos],
		Coin: coin,
	}
	betDetail, _ := json.Marshal(betInfo)
	betDetailStr := string(betDetail)

	gameRes := struct {
		Red string
		Blue string
	}{
		Red: resStr[redRes],
		Blue: resStr[blueRes],
	}
	gameResByte, _ := json.Marshal(gameRes)
	gameResStr := string(gameResByte)
	var recordParams gameStorage.BetRecordParam
	recordParams.Uid = uid
	recordParams.GameNo = record.GameNo
	recordParams.BetAmount = coin
	recordParams.BotProfit = anProfit
	recordParams.SysProfit = mingProfit
	recordParams.BetDetails = betDetailStr
	recordParams.GameResult = gameResStr
	recordParams.CurBalance = wallet.VndBalance + wallet.SafeBalance
	recordParams.GameType = game.Roshambo
	recordParams.Income = score
	recordParams.IsSettled = true
	gameStorage.InsertBetRecord(recordParams)

	res := struct {
		RedRes  int   `json:"redRes"`
		BlueRes int   `json:"blueRes"`
		WinPos  int   `json:"winPos"`
		Score   int64 `json:"score"`
	}{
		RedRes:  redRes,
		BlueRes: blueRes,
		WinPos:  winPos,
		Score:   score,
	}

	activity.CalcEncouragementFunc(uid)

	return errCode.Success(res).GetI18nMap(), nil
}
