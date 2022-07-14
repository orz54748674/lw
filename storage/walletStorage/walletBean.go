package walletStorage

import (
	"context"
	"encoding/json"
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo"
	"vn/framework/mqant/log"
)

type CallBack interface {
	TransactionUnit(db *mongo.Database, ctx context.Context) error
}

type Wallet struct {
	ID           uint64             `bson:"-" json:"-"`
	Oid          primitive.ObjectID `bson:"_id,omitempty" json:"Oid" gorm:"unique"`
	VndBalance   int64              `bson:"VndBalance"`
	AgentBalance int64              `bson:"AgentBalance"`
	SafeBalance  int64              `bson:"SafeBalance"`
	UpdateAt     time.Time          `bson:"UpdateAt"`
}

func (s *Wallet) JsonData() []byte {
	msg := map[string]Wallet{"wallet": *s}
	b, err := json.Marshal(msg)
	if err != nil {
		log.Error(err.Error())
	}
	return b
}
func (Wallet) TableName() string {
	return "wallet"
}

type Bill struct {
	ID       uint64             `bson:"-" json:"-"`
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Uid      string             `bson:"Uid"`
	Type     string             `bson:"Type"`
	Event    string             `bson:"Event"`
	EventId  string             `bson:"EventId"` //游戏结算用gameShowId
	Amount   int64              `bson:"Amount"`  //支出一定是负数
	Balance  int64              `bson:"Balance"` //结算后余额
	Status   int8               `bson:"Status"`
	Remark   string             `bson:"Remark"`
	CreateAt time.Time          `bson:"CreateAt"`
	UpdateAt time.Time          `bson:"UpdateAt"`
}

func (Bill) TableName() string {
	return "bill"
}

var (
	TypeIncome   = "income"   //收入
	TypeExpenses = "expenses" //支出

	EventGameDx            = "gameDx"
	EventGameDxJackpot     = "gameDxJackpot"
	EventGameYxx           = "gameYxx"
	EventGameSd            = "gameSd"
	EventGameSlotLs        = "gameSlotLs"
	EventGameSlotDance     = "gameSlotDance"
	EventGameSlotCs        = "gameSlotCs"
	EventGameSlotSex       = "gameSlotSex"
	EventGameCardSss       = "gameCardSss"
	EventGameFish          = "gameFish"
	EventGameCardCddS      = "gameCardCddS"
	EventGameCardCddN      = "gameCardCddN"
	EventGameCardPhom      = "gameCardPhom"
	EventGameCardCatte     = "gameCardCatte"
	EventGameCardLhd       = "gameCardLhd"
	EventGameCardQzsg      = "gameCardQzsg"
	EventAgentIncome       = "agentIncome"
	EventAgentIncomeRefund = "agentIncomeRefund"
	EventAgentDouDou       = "agentDouDou"
	EventCharge            = "charge"
	EventFromAgent         = "fromAgent"
	EventDouDou            = "douDou"
	EventActivityAward     = "activityAward"
	EventDouDouRefund      = "douDouRefund"
	EventAdmin             = "Admin"
	EventGameLottery       = "gameLottery"
	EventGameMiniPoker     = "gameMiniPoker"
	EventGameBjl           = "gameBjl"
	EventGameRoshambo      = "gameRoshambo"
	EventGameGuessBs       = "gameGuessBs"
	EventGameXg            = "gameXg"
	EventGameSport         = "gameApiSport"
	EventApiCmd            = "apiCmd"
	EventGameAwc           = "gameAwc"
	EventGameWm            = "gameWm"
	EventGameSbo           = "gameSbo"
	EventApiDg             = "apiDg"
	EventGameSaba          = "gameSaba"
	EventGameCq9           = "apiCq9"

	//活动
	EventGiftCode      = "giftCode"
	EventFirstCharge   = "firstCharge"
	EventBindPhone     = "bindPhone"
	EventTotalCharge   = "totalCharge"
	EventSignIn        = "signIn"
	EventEncouragement = "encouragement"
	EventDayCharge     = "dayCharge"
	EventDayGame       = "dayGame"
	EventDayInvite     = "dayInvite"
	EventVip           = "vip"
	EventVipWeek       = "vipWeek"
	EventVipChargeGift = "vipChargeGift"
	EventTurnTable     = "turnTable"
	EventSafeChange    = "SafeChange"

	StatusInit int8 = 0
)
var ActivityEvent = []string{ //用来统计活动
	EventGiftCode,
	EventFirstCharge,
	EventBindPhone,
	EventTotalCharge,
	EventSignIn,
	EventEncouragement,
	EventDayCharge,
	EventDayGame,
	EventDayInvite,
}

func newWallet(uid primitive.ObjectID) *Wallet {
	wallet := &Wallet{
		Oid:      uid,
		UpdateAt: utils.Now(),
	}
	return wallet
}
