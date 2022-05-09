package pk

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	mqrpc "vn/framework/mqant/rpc"
	"vn/game"
	"vn/game/activity"
	common2 "vn/game/common"
	gate2 "vn/gate"
	"vn/storage/chatStorage"
	"vn/storage/gameStorage"
	"vn/storage/miniPkStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

var (
	actionInfo        = "HD_info"
	actionOpen        = "HD_open"
	actionRecord      = "HD_record"
	actionBigPrize    = "HD_bigPrize"
	actionLeaderBoard = "HD_leaderBoard"
	leaderBoardMap    []map[string]interface{}
	pump              int64 = 2
	leaderBoardWrite        = new(sync.RWMutex)
	poker             []int8
	prizePool         sync.Map
	prizePoolMultiple int64 = 1000
	maxPoolAmount           = 800000000 // 奖池上限
)

type PK struct {
	basemodule.BaseModule
	amounts  []int64
	plays    map[int8]*miniPkStorage.PkPlay
	push     *gate2.OnlinePush
	app      module.App
	jackpots map[int][]int8
}

var Module = func() module.Module {
	return new(PK)
}

func (m *PK) Version() string {
	return "1.0.0"
}

func (m *PK) GetType() string {
	return "mini_poker"
}

func (m *PK) OnInit(app module.App, settings *conf.ModuleSettings) {
	m.BaseModule.OnInit(m, app, settings)
	m.app = app
	m.init()
	mongoIncDataExpireDay := int64(app.GetSettings().Settings["mongoIncDataExpireDay"].(float64))
	miniPkStorage.InitPokerRecord(mongoIncDataExpireDay)
	miniPkStorage.InitPkPlay()
	miniPkStorage.InitAutoIncr()
	record := &miniPkStorage.PokerRecord{}
	miniPkStorage.SetAutoIncr(record.SetName(), "Number", time.Now().Unix())
	initPkMap()
	go m.statsLeaderBoard()
	m.push = &gate2.OnlinePush{
		TraceSpan: log.CreateRootTrace(),
		App:       app,
	}
	m.push.OnlinePushInit(nil, 2048)
	hook := game.NewHook(m.GetType())
	hook.RegisterAndCheckLogin(m.GetServer(), actionInfo, m.Info)
	hook.RegisterAndCheckLogin(m.GetServer(), actionOpen, m.Open)
	hook.RegisterAndCheckLogin(m.GetServer(), actionRecord, m.Record)
	hook.RegisterAndCheckLogin(m.GetServer(), actionBigPrize, m.BigPrize)
	hook.RegisterAndCheckLogin(m.GetServer(), actionLeaderBoard, m.LeaderBoard)
}

func (m *PK) Run(closeSig chan bool) {
	log.Info("%v 模块运行中...", m.GetType())
	go m.push.Run(100 * time.Millisecond)
	<-closeSig
	log.Info("%v 模块已停止...", m.GetType())
}

func (m *PK) OnDestroy() {
	//一定别忘了继承
	m.BaseModule.OnDestroy()
	log.Info("%v 模块已回收...", m.GetType())
}

func (m *PK) Info(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	prizePool := GetPrizePool()
	data := make(map[string]interface{})
	nPoker := m.newPoker()
	m.shuffle(nPoker)
	data["Pokers"] = nPoker[:5]
	data["Amounts"] = m.amounts
	data["PrizePool"] = prizePool
	data["GroupId"] = game.MiniPoker
	go m.joinChatGroup(session)
	return errCode.Success(data).GetI18nMap(), nil
}

func (m *PK) Open(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {

	if check, err := utils.CheckParams2(params, []string{"BetAmount"}); err != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), err
	}
	betAmount := int64(params["BetAmount"].(float64))
	if _, ok := prizePool.Load(betAmount); !ok {
		return errCode.AmountNotAllow.GetMap(), nil
	}

	uid := session.GetUserID()
	user := userStorage.QueryUserId(utils.ConvertOID(uid))
	if user.Oid.IsZero() {
		return errCode.AccountNotExist.GetI18nMap(), fmt.Errorf("not find user")
	}
	bet := &miniPkStorage.PokerRecord{
		Oid:       primitive.NewObjectID(),
		Uid:       uid,
		NickName:  user.NickName,
		BetAmount: betAmount,
	}
	auto := &miniPkStorage.AutoIncr{}
	err := auto.GetAutoValue(bet.SetName(), "Number")
	if err != nil {
		log.Error("mini_poker Open() create err:%s", err.Error())
		return errCode.ServerError.GetI18nMap(), err
	}
	bet.Number = fmt.Sprintf("%d", auto.Value)
	wallet := walletStorage.QueryWallet(utils.ConvertOID(bet.Uid))
	gData := gameStorage.QueryGameCommonData(bet.Uid)
	if (wallet.VndBalance - gData.InRoomNeedVnd) < bet.BetAmount {
		return errCode.BalanceNotEnough.GetI18nMap(), fmt.Errorf("balance not enough")
	}

	if err := m.createPks(bet, user.Type); err != nil {
		return errCode.ServerError.GetI18nMap(), err
	}

	if err := m.award(bet); err != nil {
		return errCode.ServerError.GetI18nMap(), err
	}
	payType := walletStorage.TypeExpenses
	if bet.Profit > 0 {
		payType = walletStorage.TypeIncome
	}
	bill := walletStorage.NewBill(user.Oid.Hex(), payType, walletStorage.EventGameMiniPoker, bet.Oid.Hex(), bet.Profit)
	bill.Remark = "mini_poker"
	bet.SetTransactionUnits(miniPkStorage.AddPokerRecord)
	if err := walletStorage.OperateVndBalanceV1(bill, bet); err != nil {
		log.Error("wallet pay bet _id:%s err:%s", bet.Oid.Hex(), err.Error())
		return errCode.WalletPayErr.GetI18nMap(), err
	}
	if bet.Profit < 0 {
		activity.CalcEncouragementFunc(bet.Uid)
	}

	go func(record *miniPkStorage.PokerRecord, wallet *walletStorage.Wallet, userType int8) {
		botAmount := -record.Bonus
		var botProfit int64 = 0
		if bet.PumpAmount == 0 {
			botAmount = record.BetAmount * (100 - pump) / 100
			botProfit = record.BetAmount * pump / 100
		}
		if userType == userStorage.TypeNormal {
			gameStorage.IncProfit(record.Uid, game.MiniPoker, bet.PumpAmount, botAmount, botProfit)
		}
		//gameStorage.IncProfit(game.MiniPoker, bet.PumpAmount, -bet.Profit, 0)
		gameRes, _ := json.Marshal(bet.Pokers)
		betDetails := fmt.Sprintf("{\"PrizeType\":%d}", bet.PrizeType)
		params := gameStorage.BetRecordParam{
			Uid:        bet.Uid,
			GameType:   game.MiniPoker,
			Income:     bet.Profit,
			BetAmount:  bet.BetAmount,
			CurBalance: wallet.VndBalance + wallet.SafeBalance + bet.Profit,
			SysProfit:  0,
			BotProfit:  0,
			BetDetails: betDetails,
			GameId:     "",
			GameNo:     bet.Number,
			GameResult: string(gameRes),
			IsSettled:  true,
		}
		gameStorage.InsertBetRecord(params)
	}(bet, wallet, user.Type)

	data := make(map[string]interface{})

	data["Number"] = bet.Number
	data["Pokers"] = bet.Pokers
	data["PrizeType"] = bet.PrizeType
	data["Bonus"] = bet.Bonus
	return errCode.Success(data).GetI18nMap(), nil
}

// 定时开大奖
func (m *PK) jackpot() {
	go func() {
		for {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			betAmount := m.amounts[r.Intn(len(m.amounts))]
			m.sendJackpot(betAmount)
			// bots := common2.RandBotN(1, r)
			// if len(bots) >= 1 {
			// 	bot := bots[0]
			// 	bet := &miniPkStorage.PokerRecord{
			// 		Oid:       primitive.NewObjectID(),
			// 		Uid:       bot.Oid.Hex(),
			// 		NickName:  bot.NickName,
			// 		BetAmount: m.amounts[r.Intn(len(m.amounts))],
			// 		PrizeType: Prize9,
			// 		Pokers:    m.getJackpot(),
			// 	}
			// 	auto := &miniPkStorage.AutoIncr{}
			// 	err := auto.GetAutoValue(bet.SetName(), "Number")
			// 	if err != nil {
			// 		log.Error("mini_poker Open() create err:%s", err.Error())
			// 		break
			// 	}
			// 	bet.Number = fmt.Sprintf("%d", auto.Value)
			// 	if err := m.award(bet); err != nil {
			// 		continue
			// 	}
			// 	bet.AddPokerRecord()
			// }
			//time.Sleep(1 * time.Second)
			time.Sleep(time.Duration(r.Intn(86400)+60) * time.Second)
		}
	}()
}

func (m *PK) sendJackpot(betAmount int64) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	bots := common2.RandBotN(1, r)
	if len(bots) >= 1 {
		bot := bots[0]
		bet := &miniPkStorage.PokerRecord{
			Oid:       primitive.NewObjectID(),
			Uid:       bot.Oid.Hex(),
			NickName:  bot.NickName,
			BetAmount: betAmount,
			PrizeType: Prize9,
			Pokers:    m.getJackpot(),
		}
		auto := &miniPkStorage.AutoIncr{}
		err := auto.GetAutoValue(bet.SetName(), "Number")
		if err != nil {
			log.Error("mini_poker Open() create err:%s", err.Error())
			return
		}
		bet.Number = fmt.Sprintf("%d", auto.Value)
		if err := m.award(bet); err != nil {
			return
		}
		bet.AddPokerRecord()
	}
}

func (m *PK) getJackpot() (pks []int8) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	pks = m.jackpots[r.Intn(16)]
	m.shuffle(pks)
	return
}

func (m *PK) Record(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	checkKey := []string{"Limit", "Offset"}
	if check, err := utils.CheckParams2(params, checkKey); err != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), err
	}
	uid := session.GetUserID()
	offset := int(params["Offset"].(float64))
	limit := int(params["Limit"].(float64))
	pkRecord := &miniPkStorage.PokerRecord{}
	res, err := pkRecord.GetPrizeList(offset, limit, uid)
	if err != nil {
		log.Debug("bigPrize GetBigPrizeList err:%s", err.Error())
		return errCode.ServerError.GetI18nMap(), err
	}

	data := make(map[string]interface{})
	for k, v := range res {
		createAt := v["CreateAt"].(primitive.DateTime).Time().Format("2006-01-02 15:04:05")
		delete(v, "CreateAt")
		v["CreateAt"] = createAt
		res[k] = v
	}
	data["list"] = res
	return errCode.Success(data).GetI18nMap(), nil
}

func (m *PK) BigPrize(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	checkKey := []string{"Limit", "Offset"}
	if check, err := utils.CheckParams2(params, checkKey); err != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), err
	}
	offset := int(params["Offset"].(float64))
	limit := int(params["Limit"].(float64))
	pkRecord := &miniPkStorage.PokerRecord{}
	res, err := pkRecord.GetBigPrizeList(offset, limit)
	if err != nil {
		log.Debug("bigPrize GetBigPrizeList err:%s", err.Error())
		return errCode.ServerError.GetI18nMap(), err
	}
	data := make(map[string]interface{})
	for k, v := range res {
		createAt := v["CreateAt"].(primitive.DateTime).Time().Format("2006-01-02 15:04:05")
		delete(v, "CreateAt")
		v["CreateAt"] = createAt
		res[k] = v
	}
	data["list"] = res
	return errCode.Success(data).GetI18nMap(), nil
}

func (m *PK) LeaderBoard(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	checkKey := []string{"Limit", "Offset"}
	if check, err := utils.CheckParams2(params, checkKey); err != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), err
	}
	offset := int(params["Offset"].(float64))
	limit := int(params["Limit"].(float64))

	data := make(map[string]interface{})
	data["list"] = m.getLeaderBoard(offset, limit)
	return errCode.Success(data).GetI18nMap(), nil
}

// 洗牌
func (m *PK) shuffle(pk []int8) {
	utils.ShuffleInt8(pk)
}

func (m *PK) newPoker() []int8 {
	newPoker := make([]int8, len(poker))
	copy(newPoker, poker)
	return newPoker
}

func (m *PK) createPks(record *miniPkStorage.PokerRecord, userType int8) (err error) {
	profit := gameStorage.QueryProfit(game.MiniPoker)
	nPoker := m.newPoker()
	if userType != userStorage.TypeNormal {
		m.shuffle(nPoker)
		record.Pokers = nPoker[:5]
		record.PrizeType = win(record.Pokers)
		return
	}
	// 如果系统余额为负数,发两次牌都中奖才算中奖
	ResendCount := 1
	for {
		m.shuffle(nPoker)
		record.Pokers = nPoker[:5]
		record.PrizeType = win(record.Pokers)
		fmt.Println("prizeType", record.PrizeType, profit.BotBalance, record.Pokers)
		if record.PrizeType == Prize0 {
			break
		}
		if record.PrizeType == Prize9 {
			continue
		}
		if profit.BotBalance < 0 && ResendCount <= 0 {
			break
		}
		ResendCount--
		play, ok := m.plays[record.PrizeType]
		if !ok {
			err = fmt.Errorf("%s createPks not find Play", m.GetType())
			break
		}
		// record.Bonus = play.Odds * record.BetAmount / 100
		if (play.Odds * record.BetAmount / 100) < profit.BotBalance {
			break
		}
	}

	// record.Pokers = []int8{7, 1, 5, 43, 19}
	// record.PrizeType = win(record.Pokers)
	return
}

func (m *PK) award(record *miniPkStorage.PokerRecord) (err error) {
	if record.PrizeType == Prize0 {
		record.Bonus = 0
		record.Profit = -record.BetAmount
		return
	}
	if record.PrizeType != Prize9 { //大奖
		play, ok := m.plays[record.PrizeType]
		if !ok {
			err = fmt.Errorf("%s award not find Play", m.GetType())
			return
		}
		record.Bonus = play.Odds * record.BetAmount / 100
	} else {
		prize, ok := prizePool.Load(record.BetAmount)
		if !ok {
			err = fmt.Errorf("%s award not find prizePool", m.GetType())
			return
		}
		record.Bonus = prize.(int64)
		prizePool.Store(record.BetAmount, record.BetAmount*prizePoolMultiple)
	}
	record.Profit = record.Bonus - record.BetAmount
	record.PumpAmount = record.Bonus * pump / 100
	record.Pump = pump
	return
}

func (m *PK) init() {
	m.initJackpot()
	m.amounts = []int64{100, 1000, 10000, 100000, 500000}
	// init poker
	var i int8 = 1
	for ; i <= 52; i++ {
		poker = append(poker, i)
	}

	for _, v := range m.amounts {
		prizePool.Store(v, v*prizePoolMultiple)
	}
	play := &miniPkStorage.PkPlay{}
	plays, err := play.GetPlays()
	if err != nil {
		log.Error("init poker play error:%s", err.Error())
	}
	m.plays = make(map[int8]*miniPkStorage.PkPlay)
	for _, p := range plays {
		m.plays[p.PrizeType] = p
	}
	go m.addPrizePool()
	m.jackpot()
}

func (m *PK) initJackpot() {
	m.jackpots = make(map[int][]int8)
	m.jackpots[0] = []int8{7, 8, 9, 10, 11}

	for i := 1; i < 16; i++ {
		k := i % 4
		j := i / 4
		inc := int8(k + j*13)
		end := m.jackpots[0][4] + inc
		if end > int8((j+1)*13) {
			end -= 13
		}
		m.jackpots[i] = []int8{m.jackpots[0][0] + inc, m.jackpots[0][1] + inc, m.jackpots[0][2] + inc, m.jackpots[0][3] + inc, end}
	}
}

func (m *PK) addPrizePool() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		data := map[int64]int64{}
		prizePool.Range(func(k, v interface{}) bool {
			key := k.(int64)
			data[key] = v.(int64) + r.Int63n(key)
			prizePool.Store(key, data[key])
			if data[key] >= int64(maxPoolAmount) {
				m.sendJackpot(key)
			}
			return true
		})
		notify := map[string]interface{}{
			"Data":     data,
			"Action":   "increasePrize",
			"GameType": game.MiniPoker,
		}

		m.broadcast(game.Push, notify)
		time.Sleep(time.Duration(r.Intn(3)+2) * time.Second)
	}
}

func (m *PK) statsLeaderBoard() {
	pkRecord := &miniPkStorage.PokerRecord{}
	for {
		res, err := pkRecord.StatsLeaderBoard(0, 100)
		if err != nil {
			log.Error("stats leader board err:%s", err.Error())
			return
		}
		if len(res) > 0 {
			m.setLeaderBoard(res)
		}
		time.Sleep(600 * time.Second)
	}
}

func (m *PK) setLeaderBoard(data []map[string]interface{}) {
	leaderBoardWrite.Lock()
	defer leaderBoardWrite.Unlock()
	leaderBoardMap = data
}

func (m *PK) getLeaderBoard(offset, limit int) []map[string]interface{} {
	leaderBoardWrite.RLock()
	defer leaderBoardWrite.RUnlock()
	if offset < 0 || offset > len(leaderBoardMap) {
		offset = 0
	}
	end := offset + limit
	if end > len(leaderBoardMap) {
		end = len(leaderBoardMap)
	}
	return leaderBoardMap[offset:end]
}

func (m *PK) broadcast(topic string, msg map[string]interface{}) {
	uids := chatStorage.QueryGroup(string(game.MiniPoker))
	userIds := utils.ConvertUidToOid(uids)
	if len(userIds) == 0 {
		return
	}
	sessionIds := gate2.GetSessionIds(userIds)
	body, _ := json.Marshal(msg)
	m.push.SendCallBackMsgNR(sessionIds, topic, body)
}

func (m *PK) joinChatGroup(session gate.Session) bool {
	params := map[string]interface{}{"groupId": game.MiniPoker}
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
		log.Error("uid:%v join chat group:%s err:%s", session.GetUserID(), game.MiniPoker, err.Error())
		return false
	}
	return true
}

func GetPrizePool() map[int64]int64 {
	data := make(map[int64]int64)
	prizePool.Range(func(k, v interface{}) bool {
		data[k.(int64)] = v.(int64)
		return true
	})
	return data
}
