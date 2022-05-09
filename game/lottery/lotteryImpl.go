package lottery

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	mqrpc "vn/framework/mqant/rpc"
	"vn/game"
	"vn/game/activity"
	gate2 "vn/gate"
	"vn/storage/activityStorage"
	"vn/storage/chatStorage"
	"vn/storage/gameStorage"
	"vn/storage/lotteryStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"

	"github.com/mitchellh/mapstructure"
	"github.com/mohae/deepcopy"
)

const (
	actionInfo             = "HD_info"
	actionRecord           = "HD_record"
	actionAddBet           = "HD_addBet"
	actionGetBetRecordList = "HD_getBetRecordList"
	actionBetMsgLog        = "HD_betMsgLog"
)

var combinationPlay = []string{"ZH", "BCZH"}

type Impl struct {
	app                 module.App
	push                *gate2.OnlinePush
	playMap             sync.Map
	lotteryMap          sync.Map
	clientPlayRWLock    *sync.RWMutex
	clientLotteryRWLock *sync.RWMutex
	clientPlayMap       map[string][]*lotteryStorage.Play
	clientLotteryMap    map[string][]map[string]interface{}
}

func (m *Impl) initPlayMap() {
	m.clientPlayRWLock.Lock()
	defer m.clientPlayRWLock.Unlock()
	play := &lotteryStorage.LotteryPlay{}
	playList, err := play.GetLotteryPlays()
	if err != nil {
		log.Error("initPlayMap err:%s", err.Error())
	}
	for _, p := range playList {
		key := fmt.Sprintf("%s_%s_%s", p.AreaCode, p.PlayCode, p.SubPlayCode)
		m.playMap.Store(key, p)
	}
	m.clientPlayMap = make(map[string][]*lotteryStorage.Play)
	tmpMap := make(map[string]map[string]*lotteryStorage.Play)

	m.playMap.Range(func(k, play interface{}) bool {
		p := play.(*lotteryStorage.LotteryPlay)
		subPlay := &lotteryStorage.SubPlay{
			SubName:          p.SubName,
			SubPlaySort:      p.SubPlaySort,
			Odds:             p.Odds,
			CodeRule:         p.CodeRule,
			Rules:            p.Rules,
			SubPlayCode:      p.SubPlayCode,
			CodeLength:       p.CodeLength,
			MaxBetNumber:     p.MaxBetNumber,
			UnitBetAmount:    p.UnitBetAmount,
			UnitBetCodeCount: p.UnitBetCodeCount,
			OpenCodeCount:    p.OpenCodeCount,
			Description:      p.Description,
		}
		if _, ok := tmpMap[p.AreaCode]; ok {
			if _, exist := tmpMap[p.AreaCode][p.PlayCode]; exist {
				tmpMap[p.AreaCode][p.PlayCode].SubPlays = append(tmpMap[p.AreaCode][p.PlayCode].SubPlays, subPlay)
			} else {
				tmpMap[p.AreaCode][p.PlayCode] = &lotteryStorage.Play{
					Name:     p.Name,
					PlayCode: p.PlayCode,
					PlaySort: p.PlaySort,
					SubPlays: []*lotteryStorage.SubPlay{subPlay},
				}
			}
		} else {
			areaPlay := make(map[string]*lotteryStorage.Play)
			areaPlay[p.PlayCode] = &lotteryStorage.Play{
				Name:     p.Name,
				PlayCode: p.PlayCode,
				PlaySort: p.PlaySort,
				SubPlays: []*lotteryStorage.SubPlay{subPlay},
			}
			tmpMap[p.AreaCode] = areaPlay
		}
		return true
	})
	for key, areaPlay := range tmpMap {
		for _, v := range areaPlay {
			m.clientPlayMap[key] = append(m.clientPlayMap[key], v)
		}
	}
	for key, areaPlay := range m.clientPlayMap {
		sort.SliceStable(areaPlay, func(i, j int) bool {
			return areaPlay[i].PlaySort < areaPlay[j].PlaySort
		})
		for k, p := range areaPlay {
			sort.SliceStable(p.SubPlays, func(i, j int) bool {
				return p.SubPlays[i].SubPlaySort < p.SubPlays[j].SubPlaySort
			})
			areaPlay[k] = p
		}
		m.clientPlayMap[key] = areaPlay
	}
}
func (m *Impl) initLotteryMap() {
	m.clientLotteryRWLock.Lock()
	defer m.clientLotteryRWLock.Unlock()
	lottery := &lotteryStorage.Lottery{}
	lotteries, err := lottery.GetLotteries()
	if err != nil {
		log.Error("GetLotteries err:%s", err.Error())
	}

	m.clientLotteryMap = make(map[string][]map[string]interface{})
	for _, lottery := range lotteries {
		m.lotteryMap.Store(fmt.Sprintf("%s_%v", lottery.LotteryCode, int(lottery.WeekNumber)), lottery)

		weekDay := fmt.Sprintf("%d", int(lottery.WeekNumber))

		lMap := map[string]interface{}{}
		if err := mapstructure.Decode(lottery, &lMap); err != nil {
			log.Debug("initLotteryMap map structure.Decode err:%s", err.Error())
			continue
		}
		if lottery.AreaCode == lotteryStorage.North {
			lMap["CityName"] = lottery.AreaName
		}
		delete(lMap, "CollectUrl")
		delete(lMap, "Oid")
		delete(lMap, "Remark")
		delete(lMap, "Status")
		delete(lMap, "AreaName")
		m.clientLotteryMap[weekDay] = append(m.clientLotteryMap[weekDay], lMap)
	}
}

func (m *Impl) open(session gate.Session, data map[string]interface{}) (res map[string]interface{}, err error) {
	log.Debug("open params:%v", data)
	params := &struct {
		Number      string                                 `json:"number"`
		LotteryCode string                                 `json:"lotteryCode"`
		OpenCode    map[lotteryStorage.PrizeLevel][]string `json:"openCode"`
	}{}
	if err = mapstructure.Decode(data, params); err != nil {
		log.Error("lottery open mapstructure.Decode err:%v", err.Error())
		return
	}
	betRecord := &lotteryStorage.LotteryBetRecord{}
	_, err = betRecord.SetOpenCode(params.Number, params.LotteryCode, params.OpenCode)
	if err != nil {
		log.Error("lottery Code: %s set Number:%s err:%s", params.LotteryCode, params.Number, err.Error())
		return
	}
	rpcParams := map[string]interface{}{}
	rpcParams["Number"] = params.Number
	rpcParams["LotteryCode"] = params.LotteryCode
	ctx, _ := context.WithTimeout(context.TODO(), time.Second*3) //3s后超时
	rpcRes, err := mqrpc.String(m.app.Call(
		ctx,
		"settle",        //要访问的moduleType
		"/lottery/open", //访问模块中handler路径
		mqrpc.Param(rpcParams),
	))
	if err != nil {
		log.Debug("lottery Code: %s  rpc res:%v, err：%v", params.LotteryCode, res, err.Error())
		return
	}
	log.Debug("lottery Code: %s open Number:%s end  rpc res:%v", params.LotteryCode, params.Number, rpcRes)
	return
}
func (m *Impl) modifyBetInfo(session gate.Session, data map[string]interface{}) (res map[string]interface{}, err error) {
	res = map[string]interface{}{}
	params := &struct {
		Oid         string `json:"oid"`
		Number      string `json:"number"`
		LotteryCode string `json:"lotteryCode"`
	}{}
	if err = mapstructure.Decode(data, params); err != nil {
		log.Error("lottery modifyBetInfo mapstructure.Decode err:%v", err.Error())
		return errCode.ErrParams.GetI18nMap(), err
	}
	t, err := utils.StrToTime(fmt.Sprintf("%v 00:00:00", params.Number))
	if err != nil {
		log.Error("lottery modifyBetInfo utils.StrToTime err:%v", err.Error())
		return errCode.LotteryNumberErr.GetI18nMap(), err
	}
	iLottery, ok := m.lotteryMap.Load(fmt.Sprintf("%s_%v", params.LotteryCode, int(t.Weekday())))
	if !ok {
		return errCode.LotteryNumberErr.GetI18nMap(), fmt.Errorf("期号与彩种不服")
	}
	lottery := iLottery.(*lotteryStorage.Lottery)
	strOpenTime := lottery.OpenTime
	openTime, err := utils.StrToTime(fmt.Sprintf("%s %s", params.Number, strOpenTime))
	if err != nil {
		log.Error("lottery modifyBetInfo utils.StrToTime err:%v", err.Error())
		return errCode.ServerError.GetI18nMap(), err
	}
	mBetRecord := &lotteryStorage.LotteryBetRecord{
		Oid:         utils.ConvertOID(params.Oid),
		Number:      params.Number,
		CnNumber:    utils.GetCnDate(t),
		OpenTime:    openTime,
		AreaCode:    lottery.AreaCode,
		CityCode:    lottery.CityCode,
		LotteryCode: lottery.LotteryCode,
	}
	err = mBetRecord.ModifyBetInfo()
	if err != nil {
		log.Error("lottery modifyBetInfo mBetRecord.ModifyBetInfo err:%v", err.Error())
		return
	}
	gameStorage.ModifyBetNumber(params.Oid, params.Number)
	return errCode.Success(nil).GetI18nMap(), nil
}
func (m *Impl) numberRefund(session gate.Session, data map[string]interface{}) (res map[string]interface{}, err error) {
	log.Debug("numberRefund params:%v", data)
	params := &struct {
		LotteryCode string `json:"lotteryCode"`
		Number      string `json:"number"`
	}{}
	if err = mapstructure.Decode(data, params); err != nil {
		log.Error("lottery numberRefund mapstructure.Decode err:%v", err.Error())
		return
	}
	mBetRecord := &lotteryStorage.LotteryBetRecord{}
	offset, limit := 0, 1000
	for {
		bets, err := mBetRecord.GetNumberRecords(params.LotteryCode, params.Number, offset, limit)
		if err != nil && err != mongo.ErrNoDocuments {
			log.Error("lottery batchRefund mBetRecord.GetNumberRecords err:%v", err.Error())
			break
		}
		for _, bet := range bets {
			go m.cancel(bet)
			time.Sleep(100 * time.Millisecond)
		}
		if len(bets) < limit {
			break
		}
		offset += limit
	}

	return
}

func (m *Impl) batchRefund(session gate.Session, data map[string]interface{}) (res map[string]interface{}, err error) {
	log.Debug("batchRefund params:%v", data)
	params := &struct {
		Oids []string `json:"oids"`
	}{}
	if err = mapstructure.Decode(data, params); err != nil {
		log.Error("lottery batchRefund mapstructure.Decode err:%v", err.Error())
		return
	}
	log.Debug("lottery batchRefund params.Oids:%v", params.Oids)
	if len(params.Oids) == 0 {
		return
	}
	oids := []primitive.ObjectID{}
	for _, sId := range params.Oids {
		oids = append(oids, utils.ConvertOID(sId))
	}
	log.Debug("lottery batchRefund oids:%v", oids)
	mBetRecord := &lotteryStorage.LotteryBetRecord{}
	bets, err := mBetRecord.GetBetByOids(oids)
	if err != nil {
		log.Error("lottery batchRefund mBetRecord.GetBetByOids err:%v", err.Error())
		return
	}
	log.Debug("lottery batchRefund mBetRecord.GetBetByOids bets:%v", bets)
	for _, bet := range bets {
		if bet.SettleStatus == 0 {
			go m.cancel(bet)
		}
		time.Sleep(100 * time.Millisecond)
	}
	return
}
func (m *Impl) cancel(bet *lotteryStorage.LotteryBetRecord) {
	bet.SettleStatus = 7 // 取消状态
	bet.SProfit = 0
	bill := walletStorage.NewBill(bet.Uid, walletStorage.TypeIncome, walletStorage.EventGameLottery, bet.Oid.Hex(), bet.TotalAmount)
	bill.Remark = "lottery cancel bet"
	bet.SetTransactionUnits(lotteryStorage.CancelSettleStatus)
	if err := walletStorage.OperateVndBalanceV1(bill, bet); err != nil {
		log.Error("cancel bet wallet pay bet _id:%s err:%s", bet.Oid.Hex(), err.Error())
		return
	}
	gameStorage.CloseLotteryBetRecord(bet.Oid.Hex())
	activityStorage.UpsertGameDataInBet(bet.Uid, game.Lottery, -1)
	activity.CalcEncouragementFunc(bet.Uid)
}

/**
 * @title info
 * @description   进入页面初始化所需数据
 */
func (m *Impl) info(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	nowTime := time.Now()
	nowDate := utils.GetDate(nowTime)
	lotteries := map[string][]map[string]interface{}{}
	for i := 0; i <= 6; i++ {
		key := fmt.Sprintf("%d", i)
		weekDayLotteries := m.getClientLottery(key)
		for _, lottery := range weekDayLotteries {
			// lottery := deepcopy.Copy(item).(map[string]interface{})
			if i == int(nowTime.Weekday()) || i == int(nowTime.Add(time.Hour*24).Weekday()) {
				st, _ := utils.StrToTime(fmt.Sprintf("%s %v", nowDate, lottery["StopBetTime"]))
				ot, _ := utils.StrToTime(fmt.Sprintf("%s %v", nowDate, lottery["OpenTime"]))
				lottery["StopBetTimestamp"] = st.Unix()
				lottery["OpenTimestamp"] = ot.Unix()
				if st.Unix() <= nowTime.Unix() {
					lottery["StopBetTimestamp"] = lottery["StopBetTimestamp"].(int64) + 86400
				}
				if ot.Unix() <= nowTime.Unix() {
					lottery["OpenTimestamp"] = lottery["OpenTimestamp"].(int64) + 86400
				}
			}
			lotteries[key] = append(lotteries[key], lottery)
			// m.clientLotteryMap[key][k] = lottery
		}
	}
	data := make(map[string]interface{})
	data["Lotteries"] = lotteries
	data["Plays"] = m.getClientPlay()
	data["GroupId"] = game.Lottery
	data["JoinGroup"] = m.joinChatGroup(session)
	data["Timestamp"] = time.Now().Unix()
	return errCode.Success(data).GetI18nMap(), nil

}

func (m *Impl) joinChatGroup(session gate.Session) bool {
	params := map[string]interface{}{"groupId": game.Lottery}
	ctx, _ := context.WithTimeout(context.TODO(), time.Second*3) //3s后超时
	_, err := mqrpc.InterfaceMap(
		m.app.Call(
			ctx,
			"chat",         //要访问的moduleType
			"HD_joinGroup", //访问模块中handler路径
			mqrpc.Param(session, params),
		),
	)
	if err != nil {
		log.Error("uid:%v join chat group:%s err:%s", session.GetUserID(), game.Lottery, err.Error())
		return false
	}
	return true
}

/**
 * @title lotteryRecord
 * @description 获取lottery 记录
 */
func (m *Impl) record(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	if check, err := utils.CheckParams2(msg, []string{"LotteryCode"}); err != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), err
	}
	record := &lotteryStorage.LotteryRecord{}
	lotteryCode := msg["LotteryCode"].(string)
	data, err := record.GetRecordList(lotteryCode)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return errCode.DataNotFind.GetI18nMap(), err
		} else {
			log.Error("record.GetRecordList('%s') err:%s", lotteryCode, err.Error())
			return errCode.ServerError.GetI18nMap(), err
		}
	}

	return errCode.Success(data).GetI18nMap(), nil
}

/**
 * @title addBet
 * @description 下注接口
 */
func (m *Impl) addBet(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	user := userStorage.QueryUserId(utils.ConvertOID(session.GetUserID()))
	if user.Oid.IsZero() {
		return errCode.AccountNotExist.GetI18nMap(), fmt.Errorf("not find user")
	}

	checkKey := []string{"LotteryCode", "SubPlayCode", "Number", "Code", "UnitBetAmount", "TotalAmount"}
	if check, err := utils.CheckParams2(params, checkKey); err != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), err
	}
	bet := &lotteryStorage.LotteryBetRecord{}
	if err := mapstructure.Decode(params, bet); err != nil {
		return errCode.ErrParams.GetI18nMap(), err
	}
	wallet := walletStorage.QueryWallet(user.Oid)
	if wallet.VndBalance < bet.TotalAmount {
		return errCode.BalanceNotEnough.GetI18nMap(), fmt.Errorf("balance not enough")
	}
	numTime, err := utils.StrToTime(fmt.Sprintf("%v 00:00:00", bet.Number))
	if err != nil {
		log.Error("lottery addBet utils.StrToTime err:%v", err.Error())
		return errCode.LotteryNumberErr.GetI18nMap(), err
	}

	iLottery, ok := m.lotteryMap.Load(fmt.Sprintf("%s_%v", bet.LotteryCode, int(numTime.Weekday())))
	if !ok {
		return errCode.InvalidLotteryCode.GetI18nMap(), fmt.Errorf("sync.map not find lotteryCode:%v", bet.LotteryCode)
	}
	lottery := iLottery.(*lotteryStorage.Lottery)
	bet.NickName = user.NickName
	bet.AreaCode = lottery.AreaCode
	bet.CityCode = lottery.CityCode
	bet.Uid = user.Oid.Hex()

	index := strings.Index(bet.SubPlayCode, "_")
	if index < 0 {
		err := fmt.Errorf("SubPlayCode not find '_' 字符")
		log.Error(err.Error())
		return errCode.LotteryPlayErr.GetI18nMap(), err
	}
	bet.PlayCode = bet.SubPlayCode[0:index]
	stopTime, err := utils.StrToTime(fmt.Sprintf("%s %s", bet.Number, lottery.StopBetTime))

	if err != nil || (err == nil && (stopTime.Unix() < time.Now().Unix())) {
		if err == nil {
			err = fmt.Errorf("the betting time has passed the closing time")
		}
		return errCode.LotteryNumberErr.GetI18nMap(), err
	}
	bet.OpenTime, _ = utils.StrToTime(fmt.Sprintf("%s %s", bet.Number, lottery.OpenTime))
	bet.CnNumber = utils.GetCnDate(stopTime)
	play := &lotteryStorage.LotteryPlay{}
	if err := play.GetPlayInfo(bet.AreaCode, bet.PlayCode, bet.SubPlayCode); err != nil {
		log.Error("bet.AreaCode:%s, bet.PlayCode:%s, bet.SubPlayCode:%s LotteryPlayErr:%s,", bet.AreaCode, bet.PlayCode, bet.SubPlayCode, err.Error())
		return errCode.LotteryPlayErr.GetI18nMap(), err
	}
	// 把 code 排序
	codes := strings.Split(bet.Code, "-")
	sort.Strings(codes)
	bet.Code = strings.Join(codes, "-")
	// 判断code 是否合法
	if err := m.checkBetCode(bet, play); err != nil {
		return err.GetI18nMap(), fmt.Errorf(err.ErrMsg)
	}

	bet.Odds = play.Odds
	if err := bet.Add(); err != nil {
		log.Error("addBet func=>bet.Add() err:%s", err.Error())
		return errCode.ServerError.GetI18nMap(), err
	}
	bill := walletStorage.NewBill(user.Oid.Hex(), walletStorage.TypeExpenses, walletStorage.EventGameLottery, bet.Oid.Hex(), -1*bet.TotalAmount)
	bet.PayStatus = lotteryStorage.PayEnd
	bet.SetTransactionUnits(lotteryStorage.ChangePayStatus)
	if err := walletStorage.OperateVndBalanceV1(bill, bet); err != nil {
		log.Error("wallet pay bet _id:%s err:%s", bet.Oid.Hex(), err.Error())
		return errCode.WalletPayErr.GetI18nMap(), err
	}
	activityStorage.UpsertGameDataInBet(bet.Uid, game.Lottery, 1)
	data := make(map[string]interface{})
	m.broadcastUserBetInfo(bet)
	betDetails := map[string]interface{}{
		"BetCode":     bet.Code,
		"AreaCode":    bet.AreaCode,
		"LotteryCode": bet.LotteryCode,
		"CityName":    lottery.CityName,
		"SubPlayCode": bet.SubPlayCode,
		"Oid":         bet.Oid.Hex(),
		"Odds":        bet.Odds,
		"VndBalance":  wallet.VndBalance + wallet.SafeBalance - bet.TotalAmount,
	}
	jsonDetails, _ := json.Marshal(betDetails)
	betRecordData := gameStorage.BetRecordParam{
		Uid:        bet.Uid,
		GameType:   game.Lottery,
		Income:     0,
		BetAmount:  bet.TotalAmount,
		CurBalance: 0,
		SysProfit:  0,
		BotProfit:  0,
		BetDetails: string(jsonDetails),
		GameId:     bet.Oid.Hex(),
		GameNo:     bet.Number,
		GameResult: bet.Oid.Hex(),
		IsSettled:  false,
	}
	gameStorage.InsertBetRecord(betRecordData)
	return errCode.Success(data).GetI18nMap(), nil
}

/**
 * @title betMsgLog
 * @description 获取广播下注信息
 */
func (m *Impl) betMsgLog(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"size"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), ok
	}
	size, _ := utils.ConvertInt(params["size"])
	if size > 100 {
		return errCode.PageSizeErr.GetI18nMap(), nil
	}
	betRecord := &lotteryStorage.LotteryBetRecord{}
	records, err := betRecord.GetBets(0, int(size))
	if err != nil {
		log.Error("GetBets err:%s", err.Error())
		return errCode.ServerError.GetI18nMap(), err
	}
	var msgList []map[string]interface{}
	for _, bRecord := range records {
		msg := m.getBetMsg(bRecord)
		if msg != nil {
			msgList = append(msgList, msg)
		}
	}
	res := map[string]interface{}{"betMsgList": msgList}
	return errCode.Success(res).GetI18nMap(), nil
}

/**
 * @title getBetRecordList
 * @description 获取用户下注记录接口
 */
func (m *Impl) getBetRecordList(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	checkKey := []string{"limit", "offset"}
	if check, err := utils.CheckParams2(params, checkKey); err != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), err
	}
	uid := session.GetUserID()
	offset := int(params["offset"].(float64))
	limit := int(params["limit"].(float64))
	if limit <= 0 || limit > 1000 {
		limit = 10
	}
	record := lotteryStorage.LotteryBetRecord{}
	list, err := record.GetRecordListByUid(uid, offset, limit)
	if err != nil {
		log.Error("getBetRecordList select data err:%s", err.Error())
		return errCode.ServerError.GetI18nMap(), nil
	}
	for k, bRecord := range list {
		numTime, err := utils.StrToTime(fmt.Sprintf("%v 00:00:00", bRecord["Number"]))
		if err != nil {
			log.Error("lottery getBetRecordList utils.StrToTime err:%v", err.Error())
			continue
		}
		oTime := bRecord["OpenTime"].(primitive.DateTime)
		delete(list[k], "OpenTime")
		list[k]["OpenTime"] = oTime.Time().Format("02-01")
		iLottery, ok := m.lotteryMap.Load(fmt.Sprintf("%v_%v", bRecord["LotteryCode"], int(numTime.Weekday())))
		if !ok {
			log.Debug("key:%s", fmt.Sprintf("%v_%v", bRecord["LotteryCode"], int(numTime.Weekday())))
			continue
		}
		lottery := iLottery.(*lotteryStorage.Lottery)
		list[k]["CityName"] = lottery.CityName
		if lottery.AreaCode == lotteryStorage.North {
			list[k]["CityName"] = lottery.AreaName
		}
		delete(list[k], "LotteryCode")

		subPlayKey := fmt.Sprintf("%s_%v_%s", lottery.AreaCode, bRecord["PlayCode"], bRecord["SubPlayCode"])
		delete(list[k], "PlayCode")
		delete(list[k], "SubPlayCode")
		iPlay, _ := m.playMap.Load(subPlayKey)
		play := iPlay.(*lotteryStorage.LotteryPlay)
		list[k]["SubPlayName"] = play.SubName
		list[k]["UnitBetAmount"] = play.UnitBetAmount * play.OpenCodeCount * 1000
	}
	data := map[string]interface{}{"List": list}
	return errCode.Success(data).GetI18nMap(), nil
}

func (m *Impl) noticeOpen(params map[string]interface{}) (r string, err error) {
	number := params["Number"].(string)
	lotteryCode := params["LotteryCode"].(string)
	record := &lotteryStorage.LotteryRecord{}
	log.Debug("noticeOpen %s  %s", number, lotteryCode)
	res, err := record.GetNumberOpenCode(number, lotteryCode)
	if err != nil {
		log.Error("record.GetNumberOpenCode err:%s", err.Error())
		return
	}
	data := map[string]interface{}{
		"LotteryCode": lotteryCode,
		"Record": map[string]interface{}{
			"Number":   number,
			"OpenCode": res.OpenCode,
		},
	}
	notify := map[string]interface{}{
		"Data":     data,
		"Action":   "noticeOpen",
		"GameType": "lottery",
	}
	m.broadcast(game.Push, notify)
	r = "notice open ok"
	return
}

func (m *Impl) broadcastUserBetInfo(bet *lotteryStorage.LotteryBetRecord) {
	user := userStorage.QueryUserId(utils.ConvertOID(bet.Uid))
	if user.Oid.IsZero() {
		log.Debug("broadcastUserBetInfo not find user")
		return
	}
	playKey := fmt.Sprintf("%s_%s_%s", bet.AreaCode, bet.PlayCode, bet.SubPlayCode)
	play, ok := m.playMap.Load(playKey)
	if !ok {
		log.Debug("broadcastUserBetInfo not find play")
		return
	}
	numTime, err := utils.StrToTime(fmt.Sprintf("%v 00:00:00", bet.Number))
	if err != nil {
		log.Error("lottery broadcastUserBetInfo utils.StrToTime err:%v", err.Error())
		return
	}
	l, ok := m.lotteryMap.Load(fmt.Sprintf("%s_%v", bet.LotteryCode, int(numTime.Weekday())))
	if !ok {
		log.Debug("broadcastUserBetInfo not find lottery")
		return
	}
	lottery := l.(*lotteryStorage.Lottery)
	data := map[string]interface{}{
		"NickName":    bet.NickName,
		"TotalAmount": bet.TotalAmount,
		"Code":        bet.Code,
		"SubPlayName": play.(*lotteryStorage.LotteryPlay).SubName,
		"AreaName":    lottery.AreaName,
		"CityName":    lottery.CityName}
	if lottery.AreaCode == lotteryStorage.North {
		data["CityName"] = lottery.AreaName
	}
	notify := map[string]interface{}{
		"Data":     data,
		"Action":   "addBet",
		"GameType": "lottery",
	}
	m.broadcast(game.Push, notify)
}

func (m *Impl) broadcastTime() {
	count := 0
	for {
		data := map[string]interface{}{"Timestamp": time.Now().Unix()}
		notify := map[string]interface{}{
			"Data":     data,
			"Action":   "timestamp",
			"GameType": "lottery",
		}
		m.broadcast(game.Push, notify)
		time.Sleep(time.Second * 10)
		count++
	}
}

func (m *Impl) broadcast(topic string, msg map[string]interface{}) {
	uids := chatStorage.QueryGroup(string(game.Lottery))
	userIds := utils.ConvertUidToOid(uids)
	if len(userIds) == 0 {
		//log.Debug("No online users need to be notified")
		return
	}
	sessionIds := gate2.GetSessionIds(userIds)
	body, _ := json.Marshal(msg)
	m.push.SendCallBackMsgNR(sessionIds, topic, body)
}

func (m *Impl) getBetMsg(bet *lotteryStorage.LotteryBetRecord) map[string]interface{} {
	playKey := fmt.Sprintf("%s_%s_%s", bet.AreaCode, bet.PlayCode, bet.SubPlayCode)
	play, ok := m.playMap.Load(playKey)
	if !ok {
		log.Debug("broadcastUserBetInfo not find play")
		return nil
	}
	numTime, err := utils.StrToTime(fmt.Sprintf("%v 00:00:00", bet.Number))
	if err != nil {
		log.Error("lottery getBetMsg utils.StrToTime err:%v", err.Error())
		return nil
	}
	l, ok := m.lotteryMap.Load(fmt.Sprintf("%s_%v", bet.LotteryCode, int(numTime.Weekday())))
	if !ok {
		log.Debug("broadcastUserBetInfo not find lottery")
		return nil
	}
	lottery := l.(*lotteryStorage.Lottery)
	cityName := lottery.CityName
	if lottery.AreaCode == lotteryStorage.North {
		cityName = lottery.AreaName
	}
	return map[string]interface{}{
		"NickName":    bet.NickName,
		"TotalAmount": bet.TotalAmount,
		"Code":        bet.Code,
		"SubPlayName": play.(*lotteryStorage.LotteryPlay).SubName,
		"CityName":    cityName,
	}
}

func (m *Impl) checkBetCode(bet *lotteryStorage.LotteryBetRecord, play *lotteryStorage.LotteryPlay) *common.Err {
	codes := strings.Split(bet.Code, "-")
	if len(codes) > play.MaxBetNumber {
		log.Debug("checkBetCode err: bet.Code:%v >  MaxBetNumber:%v", len(codes), play.MaxBetNumber)
		return errCode.BetCodeErr
	}
	for _, code := range codes {
		if len(code) > play.CodeLength {
			log.Debug("checkBetCode err: unitCode:%v   CodeLength:%v", len(code), play.CodeLength)
			return errCode.BetCodeErr
		}
	}

	betNumber := int64(len(codes)) / play.UnitBetCodeCount
	log.Debug("checkBetCode (betNumber * bet.UnitBetAmount * play.OpenCodeCount):%v,  bet.TotalAmount:%v", betNumber*bet.UnitBetAmount*play.OpenCodeCount, bet.TotalAmount)
	if (betNumber * bet.UnitBetAmount * play.OpenCodeCount) != bet.TotalAmount {
		log.Debug("checkBetCode err: Amount err ")
		return errCode.BetCodeErr
	}
	betRecords, err := bet.GetUserRecords(bet.Uid, bet.Number, bet.LotteryCode, bet.SubPlayCode)
	if err != nil {
		log.Error("checkBetCode => GetUserRecords err:%s ", err.Error())
		return errCode.BetCodeErr
	}
	codes = []string{}
	betRecords = append(betRecords, bet)
	for _, betRecord := range betRecords {
		if utils.StrInArray(betRecord.PlayCode, combinationPlay) {
			log.Debug("if %s %v", betRecord.PlayCode, combinationPlay)
			if !utils.StrInArray(betRecord.Code, codes) {
				codes = append(codes, betRecord.Code)
			}
		} else {
			log.Debug("else %s %v", betRecord.PlayCode, combinationPlay)
			tmpCodes := strings.Split(betRecord.Code, "-")
			for _, code := range tmpCodes {
				if !utils.StrInArray(code, codes) {
					codes = append(codes, code)
				}
			}
		}
	}
	count := len(codes)
	log.Debug("PlayCode:%s count:%d play.MaxCodeCount:%d", play.PlayCode, count, play.MaxCodeCount)
	if count > play.MaxCodeCount {
		log.Debug("checkBetCode err: the number of bets in a single period exceeds(count:%d, MaxCodeCount:%d)", count, play.MaxCodeCount)
		return errCode.BetLimit
	}
	return nil
}

func (m *Impl) getClientPlay() map[string][]*lotteryStorage.Play {
	m.clientPlayRWLock.RLock()
	defer m.clientPlayRWLock.RUnlock()
	return m.clientPlayMap
}

func (m *Impl) getClientLottery(weekNumber string) []map[string]interface{} {
	m.clientLotteryRWLock.RLock()
	defer m.clientLotteryRWLock.RUnlock()
	return deepcopy.Copy(m.clientLotteryMap[weekNumber]).([]map[string]interface{})
}

func (m *Impl) reloadPlay(session gate.Session, params map[string]interface{}) (res map[string]interface{}, err error) {
	m.initPlayMap()
	m.configChange()
	return
}

func (m *Impl) reloadLottery(session gate.Session, params map[string]interface{}) (res map[string]interface{}, err error) {
	m.initLotteryMap()
	m.configChange()
	return
}

func (m *Impl) configChange() {
	data := map[string]interface{}{"change": true}
	notify := map[string]interface{}{
		"Data":     data,
		"Action":   "configChange",
		"GameType": "lottery",
	}
	m.broadcast(game.Push, notify)
}
