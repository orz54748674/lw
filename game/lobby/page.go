package lobby

import (
	"strings"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/gate"
	"vn/storage/dataStorage"
	"vn/storage/userStorage"

	"github.com/robfig/cron/v3"
)

type DataPage struct {
}

var (
	page = map[string]game.Type{}
)
var login game.Type = "login"

func initPage() {
	page["Yxxgame"] = game.YuXiaXie

	page["SdHall"] = game.SeDie
	page["Sdgame"] = game.SeDie

	page["LsLodingGame"] = game.SlotLs
	page["LsGameScene"] = game.SlotLs
	page["LsFbStartScene"] = game.SlotLs
	page["LsFbGameScene"] = game.SlotLs

	page["SssHallScene"] = game.CardSss
	page["SssGameScene"] = game.CardSss

	page["CddWHallScene"] = game.CardCddN
	page["CddWGameScene"] = game.CardCddN

	page["CddSHallScene"] = game.CardCddS
	page["CddSGameScene"] = game.CardCddS

	page["CsdLodingScene"] = game.SlotCs
	page["CsdGameScene"] = game.SlotCs

	page["PhomHallScene"] = game.CardPhom
	page["PhomGameScene"] = game.CardPhom

	page["FishGameScene"] = game.Fish

	page["CpGameScene"] = game.Lottery

	page["AvLodingScene"] = game.SlotSex
	page["AvGameScene"] = game.SlotSex

	page["SixHallScene"] = game.CardCatte
	page["SixGameScene"] = game.CardCatte

	page["LhLoadingScene"] = game.CardLhd
	page["LhGameScene"] = game.CardLhd

	page["SgHallScene"] = game.CardQzsg
	page["SgGameScene"] = game.CardQzsg

	page["BaccaratGameScene"] = game.Bjl

	page["DanceLodingScene"] = game.SlotDance
	page["DanceGameScene"] = game.SlotDance

	page["LiveCasinoScene"] = game.ApiLive

	page["LiveSportsScene"] = game.ApiSport

	page["GuessBigSmallMainView"] = game.GuessBigSmall

	page["FingGuessMainView"] = game.Roshambo

	page["MiniPokerMainView"] = game.MiniPoker

	page["DxgameLayer"] = game.BiDaXiao

	page["HallScene"] = game.Lobby
	page["LoginScene"] = login

	clearOnline()
	syncUserOnline()
	timerSyncUserOnline()
	_ = common.GetMysql().AutoMigrate(&dataStorage.DataGameStart{})
	_ = common.GetMysql().AutoMigrate(&dataStorage.UserOnlinePage{})
}

var cUserOnlinePage = "UserOnlinePage"

func updateUserOnlinePage(uid primitive.ObjectID, online []game.Type) {
	c := common.GetMongoDB().C(cUserOnlinePage)
	var oldOnline []dataStorage.UserOnlinePage
	if err := c.Find(bson.M{"Uid": uid}).All(&oldOnline); err != nil {
	}

	gtSame := make([]game.Type, 0)
	gtNew := make([]game.Type, 0) //新增
	gtOff := make([]game.Type, 0) //下线
	if len(oldOnline) == 0 {
		gtNew = online
	} else {
		for _, on := range online {
			found := false
			for _, old := range oldOnline {
				if on == old.GameType {
					gtSame = append(gtSame, on)
					found = true
				}
			}
			if !found {
				gtNew = append(gtNew, on)
			}
		}
		for _, old := range oldOnline {
			found := false
			for _, on := range online {
				if on == old.GameType {
					found = true
				}
			}
			if !found {
				gtOff = append(gtOff, old.GameType)
			}
		}
	}
	user := userStorage.QueryUserId(uid)
	if user.Oid.IsZero() {
		return
	}
	parserUserNewPage(user, gtNew)
	parserUserOffPage(user, gtOff)
	increaseGameOnline(user, gtSame)
}

func parserUserNewPage(user userStorage.User, gtNew []game.Type) {
	c := common.GetMongoDB().C(cUserOnlinePage)
	for _, gt := range gtNew {
		online := &dataStorage.UserOnlinePage{
			Uid:      user.Oid,
			GameType: gt,
			CreateAt: utils.Now(),
		}
		if err := c.Insert(online); err != nil {
			log.Error(err.Error())
		}
		common.GetMysql().Create(online)
	}
	updateGameOnline(user, gtNew)
}

func parserUserOffPage(user userStorage.User, gtOff []game.Type) {
	c := common.GetMongoDB().C(cUserOnlinePage)
	query := bson.M{"Uid": user.Oid, "GameType": bson.M{"$in": gtOff}}
	if _, err := c.RemoveAll(query); err != nil {
		log.Error(err.Error())
	}
	increaseGameOnline(user, gtOff)
}
func updateGameOnline(user userStorage.User, gtNew []game.Type) {
	today := utils.GetCnDate(utils.Now())
	for _, gt := range gtNew {
		data := dataStorage.GetDataGameStart(user, gt, today)
		data.Save()
	}
}
func increaseGameOnline(user userStorage.User, gtOff []game.Type) {
	today := utils.GetCnDate(utils.Now())
	for _, gt := range gtOff {
		data := dataStorage.GetDataGameStart(user, gt, today)
		onlineSec := utils.Now().Unix() - data.UpdateAt.Unix()
		data.OnlineSec += onlineSec
		data.Save()
	}
}
func OnUserPageOffline(uid string) {
	userOid := utils.ConvertOID(uid)
	c := common.GetMongoDB().C(cUserOnlinePage)
	var oldOnline []dataStorage.UserOnlinePage
	if err := c.Find(bson.M{"Uid": userOid}).All(&oldOnline); err != nil {
	}
	gtOff := make([]game.Type, 0) //下线
	for _, old := range oldOnline {
		gtOff = append(gtOff, old.GameType)
	}
	query := bson.M{"Uid": userOid}
	if _, err := c.RemoveAll(query); err != nil {
		log.Error(err.Error())
	}
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	if user.Oid.IsZero() {
		return
	}
	increaseGameOnline(user, gtOff)
}

func updateUserPage(uid primitive.ObjectID, path string) {
	screens := strings.Split(path, ";")
	gameTypes := make([]game.Type, 0)
	for _, screen := range screens {
		if gt, ok := page[screen]; ok {
			gameTypes = append(gameTypes, gt)
		}
	}
	if len(gameTypes) > 0 {
		updateUserOnlinePage(uid, gameTypes)
	} else {
		log.Warning("didn't find screen path:%v", path)
	}
}

func clearOnline() {
	c := common.GetMongoDB().C(cUserOnlinePage)
	_, _ = c.RemoveAll(bson.M{})
}

func syncUserOnline() {
	c := common.GetMongoDB().C(cUserOnlinePage)
	allSession := gate.QueryAllSession()
	uOids := make([]primitive.ObjectID, len(*allSession))
	for i, s := range *allSession {
		uOids[i] = s.Oid
	}
	query := bson.M{"Uid": bson.M{"$nin": uOids}}
	var offOnline []dataStorage.UserOnlinePage
	_ = c.Find(query).All(&offOnline)
	for _, off := range offOnline {
		OnUserPageOffline(off.Uid.Hex())
	}
	if _, err := c.RemoveAll(query); err != nil {
		log.Error(err.Error())
	}
}
func Test() {
	syncUserOnline()
}

func timerSyncUserOnline() {
	c := cron.New()
	env := common.App.GetSettings().Settings["env"].(string)
	minute1 := "*/1 * * * *" //1分钟
	if env == "dev" {
		//minite3 = "*/5 * * * * ?"
	}
	c.AddFunc(minute1, syncUserOnline)
	c.Start()
}
