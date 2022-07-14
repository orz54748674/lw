package lobbyStorage

import (
	"github.com/fatih/structs"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/storage"
	"vn/storage/userStorage"
)

var (
	cUserGrade       = "userGrade"
	cLobbyBanner     = "lobbyBanner"
	cLobbyGameLayout = "lobbyGameLayout"
	cLobbyBubble     = "lobbyBubble"
)

func GetWinRank(sumGrade *SumGrade) (uint64, error) {
	if sumGrade.WinRank == 0 {
		lastUid := userStorage.GetLastUserId()
		return uint64(lastUid), nil
	}
	c := common.GetMongoDB().C(cUserGrade)
	query := bson.M{
		"Win":      bson.M{"$gte": sumGrade.Win},
		"GameType": sumGrade.GameType,
	}
	rank, err := c.Find(query).Count()
	if err != nil {
		log.Error("get rank win: %s, err: %s", sumGrade.Win, err)
		return sumGrade.WinRank, err
	}
	return uint64(rank), nil
}
func QuerySumGrade(uid primitive.ObjectID, gameType game.Type) *SumGrade {
	c := common.GetMongoDB().C(cUserGrade)
	var sumGrade SumGrade
	query := bson.M{"_id": uid, "GameType": gameType}
	if err := c.Find(query).One(&sumGrade); err != nil {
		//log.Info("not found sum Grade uid: %s ", uid)
		return nil
	}
	return &sumGrade
}
func UpsertSumGrade(sumGrade *SumGrade) error {
	c := common.GetMongoDB().C(cUserGrade)
	selector := bson.M{"_id": sumGrade.Oid, "GameType": sumGrade.GameType}
	_, err := c.Upsert(selector, sumGrade)
	if err != nil {
		log.Error("Upsert user grade error: %s", err)
		return err
	}
	return nil
}
func Win(uid primitive.ObjectID, nickName string, win int64, gameType game.Type, isJackpot bool) {
	//parserBanner(uid,nickName,win,gameType, isJackpot)
	//if !isJackpot{
	//	//parserSumGrade(uid,win, gameType)
	//}
}
func Win2(uid primitive.ObjectID, nickName string, win int64, gameType game.Type, isJackpot bool) {
	parserBanner(uid, nickName, win, gameType, isJackpot)
	if !isJackpot {
		//parserSumGrade(uid,win, gameType)
	}
}
func parserSumGrade(uid primitive.ObjectID, win int64, gameType game.Type) {
	sumGrade := QuerySumGrade(uid, gameType)
	if sumGrade == nil {
		sumGrade = NewSumGrade(uid, gameType)
		rank, _ := GetWinRank(sumGrade)
		sumGrade.WinRank = rank
	}
	sumGrade.PlayCount = sumGrade.PlayCount + 1
	sumGrade.WinAndLost = sumGrade.WinAndLost + win
	sumGrade.UpdateTime = utils.Now()
	if win > 0 {
		sumGrade.Win = sumGrade.Win + uint64(win)
		sumGrade.WinCount = sumGrade.WinCount + 1
		sumGrade.CurWinCount = sumGrade.CurWinCount + 1
		if sumGrade.CurWinCount > sumGrade.MaxWinCount {
			sumGrade.MaxWinCount = sumGrade.CurWinCount
		}

	} else {
		sumGrade.CurWinCount = 0
	}
	if err := UpsertSumGrade(sumGrade); err != nil {
		log.Error(err.Error())
	}
}
func parserBanner(uid primitive.ObjectID, nickName string, win int64, gameType game.Type, isJackpot bool) {
	//计算 lobbyBanner
	lowestAmount, _ := utils.ConvertInt(storage.QueryConf(
		storage.KLobbyBannerLowestAmount).(string))
	if win > lowestAmount {
		winType := WinTypeNormal
		if isJackpot {
			winType = WinTypeJackpot
		}
		banner := &LobbyBanner{
			Uid:        uid,
			UserName:   nickName,
			GameType:   string(gameType),
			Amount:     win,
			WinType:    common.I18str(winType),
			CreateTime: utils.Now(),
		}
		UpsertLobbyBanner(banner)
	} //计算 lobbyBanner end
}

func QueryBanner(size int) *[]LobbyBanner {
	c := common.GetMongoDB().C(cLobbyBanner)
	var banners []LobbyBanner
	if err := c.Find(bson.M{}).Sort("-Amount").
		Limit(size).All(&banners); err != nil {
		log.Error("query lobby banner err: %s", err)
	}
	return &banners
}

func UpsertLobbyBanner(banner *LobbyBanner) {
	c := common.GetMongoDB().C(cLobbyBanner)
	query := bson.M{
		"Uid":      banner.Uid,
		"GameType": banner.GameType,
		"WinType":  banner.WinType,
	}
	update := bson.M{
		"$inc": bson.M{"Amount": banner.Amount},
		"$set": bson.M{
			"UserName": banner.UserName,
			"CreateAt": banner.CreateTime,
		},
	}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
}

func RemoveLobbyBanner(ids []primitive.ObjectID) {
	c := common.GetMongoDB().C(cLobbyBanner)
	_, _ = c.RemoveAll(bson.M{"_id": bson.M{"$in": ids}})
}

func RefreshLobbyBanner(keepTime int64) {
	c := common.GetMongoDB().C(cLobbyBanner)
	var banners []LobbyBanner
	if err := c.Find(bson.M{}).All(&banners); err != nil {
	}
	now := time.Now().Unix()
	for _, banner := range banners {
		d := banner.CreateTime.Unix() - now
		if d > keepTime {
			_ = c.Remove(bson.M{"_id": banner.Oid})
		}
	}
}

//func initLobbyBanner() []LobbyBanner{
//	banners := []LobbyBanner{
//		LobbyBanner{Text: "游客XXXX 在 XXXX 中赢取了 500000 VND ！SO COOL ！！！"},
//		LobbyBanner{Text: "游客XXXX 在 XXXX 中赢取了 500000 VND ！SO COOL ！！！"},
//		LobbyBanner{Text: "游客XXXX 在 XXXX 中赢取了 500000 VND ！SO COOL ！！！"},
//		LobbyBanner{Text: "游客XXXX 在 XXXX 中赢取了 500000 VND ！SO COOL ！！！"},
//	}
//	for _,banner := range banners{
//		_ = UpsertLobbyBanner(&banner)
//	}
//	return banners
//}
func InitNotice() {
	_ = common.GetMysql().AutoMigrate(&Notice{})
}
func InsertNotice(notice *Notice) {
	common.GetMysql().Create(&notice)
}

func InitLobbyGameLayout() {
	log.Info("init InitLobbyGameLayout of mongo db")
	c2 := common.GetMongoDB().C(cLobbyGameLayout)
	count2, err := c2.Find(bson.M{}).Count()
	if err == nil && count2 == 0 {
		InitGameLayout()
	}
}
func InitGameLayout() {
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.BiDaXiao, GameType: Normal, SortType: 1, LobbyPos: 1, IsHot: 1, Status: 1, UpdateTime: utils.Now()})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.YuXiaXie, GameType: Normal, SortType: 2, LobbyPos: 1, IsHot: 1, Status: 1, UpdateTime: utils.Now()})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.SeDie, GameType: Normal, SortType: 3, LobbyPos: 1, IsHot: 1, Status: 1, UpdateTime: utils.Now()})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.Fish, GameType: Normal, SortType: 4, LobbyPos: 1, IsHot: 1, Status: 0, UpdateTime: utils.Now()})

	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.SlotLs, GameType: Slot, SortType: 1, LobbyPos: 2, IsHot: 0, Status: 1, UpdateTime: utils.Now()})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.SlotCs, GameType: Slot, SortType: 2, LobbyPos: 2, IsHot: 0, Status: 1, UpdateTime: utils.Now()})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.SlotSex, GameType: Slot, SortType: 3, LobbyPos: 2, IsHot: 0, Status: 1, UpdateTime: utils.Now()})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.SlotDance, GameType: Slot, SortType: 4, LobbyPos: 2, IsHot: 0, Status: 1, UpdateTime: utils.Now()})

	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.CardCddS, GameType: Card, SortType: 1, LobbyPos: 3, IsHot: 0, Status: 1, UpdateTime: utils.Now(), IsNotAllowPlay: 1})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.CardCddN, GameType: Card, SortType: 2, LobbyPos: 3, IsHot: 0, Status: 1, UpdateTime: utils.Now(), IsNotAllowPlay: 1})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.CardSss, GameType: Card, SortType: 3, LobbyPos: 3, IsHot: 0, Status: 1, UpdateTime: utils.Now(), IsNotAllowPlay: 1})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.CardQzsg, GameType: Card, SortType: 4, LobbyPos: 3, IsHot: 0, Status: 1, UpdateTime: utils.Now(), IsNotAllowPlay: 1})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.CardPhom, GameType: Card, SortType: 5, LobbyPos: 3, IsHot: 0, Status: 1, UpdateTime: utils.Now(), IsNotAllowPlay: 1})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.CardCatte, GameType: Card, SortType: 6, LobbyPos: 3, IsHot: 0, Status: 1, UpdateTime: utils.Now(), IsNotAllowPlay: 1})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.CardLhd, GameType: Card, SortType: 7, LobbyPos: 3, IsHot: 0, Status: 1, UpdateTime: utils.Now()})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.Bjl, GameType: Card, SortType: 8, LobbyPos: 3, IsHot: 0, Status: 1, UpdateTime: utils.Now()})

	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.Lottery, GameType: Lottery, SortType: 1, LobbyPos: 4, IsHot: 0, Status: 1, UpdateTime: utils.Now()})

	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.MiniPoker, GameType: Mini, SortType: 1, LobbyPos: 5, IsHot: 0, Status: 1, UpdateTime: utils.Now()})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.GuessBigSmall, GameType: Mini, SortType: 2, LobbyPos: 5, IsHot: 0, Status: 1, UpdateTime: utils.Now()})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: game.Roshambo, GameType: Mini, SortType: 3, LobbyPos: 5, IsHot: 0, Status: 1, UpdateTime: utils.Now()})

	insertLobbyGameLayout(&LobbyGameLayout{GameName: "LiveCasino", GameType: Api, SortType: 1, LobbyPos: 6, IsHot: 0, Status: 1, UpdateTime: utils.Now()})
	insertLobbyGameLayout(&LobbyGameLayout{GameName: "LiveSports", GameType: Api, SortType: 2, LobbyPos: 6, IsHot: 0, Status: 1, UpdateTime: utils.Now()})
}
func insertLobbyGameLayout(lobbyGameLayout *LobbyGameLayout) {
	c := common.GetMongoDB().C(cLobbyGameLayout)
	if err := c.Insert(lobbyGameLayout); err != nil {
		log.Error(err.Error())
	}
}
func QueryLobbyGameLayout() []LobbyGameLayout {
	c := common.GetMongoDB().C(cLobbyGameLayout)
	query := bson.M{}
	var lobbyGameLayout []LobbyGameLayout
	if err := c.Find(query).All(&lobbyGameLayout); err != nil {
		log.Error(err.Error())
	}
	return lobbyGameLayout
}
func QueryLobbyGameLayoutByName(gameName game.Type) LobbyGameLayout {
	c := common.GetMongoDB().C(cLobbyGameLayout)
	selector := bson.M{"GameName": gameName}
	var lobbyGameLayout LobbyGameLayout
	if err := c.Find(selector).One(&lobbyGameLayout); err != nil {
		//log.Error(err.Error())
	}
	return lobbyGameLayout
}

//-----------------------气泡----------------------------------
func InitLobbyBubble() {
	c := common.GetMongoDB().C(cLobbyBubble)
	key := bsonx.Doc{{Key: "Uid", Value: bsonx.Int32(1)}, {Key: "BubbleType", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index()); err != nil {
		log.Error("create LobbyBubble Index: %s", err)
	}
}
func UpsertLobbyBubble(bubble LobbyBubble) {
	c := common.GetMongoDB().C(cLobbyBubble)
	selector := bson.M{"Uid": bubble.Uid, "BubbleType": bubble.BubbleType}
	update := structs.Map(bubble)
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("insert ActivityFirstCharge error: %s", err)
	}
}
func QueryLobbyBubble(uid string, bubbleType BubbleType) LobbyBubble {
	c := common.GetMongoDB().C(cLobbyBubble)
	var bubble LobbyBubble
	query := bson.M{"Uid": uid, "BubbleType": bubbleType}
	if err := c.Find(query).One(&bubble); err != nil {
		log.Info("not found LobbyBubble ", err)
	}
	return bubble
}
