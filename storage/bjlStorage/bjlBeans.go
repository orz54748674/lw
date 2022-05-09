package bjlStorage

type Record struct {
	Result     int  `json:"result"` //开牌结果，闲赢：0，庄赢：1，和：2
	BankerPair bool `json:"bankerPair"`
	PlayerPair bool `json:"playerPair"`
	BankerDian int  `json:"bankerDian"`
	PlayerDian int  `json:"playerDian"`
}

type Process struct {
	ID              int
	ProcessName     string
	ProcessLastTime int
}

type PlayerInfo struct {
	Uid      string  `json:"uid"`
	Head     string  `json:"head"`
	NickName string  `json:"nickName"`
	Golds    int64   `json:"golds"`
	BetInfos []int64 `json:"betInfos"`
	Score    int64   `json:"score"`
}

type BetInfo struct {
	Pos      int   `json:"pos"`
	BetCount int64 `json:"betCount"`
}

type CardInfo struct {
	PlayerCards []int `json:"playerCards"`
	BankerCards []int `json:"bankerCards"`
	PlayerDian  int   `json:"playerDian"`
	BankerDian  int   `json:"bankerDian"`
}

type GameState struct {
	GameNo      string       `json:"gameNo"`
	State       string       `json:"state"`
	St          int64        `json:"st"`
	Et          int64        `json:"et"`
	BetInfos    []int64      `json:"betInfos"`
	PlayerInfos []PlayerInfo `json:"playerInfos"`
	Cards       CardInfo     `json:"cards"`
	PosRes      []bool       `json:"posRes"`
	ServerTime  int64        `json:"serverTime"`
	RemainTime  int64        `json:"remainTime"`
	MinBet      int64        `json:"minBet"`
	MaxBet      int64        `json:"maxBet"`
}

type BetDetail struct {
	UserID      string `json:"uid"`
	Pos         int    `json:"pos"`
	Coin        int64  `json:"coin"`
	PlayerGolds int64  `json:"playerGolds"`
}

type UserGameRecord struct {
	Uid        string `bson:"uid" json:"uid"`
	GameNo     string `bson:"gameNo" json:"gameNo"`
	Pos        int    `bson:"pos" json:"pos"`
	BetCount   int64  `bson:"betCount" json:"betCount"`
	Score      int64  `bson:"score" json:"score"`
	PlayerDian int    `bson:"playerDian" json:"playerDian"`
	BankerDian int    `bson:"bankerDian" json:"bankerDian"`
	CreateTime int64  `bson:"createTime" json:"createTime"`
}

type Conf struct {
	ProfitPerThousand    int     `bson:"ProfitPerThousand" json:"ProfitPerThousand"`       //系统抽水(明抽)
	BotProfitPerThousand int     `bson:"BotProfitPerThousand" json:"BotProfitPerThousand"` //暗抽
	ChipList             []int64 `bson:"ChipList" json:"ChipList"`                         //筹码列表
	BetTime              int     `bson:"BetTime" json:"BetTime"`                           //下注时间
	SendCardTime         int     `bson:"SendCardTime" json:"SendCardTime"`                 //发牌时间
	CheckoutTime         int     `bson:"CheckoutTime" json:"CheckoutTime"`                 //结算时间
	KickRoomCnt          int     `bson:"KickRoomCnt" json:"KickRoomCnt"`                   //连续三轮不下注，踢出房间
	OddList              []int64 `bson:"OddList" json:"OddList"`                           //各区域赔率  0：闲，1：庄，2：和，3：闲对，4：庄队
	PosBetLimit          []int64 `bson:"PosBetLimit" json:"PosBetLimit"`                   //区域下注限制
	MinBet               int64   `bson:"MinBet" json:"MinBet"`                             //最小下注额
	MaxBet               int64   `bson:"MaxBet" json:"MaxBet"`                             //最大下注额
}
