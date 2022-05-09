package yxxStorage

import (
	"time"
	"vn/framework/mongo-driver/bson/primitive"
)

type Conf struct {
	ProfitPerThousand    int   `bson:"ProfitPerThousand" json:"ProfitPerThousand"`       //系统抽水 2%
	BotProfitPerThousand int   `bson:"BotProfitPerThousand" json:"BotProfitPerThousand"` //机器人抽水 8%
	PoolScaleThousand    int   `bson:"PoolScaleThousand" json:"PoolScaleThousand"`       //入奖千分比 500 就是 0.5
	PlayerChipsList      []int `bson:"PlayerChipsList" json:"PlayerChipsList"`           //玩家筹码列表
	XiaZhuTime           int   `bson:"XiaZhuTime" json:"XiaZhuTime"`                     //下注时间
	JieSuanTime          int   `bson:"JieSuanTime" json:"JieSuanTime"`                   //结算时间
	ReadyGameTime        int   `bson:"ReadyGameTime" json:"ReadyGameTime"`               //摇盆时间
	InitPrizePool        int   `bson:"InitPrizePool" json:"InitPrizePool"`               //初始奖池
	KickRoomCnt          int   `bson:"KickRoomCnt" json:"KickRoomCnt"`                   //连续三轮不下注，踢出房间
	ShortCutPrivate      int   `bson:"ShortCutPrivate" json:"ShortCutPrivate"`           //私人快捷语条数
	ShortCutInterval     int   `bson:"ShortCutInterval" json:"ShortCutInterval"`         //消息发送间隔
	ShortYxbLimit        int   `bson:"ShortYxbLimit" json:"ShortYxbLimit"`               //最低金币限制
}
type XiaZhuResult string

const (
	YU   XiaZhuResult = "1" //鱼
	XIA  XiaZhuResult = "2" //虾
	XIE  XiaZhuResult = "3" //蟹
	LU   XiaZhuResult = "4" //鹿
	JI   XiaZhuResult = "5" //鸡
	HULU XiaZhuResult = "6" //葫芦
)

type RoomData struct {
	ID             int64                `bson:"-" json:"-"`
	Oid            primitive.ObjectID   `bson:"_id,omitempty" json:"Oid"`
	TablesInfo     map[string]TableInfo `bson:"TablesInfo" json:"TablesInfo"` //桌子信息
	CurTableID     string               `bson:"CurTableID" json:"CurTableID"`
}
type TableInfo struct {
	TableID  string `bson:"TableID" json:"TableID"`
	PrizePool int64 `bson:"PrizePool" json:"PrizePool"` //奖池
	PrizeSwitch bool `bson:"PrizeSwitch" json:"PrizeSwitch"` //当下次盈利大于奖池瓜分  必开大奖
	ServerID string
}

type RoomRecord struct {
	ID            int64                    `bson:"-" json:"-"`
	Oid           primitive.ObjectID       `bson:"_id,omitempty" json:"Oid"`
	ResultsRecord map[string]ResultsRecord `bson:"ResultsRecord" json:"ResultsRecord"` //开奖结果
	PrizeRecord   map[string]PrizeRecord   `bson:"PrizeRecord" json:"PrizeRecord"`     //大奖瓜分记录
}
type ResultsRecord struct {
	ResultsRecordNum int                       `bson:"ResultsRecordNum" json:"ResultsRecordNum"` //战绩记录条数
	ResultsWinRate   map[XiaZhuResult]int      `bson:"ResultsWinRate" json:"ResultsWinRate"`     //图案出现几率
	Results          []map[string]XiaZhuResult `bson:"Results" json:"Results"`                   //开奖结果图案
}
type PrizeRecord struct {
	CurCnt          int64                `bson:"CurCnt" json:"CurCnt"`                   //当前期数
	PrizeWinRate    map[XiaZhuResult]int `bson:"PrizeWinRate" json:"PrizeWinRate"`       //大奖图案出现几率
	PrizeRecordList []PrizeRecordList    `bson:"PrizeRecordList" json:"PrizeRecordList"` //大奖详细记录
}
type PrizeRecordList struct { //大奖记录
	Cnt         int64        `bson:"Cnt" json:"Cnt"`                 //期数
	CreateTime  time.Time    `bson:"CreateAt" json:"CreateAt"`   //创建时间
	Result      XiaZhuResult `bson:"Result" json:"Result"`           //出现图案
	ResultsPool int64        `bson:"ResultsPool" json:"ResultsPool"` //奖池中奖金额
	PrizeList   []PrizeList  `bson:"PrizeList" json:"PrizeList"`     //瓜分名单
}
type PrizeList struct { //瓜分名单
	Name    string `bson:"Name" json:"Name"`       //用户名
	Results int64  `bson:"Results" json:"Results"` //瓜分结果
}

type RobotType string

const (
	Robot_0_1_K     RobotType = "0" //Yxb0-1K的数量
	Robot_1_20_K    RobotType = "1" //1-20K的数量
	Robot_20_50_K   RobotType = "2" //20K-50K的数量
	Robot_50_100_K  RobotType = "3" //50K-100K的数量
	Robot_100_500_K RobotType = "4" //100K-500K的数量
	Robot_500_1_M   RobotType = "5" //500K-1M的数量
	Robot_1_10_M    RobotType = "6" //1M-10M的数量
	Robot_10_30_M   RobotType = "7" //10M-30M的数量
	Robot_30_50_M   RobotType = "8" //30M-50M的数量
)

type Robot struct { //机器人
	RobotType  int    `bson:"RobotType" json:"RobotType"`   //机器人类型
	MaxBalance int    `bson:"MaxBalance" json:"MaxBalance"` //最大余额
	MinBalance int    `bson:"MinBalance" json:"MinBalance"` //最小余额
	TableID    string `bson:"TableID" json:"TableID"`       //房间号
}
type RobotRange struct {
	Max int `bson:"Max" json:"Max"` //最大值
	Min int `bson:"Min" json:"Min"` //最小值
}

type RobotConf struct { //机器人
	TableID    string `bson:"TableID" json:"TableID"`       //房间号
	StartHour  int `bson:"StartHour" json:"StartHour"` //分4个时间段 0就是 0-6点
	BaseNum  int `bson:"BaseNum" json:"BaseNum"` //数量
	MaxOffset int `bson:"MaxOffset" json:"MaxOffset"` //最大偏移量
	StepNum int `bson:"StepNum" json:"StepNum"` //每局增加量
}