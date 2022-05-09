package apiStorage

import (
	"time"
	"vn/framework/mongo-driver/bson/primitive"
)

var (
	NotSettle  int8 = 0
	VoidBet    int8 = 2
	Colse      int8 = 4
	IsSettle   int8 = 8
	ViodSettle int8 = 9
	// api type start
	XgType   int8 = 1
	AwcType  int8 = 3
	WmType   int8 = 5
	SboType  int8 = 6
	SaBaType int8 = 7

	SpadeGameType int8 = 11
	// api type end

	// GameType start
	Sports     int8 = 1
	SportsName      = "Sports"
	Live       int8 = 2
	LiveName        = "Live"
	Ae         int8 = 3
	AeName          = "Ae"
	// GameType end
)

type ApiUser struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	CreateAt time.Time          `bson:"CreateAt" json:"CreateAt"`
	UpdateAt time.Time          `bson:"UpdateAt" json:"UpdateAt"`
	Uid      string             `bson:"Uid" json:"Uid"`
	Account  string             `bson:"Account" json:"Account"`
	Type     int8               `bson:"Type" json:"Type"` // 1 xg
}

type ApiRecordCheckTime struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	CreateAt time.Time          `bson:"CreateAt" json:"CreateAt"`
	UpdateAt time.Time          `bson:"UpdateAt" json:"UpdateAt"`
	Type     int8               `bson:"Type" json:"Type"` // 1 xg
	Time     time.Time          `bson:"Time" json:"Time"`
}

type ApiConfig struct {
	Oid          primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	CreateAt     time.Time          `bson:"CreateAt" json:"CreateAt"`
	UpdateAt     time.Time          `bson:"UpdateAt" json:"UpdateAt"`
	Env          string             `bson:"Env" json:"env"`
	Module       string             `bson:"Module" json:"module"`
	GameType     int8               `bson:"GameType" json:"gameType"`
	GameTypeName string             `bson:"GameTypeName" json:"gameTypeName"`
	ApiType      int8               `bson:"ApiType" json:"ApiType"`
	ApiTypeName  string             `bson:"ApiTypeName" json:"apiTypename"`
	Topic        string             `bson:"Topic" json:"topic"`
	Status       int8               `bson:"Status" json:"status"`
	ScreenType   int8               `bson:"ScreenType" json:"screenType"`
	ProductType  string             `bson:"ProductType" json:"productType"`
	Extends      string             `bson:"Extends" json:"extends"`
}

type XgBetRecord struct {
	Oid              primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	CreateAt         time.Time          `bson:"CreateAt" json:"CreateAt"`
	UpdateAt         time.Time          `bson:"UpdateAt" json:"UpdateAt"`
	SettleStatus     int8               `bson:"SettleStatus" json:"settleStatus"`
	SettleAmount     float64            `bson:"SettleAmount" json:"settleAmount"`
	RequestId        string             `bson:"RequestId" json:"requestId"`
	WagerId          int64              `bson:"WagerId" json:"wagerId"`
	SettleRequestId  string             `bson:"SettleRequestId" json:"settleRequestId"`
	User             string             `bson:"User" json:"user"`
	Uid              string             `bson:"Uid" json:"Uid"`
	Currency         string             `bson:"Currency" json:"currency"`
	Amount           float64            `bson:"Amount" json:"amount"`
	GameType         string             `bson:"GameType" json:"gameType"`
	Table            string             `bson:"Table" json:"table"`
	Round            int64              `bson:"Round" json:"round"`
	Run              int                `bson:"Run" json:"run"`
	Bet              string             `bson:"Bet" json:"bet"`
	JsonTime         string             `bson:"-" json:"betTime"`
	BetTime          time.Time          `bson:"BetTime" json:"-"`
	ReadResult       int8               `bson:"ReadResult" json:"readResult"` // 0 未读取结果,1 结果已获取
	TransactionId    string             `bson:"TransactionId" json:"transactionId"`
	ModifiedStatus   string             `bson:"ModifiedStatus" json:"modifiedStatus"`
	transactionUnits []string           `bson:"-" json:"-" gorm:"-"`
}

type AwcBetRecord struct {
	Oid              primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	CreateAt         time.Time          `bson:"CreateAt" json:"CreateAt"`
	UpdateAt         time.Time          `bson:"UpdateAt" json:"UpdateAt"`
	PlatformTxID     string             `bson:"PlatformTxID" json:"platformTxId"`
	Uid              string             `bson:"Uid" json:"uid"`
	Account          string             `bson:"Account" json:"userId"`
	Currency         string             `bson:"Currency" json:"currency"`
	Platform         string             `bson:"Platform" json:"platform"`
	GameType         string             `bson:"GameType" json:"gameType"`
	GameCode         string             `bson:"GameCode" json:"gameCode"`
	GameName         string             `bson:"GameName" json:"gameName"`
	BetType          string             `bson:"BetType" json:"betType"`
	BetAmount        float64            `bson:"BetAmount" json:"betAmount"`
	Turnover         float64            `bson:"Turnover" json:"turnover"`
	BetTime          time.Time          `bson:"betTime" json:"betTime"`
	TxTime           time.Time          `bson:"TxTime" json:"txTime"`
	UpdateTime       time.Time          `bson:"UpdateTime" json:"updateTime"`
	WinAmount        float64            `bson:"WinAmount" json:"winAmount"`
	RoundId          string             `bson:"RoundId" json:"roundId"`
	GameInfo         interface{}        `bson:"GameInfo" json:"gameInfo"`
	SettleStatus     int8               `bson:"SettleStatus" json:"settleStatus"`
	SettleCount      int8               `bson:"SettleCount" json:"settleCount"`
	VoidType         int8               `bson:"VoidType" json:"voidType"`
	transactionUnits []string           `bson:"-" json:"-" gorm:"-"`
}

type AwcGiveRecord struct {
	Oid              primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	CreateAt         time.Time          `bson:"CreateAt" json:"CreateAt"`
	UpdateAt         time.Time          `bson:"UpdateAt" json:"UpdateAt"`
	PromotionTxId    string             `bson:"PromotionTxId" json:"promotionTxId"`
	PromotionId      string             `bson:"PromotionId" json:"promotionId"`
	PromotionTypeId  string             `bson:"PromotionTypeId" json:"promotionTypeId"`
	Platform         string             `bson:"Platform" json:"platform"`
	Uid              string             `bson:"Uid" json:"uid"`
	Account          string             `bson:"Account" json:"userId"`
	Amount           float64            `bson:"Amount" json:"amount"`
	transactionUnits []string           `bson:"-" json:"-" gorm:"-"`
}

type AwcCancelBetRecord struct {
	Oid          primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	CreateAt     time.Time          `bson:"CreateAt" json:"CreateAt"`
	UpdateAt     time.Time          `bson:"UpdateAt" json:"UpdateAt"`
	Type         int8               `bson:"Type" json:"type"` // 1 cancelBet
	PlatformTxID string             `bson:"PlatformTxID" json:"platformTxId"`
}

type WmBillRecord struct {
	Oid              primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	CreateAt         time.Time          `bson:"CreateAt" json:"CreateAt"`
	UpdateAt         time.Time          `bson:"UpdateAt" json:"UpdateAt"`
	Uid              string             `bson:"Uid" json:"uid"`
	Account          string             `bson:"Account" json:"user"`
	BetAmount        float64            `bson:"BetAmount" json:"money"`
	RequestDate      time.Time          `bson:"RequestDate" json:"requestDate"`
	Dealid           string             `bson:"Dealid" json:"dealid"`
	WinAmount        float64            `bson:"WinAmount" json:"winAmount"`
	Gtype            string             `bson:"Gtype" json:"gtype"`
	Type             string             `bson:"Type" json:"type"`
	BetDetail        string             `bson:"BetDetail" json:"betdetail"`
	Code             string             `bson:"Code" json:"code"`
	GameNo           string             `bson:"GameNo" json:"gameno"`
	Category         string             `bson:"Category" json:"category"`
	BetId            string             `bson:"BetId" json:"betId"`
	Payout           float64            `bson:"Payout" json:"payout"`
	RollbackTime     time.Time          `bson:"RollbackTime" json:"rollbackTime"`
	RollbackStatus   int8               `bson:"RollbackStatus" json:"rollbackStatus"` // 1 已回滚
	transactionUnits []string           `bson:"-" json:"-" gorm:"-"`
}

type WmBetRecord struct {
	Oid            primitive.ObjectID `schema:"Oid" bson:"_id,omitempty" json:"Oid"`
	CreateAt       time.Time          `schema:"CreateAt" bson:"CreateAt" json:"CreateAt"`
	UpdateAt       time.Time          `schema:"UpdateAt" bson:"UpdateAt" json:"UpdateAt"`
	Account        string             `schema:"user" bson:"Account" json:"user"`
	Uid            string             `schema:"uid" bson:"Uid" json:"uid"`
	BetId          string             `schema:"betId" bson:"BetId" json:"betId"`
	BetTime        string             `schema:"betTime" bson:"BetTime" json:"betTime"`
	BetAmount      float64            `schema:"bet" bson:"BetAmount" json:"bet"`
	Validbet       float64            `schema:"validbet" bson:"Validbet" json:"validbet"`
	Water          float64            `schema:"water" bson:"Water" json:"water"`
	Result         string             `schema:"result" bson:"Result" json:"result"`
	BetCode        string             `schema:"betCode" bson:"BetCode" json:"betCode"`
	BetResult      string             `schema:"betResult" bson:"BetResult" json:"betResult"`
	Waterbet       float64            `schema:"waterbet" bson:"Waterbet" json:"waterbet"`
	WinLoss        float64            `schema:"winLoss" bson:"WinLoss" json:"winLoss"`
	Gid            string             `schema:"gid" bson:"Gid" json:"gid"`
	Event          string             `schema:"event" bson:"Event" json:"event"`
	EventChild     string             `schema:"eventChild" bson:"EventChild" json:"eventChild"`
	TableId        string             `schema:"tableId" bson:"TableId" json:"tableId"`
	GameResult     string             `schema:"gameResult" bson:"GameResult" json:"gameResult"`
	GName          string             `schema:"gname" bson:"GName" json:"gname"`
	BetWalletId    string             `schema:"betwalletid" bson:"BetWalletId" json:"betwalletid"`
	ResultWalletId string             `schema:"resultwalletid" bson:"ResultWalletId" json:"resultwalletid"`
	Commission     string             `schema:"commission" bson:"Commission" json:"commission"`
	Reset          string             `schema:"reset" bson:"Reset" json:"reset"`
	SetTime        string             `schema:"settime" bson:"SetTime" json:"settime"`
}

type SboBetRecord struct {
	Oid                primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	CreateAt           time.Time          `bson:"CreateAt" json:"CreateAt"`
	UpdateAt           time.Time          `bson:"UpdateAt" json:"UpdateAt"`
	Uid                string             `bson:"Uid" json:"uid"`
	Amount             float64            `bson:"Amount" json:"Amount"`
	WinLoss            float64            `bson:"WinLoss" json:"WinLoss"`
	Stake              float64            `bson:"Stake" json:"Stake"`
	CommissionStake    float64            `bson:"CommissionStake" json:"CommissionStake"`
	GameResult         string             `bson:"GameResult" json:"GameResult"`
	ResultType         int8               `bson:"ResultType" json:"ResultType"`
	TransferCode       string             `bson:"TransferCode" json:"TransferCode"`
	TransactionId      string             `bson:"TransactionId" json:"TransactionId"`
	BetTime            time.Time          `bson:"BetTime" json:"BetTime"`
	ResultTime         time.Time          `bson:"ResultTime" json:"ResultTime"`
	GameRoundId        string             `bson:"GameRoundId" json:"GameRoundId"`
	GamePeriodId       string             `bson:"GamePeriodId" json:"GamePeriodId"`
	OrderDetail        string             `bson:"OrderDetail" json:"OrderDetail"`
	PlayerIp           string             `bson:"PlayerIp" json:"PlayerIp"`
	GameTypeName       string             `bson:"GameTypeName" json:"GameTypeName"`
	CompanyKey         string             `bson:"CompanyKey" json:"CompanyKey"`
	Username           string             `bson:"Username" json:"Username"`
	ProductType        int                `bson:"ProductType" json:"ProductType"`
	GameType           int                `bson:"GameType" json:"GameType"`
	Gpid               int                `bson:"Gpid" json:"Gpid"`
	Status             string             `bson:"Status" json:"Status"`
	CancelBeforeStatus string             `bson:"CancelBeforeStatus" json:"CancelBeforeStatus"`
	Rollback           bool               `bson:"Rollback" json:"Rollback"`
	IsCancelAll        bool               `bson:"IsCancelAll" json:"IsCancelAll"`
	transactionUnits   []string           `bson:"-" json:"-" gorm:"-"`
}

type SabaBetRecord struct {
	Oid              primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	CreateAt         time.Time          `bson:"CreateAt" json:"CreateAt"`
	UpdateAt         time.Time          `bson:"UpdateAt" json:"UpdateAt"`
	Uid              string             `bson:"Uid" json:"uid"`
	OperationId      string             `bson:"OperationId" json:"operationId"`
	UserId           string             `bson:"UserId" json:"userId"`
	Currency         int                `bson:"Currency" json:"currency"`
	MatchId          int                `bson:"MatchId" json:"matchId"`
	HomeId           int                `bson:"HomeId" json:"homeId"`
	AwayId           int                `bson:"AwayId" json:"awayId"`
	HomeName         string             `bson:"HomeName" json:"homeName"`
	AwayName         string             `bson:"AwayName" json:"awayName"`
	KickOffTime      time.Time          `bson:"KickOffTime" json:"kickOffTime"`
	BetTime          time.Time          `bson:"BetTime" json:"betTime"`
	BetAmount        float64            `bson:"BetAmount" json:"betAmount"`
	ActualAmount     float64            `bson:"ActualAmount" json:"actualAmount"`
	SportType        int                `bson:"SportType" json:"sportType"`
	SportTypeName    string             `bson:"SportTypeName" json:"sportTypeName"`
	BetType          int                `bson:"BetType" json:"betType"`
	BetTypeName      string             `bson:"BetTypeName" json:"betTypeName"`
	OddsType         int                `bson:"OddsType" json:"oddsType"`
	OddsId           int                `bson:"OddsId" json:"oddsId"`
	Odds             float64            `bson:"Odds" json:"odds"`
	BetChoice        string             `bson:"BetChoice" json:"betChoice"`
	BetChoiceEn      string             `bson:"BetChoiceEn" json:"betChoice_en"`
	UpdateTime       time.Time          `bson:"UpdateTime" json:"updateTime"`
	LeagueId         int                `bson:"LeagueId" json:"leagueId"`
	LeagueName       string             `bson:"LeagueName" json:"leagueName"`
	LeagueNameEn     string             `bson:"LeagueNameEn" json:"leagueName_en"`
	SportTypeNameEn  string             `bson:"SportTypeNameEn" json:"sportTypeName_en"`
	BetTypeNameEn    string             `bson:"BetTypeNameEn" json:"betTypeName_en"`
	HomeNameEn       string             `bson:"HomeNameEn" json:"homeName_en"`
	AwayNameEn       string             `bson:"AwayNameEn" json:"awayName_en"`
	IP               string             `bson:"IP" json:"IP"`
	IsLive           bool               `bson:"IsLive" json:"isLive"`
	RefId            string             `bson:"RefId" json:"refId"`
	TsId             string             `bson:"TsId" json:"tsId"`
	TxId             int64              `bson:"TxId" json:"txId"`
	BaStatus         bool               `bson:"BaStatus" json:"baStatus"`
	Point            string             `bson:"Point" json:"point"`
	Point2           string             `bson:"Point2" json:"point2"`
	BetTeam          string             `bson:"BetTeam" json:"betTeam"`
	HomeScore        int                `bson:"HomeScore" json:"homeScore"`
	AwayScore        int                `bson:"AwayScore" json:"awayScore"`
	HtHomeScore      int                `bson:"HtHomeScore" json:"htHomeScore"`
	HtAwayScore      int                `bson:"HtAwayScore" json:"htAwayScore"`
	Excluding        string             `bson:"Excluding" json:"excluding"`
	BetFrom          string             `bson:"BetFrom" json:"betFrom"`
	CreditAmount     float64            `bson:"CreditAmount" json:"creditAmount"`
	DebitAmount      float64            `bson:"DebitAmount" json:"debitAmount"`
	Payout           float64            `bson:"Payout" json:"payout"`
	WinlostDate      time.Time          `bson:"WinlostDate" json:"winlostDate"`
	Status           string             `bson:"Status" json:"Status"`
	SettleStatus     string             `bson:"SettleStatus" json:"settleStatus"`
	CashStatus       int                `bson:"CashStatus" json:"cashStatus"`
	ParlayType       string             `bson:"ParlayType" json:"parlayType"`
	Detail           string             `bson:"Detail" json:"detail"`
	MatchIds         []int              `bson:"MatchIds" json:"match_ids"`
	MatchOids        []string           `bson:"MatchOids" json:"Match_oids"`
	GameStatus       string             `bson:"GameStatus" json:"gameStatus"`
	IsGameData       bool               `bson:"IsGameData" json:"isGameData"`
	IsOddsChanged    bool               `bson:"IsOddsChanged" json:"isOddsChanged"`
	transactionUnits []string           `bson:"-" json:"-" gorm:"-"`
}
