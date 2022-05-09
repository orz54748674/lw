package sdStorage

import (
	"vn/framework/mongo-driver/bson/primitive"
)

type Conf struct {
	ProfitPerThousand    int                    `bson:"ProfitPerThousand" json:"ProfitPerThousand"`       //系统抽水 2%
	BotProfitPerThousand int                    `bson:"BotProfitPerThousand" json:"BotProfitPerThousand"` //机器人抽水 8%
	PlayerChipsList      map[string][]int64     `bson:"PlayerChipsList" json:"PlayerChipsList"`           //玩家筹码列表
	XiaZhuTime           int                    `bson:"XiaZhuTime" json:"XiaZhuTime"`                     //下注时间
	JieSuanTime          int                    `bson:"JieSuanTime" json:"JieSuanTime"`                   //结算时间
	ReadyGameTime        int                    `bson:"ReadyGameTime" json:"ReadyGameTime"`               //摇盆时间
	KickRoomCnt          int                    `bson:"KickRoomCnt" json:"KickRoomCnt"`                   //连续三轮不下注，踢出房间
	HundredRoomNum       int                    `bson:"HundredRoomNum" json:"HundredRoomNum"`             //百人房数量
	TableBaseList        []int                  `bson:"TableBaseList" json:"TableBaseList"`               //底分列表
	SelfTablePlayerLimit int                    `bson:"SelfTablePlayerLimit" json:"SelfTablePlayerLimit"` //自创桌子人数限制
	OddsList             map[XiaZhuResult]int64 `bson:"OddsList" json:"OddsList"`                         //赔率表
	ShortCutPrivate      int                    `bson:"ShortCutPrivate" json:"ShortCutPrivate"`           //私人快捷语条数
	ShortCutInterval     int                    `bson:"ShortCutInterval" json:"ShortCutInterval"`         //消息发送间隔
	ShortYxbLimit        int                    `bson:"ShortYxbLimit" json:"ShortYxbLimit"`               //最低金币限制
}
type XiaZhuResult string

const (
	DOUBLE     XiaZhuResult = "1" //双
	SINGLE     XiaZhuResult = "2" //单
	Red4White0 XiaZhuResult = "3" //4红0白
	Red0White4 XiaZhuResult = "4" //4白
	Red1White3 XiaZhuResult = "5" //1红3白
	Red3White1 XiaZhuResult = "6" //1红3白
)

type Result string

const (
	RED   Result = "1" //红
	WHITE Result = "2" //白
)

type RoomData struct {
	ID         int64                `bson:"-" json:"-"`
	Oid        primitive.ObjectID   `bson:"_id,omitempty" json:"Oid"`
	TablesInfo map[string]TableInfo `bson:"TablesInfo" json:"TablesInfo"` //桌子信息
}
type TableInfo struct {
	TableID  string `bson:"TableID" json:"TableID"`
	ServerID string `bson:"ServerID" json:"ServerID"`

	PlayerNum int `bson:"PlayerNum" json:"PlayerNum"`
	BaseScore      int64 `bson:"BaseScore" json:"BaseScore"`           //底分
	MinEnterTable  int64 `bson:"MinEnterTable" json:"MinEnterTable"`   //进入桌子最低分
	TotalPlayerNum int   `bson:"TotalPlayerNum" json:"TotalPlayerNum"` //总人数
	Hundred        bool  `bson:"Hundred" json:"Hundred"`               //百人场

	Creator string `bson:"Creator" json:"Creator"` //创建人ID
}

type RoomRecord struct {
	ID            int64                    `bson:"-" json:"-"`
	Oid           primitive.ObjectID       `bson:"_id,omitempty" json:"Oid"`
	ResultsRecord map[string]ResultsRecord `bson:"ResultsRecord" json:"ResultsRecord"` //开奖结果
}
type ResultsRecord struct {
	ResultsRecordNum int            `bson:"ResultsRecordNum" json:"ResultsRecordNum"` //战绩记录条数
	SingleNum        int            `bson:"SingleNum" json:"SingleNum"`               //单的数量
	DoubleNum        int            `bson:"DoubleNum" json:"DoubleNum"`               //双的数量
	Results          []XiaZhuResult `bson:"Results" json:"Results"`                   //开奖结果图案
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