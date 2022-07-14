package fishStorage

import (
	"time"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/game/fish/fishConf"
)

type GameConf struct {
	TableTypeList []int       `bson:"TableBaseList" json:"TableBaseList"` //底分列表
	EnterLimit    map[int]int `bson:"GoldsLimit" json:"GoldsLimit"`       //最低金币限制
}

type RoomData struct {
	ID         int64                `bson:"-" json:"-"`
	Oid        primitive.ObjectID   `bson:"_id,omitempty" json:"Oid"`
	TablesInfo map[string]TableInfo `bson:"TablesInfo" json:"TablesInfo"` //桌子信息
}

type TableInfo struct {
	TableID   string `bson:"TableID" json:"TableID"`
	ServerID  string `bson:"ServerID" json:"ServerID"`
	TableType int    `bson:"BaseScore" json:"BaseScore"`
}

type RateByFireInfo struct {
	FireMin int     `bson:"FireMin" json:"FireMin"`
	FireMax int     `bson:"FireMax" json:"FireMax"`
	Rate    float64 `bson:"Rete" json:"Rate"`
}

type SysBalanceRateInfo struct {
	BalanceMin int64   `bson:"BalanceMin" json:"BalanceMin"`
	BalanceMax int64   `bson:"BalanceMax" json:"BalanceMax"`
	Rate       float64 `bson:"Rate" json:"Rate"`
}

type EffectBetRate struct {
	MinValue float64 `bson:"MinValue" json:"MinValue"`
	MinRate  float64 `bson:"MinRate" json:"MinRate"`
	MaxValue float64 `bson:"MaxValue" json:"MaxValue"`
	MaxRate  float64 `bson:"MaxRate" json:"MaxRate"`
}

type FishConf struct {
	ID               int64              `bson:"-" json:"-"`
	Oid              primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	CannonConf       map[int][]int64    `bson:"CannonConf" json:"CannonConf"`
	LunZhouRewardArr []int              `bson:"LunZhouRewardArr" json:"LunZhouRewardArr"`
	RateByFireArr    []RateByFireInfo   `bson:"RateByFireArr" json:"RateByFireArr"`
	SysRoom1         SysBalanceRateInfo `bson:"SysRoom1"`
	SysRoom2         SysBalanceRateInfo `bson:"SysRoom2"`
	SysRoom3         SysBalanceRateInfo `bson:"SysRoom3"`
	RateRoom1        float64            `bson:"RateRoom1"`
	RateRoom2        float64            `bson:"RateRoom2"`
	RateRoom3        float64            `bson:"RateRoom3"`
	BlockRate        float64            `bson:"BlockRate"`
	EffectBetRoom1   EffectBetRate      `bson:"EffectBetRoom1" json:"EffectBetRoom1"`
	EffectBetRoom2   EffectBetRate      `bson:"EffectBetRoom2" json:"EffectBetRoom2"`
	EffectBetRoom3   EffectBetRate      `bson:"EffectBetRoom3" json:"EffectBetRoom3"`
}

type FishPersonalPool struct {
	ID      int64              `bson:"-" json:"-"`
	Oid     primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	UserID  string             `bson:"UserID" json:"UserID"`
	Value   int64              `bson:"Value" json:"Value"`
	IsBlock bool
}

type FishPlayerConf struct {
	ID         int64              `bson:"-" json:"-"`
	Oid        primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Uid        string             `bson:"uid" json:"uid"`
	Account    string             `bson:"account" json:"account"`
	IsBlock    bool               `bson:"isblock" json:"isblock"`
	UpdateTime time.Time          `bson:"updatetime" json:"updatetime"`
}

type FishPlayerFireInfo struct {
	Uid      string    `bson:"Uid"`
	Date     string    `bson:"Date"`
	Room1    int       `bson:"Room1"`
	Room2    int       `bson:"Room2"`
	Room3    int       `bson:"Room3"`
	UpdateAt time.Time `bson:"UpdateAt"`
}

type PlayerInfo struct {
	Uid         string `json:"uid"`
	Golds       int64  `json:"golds"`
	Head        string `json:"head"`
	Name        string `json:"name"`
	Seat        int8   `json:"seat"`
	CannonType  int8   `json:"cannonType"`
	CannonGolds int64  `json:"cannonGolds"`
}

type Point struct {
	X float32
	Y float32
}

type Fish struct {
	FishID    int   `json:"fishID"`
	FishType  int   `json:"fishType"`
	Path      int   `json:"path"`
	LiveTime  int   `json:"liveTime"`
	StartTime int64 `json:"startTime"`
}

type RoomState struct {
	ServerTime int64        `json:"serverTime"`
	Players    []PlayerInfo `json:"players"`
	Fishs      []Fish       `json:"fishs"`
	Scene      int          `json:"scene"`
}

type OnFish struct {
	Fishs      []Fish `json:"fishs"`
	ServerTime int64  `json:"serverTime"`
}

type Fire struct {
	offset     float32
	fishId     int8
	cannonType int8
}

type OnFire struct {
	uid    string
	offset float32
	golds  int32
	fishId int32
}

type KillFish struct {
	uid    string
	fishId int32
	golds  int32
}

type OnKillFish struct {
	uid         string
	fishId      int32
	getGolds    int32
	playerGolds int64
}

type ChangeCannon struct {
	cannonType  int8
	cannonGolds int32
}

type OnCannon struct {
	uid         string
	cannonType  int8
	cannonGolds int32
}

//鱼潮信息
type TideInfo struct {
	StartTime int64
	EndTime   int64
	Fishs     []fishConf.FishTideKindConf
	Index     int32
}

type GroupInfo struct {
	StartTime int64
	EndTime   int64
}

type LunZhouMsg struct {
	FirstArr     []int `json:"firstArr"`
	SecondArr    []int `json:"secondArr"`
	ThirdArr     []int `json:"thirdArr"`
	SelectReward int   `json:"selectReward"`
}

type FishSysBalance struct {
	Room1          int64  `bson:"Room1" json:"Room1"`
	Room2          int64  `bson:"Room2" json:"Room2"`
	Room3          int64  `bson:"Room3" json:"Room3"`
	EffectBetRoom1 int64  `bson:"EffectBetRoom1" json:"EffectBetRoom1"`
	EffectBetRoom2 int64  `bson:"EffectBetRoom2" json:"EffectBetRoom2"`
	EffectBetRoom3 int64  `bson:"EffectBetRoom3" json:"EffectBetRoom3"`
	Date           string `bson:"Date" json:"Date"`
}
