package dx

import (
	"math"
	"math/rand"
	"strconv"
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
	common2 "vn/game/common"
	"vn/storage/dxStorage"

	mapset "github.com/deckarep/golang-set"
	"github.com/yireyun/go-queue"
)

type Bot struct {
	bigMax        int64
	smallMax      int64
	dxConf        *dxStorage.Conf
	queue_message *queue.EsQueue
	userIds       mapset.Set
	botBetLog     []dxStorage.DxBetLog
	r             *rand.Rand
}

func (s *Bot) Init() {
	s.r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (s *Bot) NewDxGame() {
	s.userIds = mapset.NewSet()
	if s.queue_message == nil {
		s.queue_message = queue.NewQueue(128)
	}
	s.dxConf = dxStorage.GetDxConf()
	num := s.RandInt64(1, 100)
	maxNum := s.computeMax()
	if index := num % 2; index == 1 {
		s.bigMax = maxNum[index]
		s.smallMax = maxNum[0]
	} else {
		s.bigMax = maxNum[index]
		s.smallMax = maxNum[1]
	}
}
func (s *Bot) computeMax() []int64 {
	n := make([]int64, 2)
	n[0] = s.RandInt64(int64(s.dxConf.ResultMini), int64(s.dxConf.ResultMax))
	x := float64(s.RandInt64(1, 99))
	y := (-4*math.Pow(10, -6))*math.Pow(x, 4) + 0.0005*math.Pow(x, 3) - 0.0204*math.Pow(x, 2) + 0.0644*x + 99.167
	z := 100 - int64(y)
	difference := int64(s.dxConf.MaxMiniDifference) * z / 100
	n[1] = s.RandInt64(n[0]-difference, n[0]+difference)
	return n
}

func (s *Bot) RandInt64(min, max int64) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return s.r.Int63n(max-min) + min
}
func (s *Bot) DoingBet(timeLeft int, curDx *dxStorage.Dx,curRoundBet *[]CurRoundBet) {
	if timeLeft == 3 {
		go func() {
			//log.Info("bot length: %d",len(s.botBetLog))
			dxStorage.InsertBetLogMany(s.botBetLog)
			s.botBetLog = nil
		}()
	}
	if timeLeft <= 3 {
		return
	}
	go func() {
		s.parseAmount(s.smallMax, curDx.BetSmall, timeLeft, "small", curDx,curRoundBet)
		s.parseAmount(s.bigMax, curDx.BetBig, timeLeft, "big", curDx,curRoundBet)
	}()
}
func (s *Bot) parseAmount(max int64, curBet int64, timeLeft int, position string, curDx *dxStorage.Dx,curRoundBet *[]CurRoundBet) {
	difference := max - curBet
	perSecond := difference / int64(timeLeft-3)
	betListObj := &betList{}
	s.parseBet(perSecond, betListObj,timeLeft)
	botList := common2.RandomAndNotIn(len(betListObj.amountList), s.getBotIdsArray(),s.r)
	if len(botList) < len(betListObj.amountList) {
		log.Error("botList length is %v ", len(botList))
		return
	}
	for index, amount := range betListObj.amountList {
		s.userIds.Add(botList[index].Oid)
		s.bet(amount, position, &botList[index], curDx,curRoundBet)
	}
}
func (s *Bot) getBotIdsArray() []primitive.ObjectID {
	var userIds []primitive.ObjectID
	s.userIds.Each(func(i interface{}) bool {
		userIds = append(userIds, i.(primitive.ObjectID))
		return false
	})
	return userIds
}
func (s *Bot) oneBet(position string, curDx *dxStorage.Dx,curRoundBet *[]CurRoundBet) {
	amount := GetOneBetAmount(s.dxConf, s.r)
	bots := common2.RandomAndNotIn(1, s.getBotIdsArray(),s.r)
	s.bet(amount, position, &bots[0], curDx,curRoundBet)
}
func (s *Bot) bet(amount int64, position string, bot *common2.Bot, curDx *dxStorage.Dx,curRoundBet *[]CurRoundBet) {
	gameId := curDx.ShowId
	var big int64 = 0
	var small int64 = 0
	if position == "big" {
		big = amount
		curDx.BetBigCount += 1
		curDx.BetBig += amount
	} else {
		small = amount
		curDx.BetSmallCount += 1
		curDx.BetSmall += amount
	}
	dxBetLog := dxStorage.DxBetLog{
		Uid:       strconv.Itoa(int(bot.ShowId)),
		NickName:  bot.NickName,
		GameId:    gameId,
		Big:       big,
		Small:     small,
		CurBig:    curDx.BetBig,
		CurSmall:  curDx.BetSmall,
		UserType:  dxStorage.UserTypeBot,
		MoneyType: "vnd",
		CreateAt:  utils.Now(),
	}
	find := false
	for k,v := range *curRoundBet{
		if bot.NickName == v.NickName{
			(*curRoundBet)[k].Bets += amount
		}
	}
	if !find{
		*curRoundBet = append(*curRoundBet,CurRoundBet{
			NickName: bot.NickName,
			Bets: amount,
			Position: position,
		})
	}
	s.botBetLog = append(s.botBetLog, dxBetLog)

	//s.queue_message.Put(dxBetLog)
	//s.execQueue()
}

var queueRunning = false

func (s *Bot) execQueue() {
	if queueRunning {
		return
	}
	queueRunning = true
	go func() {
		s.popQueue()
		queueRunning = false
	}()
}
func (s *Bot) popQueue() {
	ok := true
	for ok {
		val, _ok, _ := s.queue_message.Get()
		if _ok {
			dxBetLog := val.(*dxStorage.DxBetLog)
			dxStorage.IncDxBetLog(dxBetLog)
		}
		ok = _ok
	}
}

type betList struct {
	sumBet     int64
	amountList []int64
}

func (s *Bot) parseBet(perSecond int64, betList *betList,timeLeft int) *betList {
	amount := GetOneBetAmount(s.dxConf, s.r)
	betList.amountList = append(betList.amountList, amount)
	betList.sumBet += amount
	flag := false
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if timeLeft == timeLeftDefault{
		v := utils.RandInt64(1,4,r)
		if len(betList.amountList) >= int(v){
			flag = true
		}
	}else if timeLeft == timeLeftDefault - 1{
		v := utils.RandInt64(3,6,r)
		if len(betList.amountList) >= int(v){
			flag = true
		}
	}else if timeLeft == timeLeftDefault - 2{
		v := utils.RandInt64(6,9,r)
		if len(betList.amountList) >= int(v){
			flag = true
		}
	}
	if betList.sumBet > perSecond || flag{
		return betList
	} else {
		return s.parseBet(perSecond, betList,timeLeft)
	}
}
func (s *Bot) openResult(curDx *dxStorage.Dx) {

}
func GetOneBetAmount(dxConf *dxStorage.Conf, r *rand.Rand) int64 {
	isAll := false
	tmp := RandInt64(1, 100, r)
	if tmp < int64(dxConf.BotFreedomPersonPercent) {
		isAll = true
	}
	if isAll {
		amount := RandInt64(1, int64(dxConf.BotBetChip[len(dxConf.BotBetChip)-1]), r)
		return amount
	}

	l := len(dxConf.BotBetChip)
	chipIndex := RandInt64(1, int64(l)+1, r) - 1
	chip := int64(dxConf.BotBetChip[chipIndex])
	mCount := int64(len(dxConf.BotBetChip)) + 3 - chipIndex //筹码数量
	chipCount := RandInt64(1, mCount, r)
	//为了区分不同的下注
	randV := utils.RandInt64(2,6,r)
	chip = RandInt64(chip / randV,chip * randV,r)
	v := utils.RandInt64(1,4,r)
	res := chipCount * chip
	if v != 1 && res >= 1000000{
		res = res / 1000000 * 1000000
	}
	return res
}
func RandInt64(min, max int64, r *rand.Rand) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return r.Int63n(max-min) + min
}
