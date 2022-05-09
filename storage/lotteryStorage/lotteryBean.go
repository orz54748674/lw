package lotteryStorage

import (
	"fmt"
	"time"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
)

type PrizeLevel string

type PlayCode string

type Area string

type City string

type Lottery struct {
	Oid              primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	LotteryName      string             `bson:"LotteryName" json:"LotteryName"`         // 名字
	LotteryType      int8               `bson:"LotteryType" json:"LotteryType"`         // 1 官方 2 系统
	LotteryCode      string             `bson:"LotteryCode" json:"LotteryCode"`         // code
	WeekNumber       time.Weekday       `bson:"WeekNumber" json:"WeekNumber"`           // 0 6 周日 ~ 周六
	AreaName         string             `bson:"AreaName" json:"AreaName"`               //
	AreaCode         string             `bson:"AreaCode" json:"AreaCode"`               //
	CityName         string             `bson:"CityName" json:"CityName"`               //
	CityCode         string             `bson:"CityCode" json:"CityCode"`               //
	StartTime        string             `bson:"StartTime" json:"StartTime"`             //
	Intervals        int64              `bson:"Intervals" json:"Intervals"`             //
	EndTime          string             `bson:"EndTime" json:"EndTime"`                 //
	AdvanceStopTime  int64              `bson:"AdvanceStopTime" json:"AdvanceStopTime"` // 封
	StopBetTime      string             `bson:"StopBetTime" json:"StopBetTime"`         //
	StopBetTimestamp int64              `bson:"-" json:"StopBetTimestamp"`              //
	OpenTime         string             `bson:"OpenTime" json:"OpenTime"`               //
	OpenTimestamp    int64              `bson:"-" json:"OpenTimestamp"`                 //
	CollectUrl       string             `bson:"CollectUrl" json:"CollectUrl"`           // 采集地址
	Status           int8               `bson:"Status" json:"Status"`                   // 是否开启 0 开始  1停止
	Remark           string             `bson:"Remark" json:"Remark"`                   // 备注
}

type LotteryRecord struct {
	Oid         primitive.ObjectID      `bson:"_id,omitempty" json:"Oid"`       //
	LotteryCode string                  `bson:"LotteryCode" json:"LotteryCode"` // code
	AreaCode    string                  `bson:"AreaCode" json:"AreaCode"`       //
	CityCode    string                  `bson:"CityCode" json:"CityCode"`       //
	Number      string                  `bson:"Number" json:"Number"`           //
	CnNumber    string                  `bson:"CnNumber" json:"CnNumber"`       // 年月日格式便于直接字符串对比
	Date        string                  `bson:"Date" json:"Date"`               //
	WeekNumber  time.Weekday            `bson:"WeekNumber" json:"WeekNumber"`   // 0 6 周日 ~ 周六
	OpenCode    map[PrizeLevel][]string `bson:"OpenCode" json:"OpenCode"`       //
	OpenTime    time.Time               `bson:"OpenTime" json:"OpenTime"`       //
	IsSettle    int8                    `bson:"IsSettle" json:"IsSettle"`       // 3 结算中 6结算未完成  8 结算完成
	CollectTime time.Time               `bson:"CollectTime" json:"CollectTime"` //
	CollectUrl  string                  `bson:"CollectUrl" json:"CollectUrl"`   // 采集地址
	CreateAt    time.Time               `bson:"CreateAt"`
	UpdateAt    time.Time               `bson:"UpdateAt"`
}

type LotteryBetRecord struct {
	Oid              primitive.ObjectID      `bson:"_id,omitempty" json:"Oid"`
	Uid              string                  `bson:"Uid" json:"Uid"`
	NickName         string                  `bson:"NickName" json:"NickName"`
	LotteryCode      string                  `bson:"LotteryCode" json:"LotteryCode"`
	AreaCode         string                  `bson:"AreaCode" json:"AreaCode"`
	CityCode         string                  `bson:"CityCode" json:"CityCode"`
	PlayCode         string                  `bson:"PlayCode" json:"PlayCode"`
	SubPlayCode      string                  `bson:"SubPlayCode" json:"SubPlayCode"`
	Number           string                  `bson:"Number" json:"Number"`
	CnNumber         string                  `bson:"CnNumber" json:"CnNumber"` // 年月日格式便于直接字符串对比
	OpenTime         time.Time               `bson:"OpenTime" json:"OpenTime"` // 开奖时间
	OpenCode         map[PrizeLevel][]string `bson:"OpenCode" json:"OpenCode"  gorm:"-"`
	Code             string                  `bson:"Code" json:"Code"`
	Odds             int64                   `bson:"Odds" json:"Odds"`
	UnitBetAmount    int64                   `bson:"UnitBetAmount" json:"UnitBetAmount"` // 注单单价
	TotalAmount      int64                   `bson:"TotalAmount" json:"TotalAmount"`     // 注单总金额
	CProfit          int64                   `bson:"CProfit" json:"CProfit"`             // 客户端计算可盈利金额
	SProfit          int64                   `bson:"SProfit" json:"SProfit"`             // 服务端计算可盈利金额
	Remark           string                  `bson:"Remark" json:"Remark"`
	PayStatus        int8                    `bson:"PayStatus" json:"PayStatus"`       // 支付状态 0 未支付 1 已支付
	SettleStatus     int8                    `bson:"SettleStatus" json:"SettleStatus"` // 注单记录状态 0 初始 1 结算中 2 结算失败 7 取消 8 已结算
	Status           int8                    `bson:"Status" json:"Status"`             // 是否开启 1 开始 0 停止
	CreateAt         time.Time               `bson:"CreateAt"`
	UpdateAt         time.Time               `bson:"UpdateAt"`
	transactionUnits []string                `bson:"-" json:"-" gorm:"-"`
}

type LotteryPlay struct {
	Oid              primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	AreaCode         string             `bson:"AreaCode" json:"AreaCode"`
	PlayCode         string             `bson:"PlayCode" json:"PlayCode"`
	Name             string             `bson:"Name" json:"Name"`
	SubName          string             `bson:"SubName" json:"SubName"`
	SubPlayCode      string             `bson:"SubPlayCode" json:"SubPlayCode"`
	Odds             int64              `bson:"Odds" json:"Odds"`
	UnitBetAmount    int64              `bson:"UnitBetAmount" json:"UnitBetAmount"`       // 注单单价
	UnitBetCodeCount int64              `bson:"UnitBetCodeCount" json:"UnitBetCodeCount"` // 单注码的数量
	OpenCodeCount    int64              `bson:"OpenCodeCount" json:"OpenCodeCount"`       // 满足开奖码个数
	CodeLength       int                `bson:"CodeLength" json:"CodeLength"`             // 单个码的长度
	MaxBetNumber     int                `bson:"MaxBetNumber" json:"MaxBetNumber"`         // 最大允许多少注或单注下多少个码
	MaxCodeCount     int                `bson:"MaxCodeCount" json:"MaxCodeCount"`         // 单期最大下注码数
	CodeRule         string             `bson:"CodeRule" json:"CodeRule"`                 // 不同玩法，下注是否合法
	PlaySort         int                `bson:"PlaySort" json:"PlaySort"`                 // 玩法展示顺序 升序
	SubPlaySort      int                `bson:"SubPlaySort" json:"SubPlaySort"`           // 子玩法展示顺序 升序
	Description      string             `bson:"Description" json:"Description"`
	Rules            map[string][]int   `bson:"Rules" json:"Rules"`
	Remark           string             `bson:"Remark" json:"Remark"`
	Status           int8               `bson:"Status" json:"Status"`
}
type Play struct {
	PlayCode string `json:"PlayCode"`
	Name     string `json:"Name"`
	PlaySort int    `json:"-"`
	SubPlays []*SubPlay
}

type SubPlay struct {
	SubPlaySort      int              `json:"-"`
	SubName          string           `json:"SubName"`
	SubPlayCode      string           `json:"SubPlayCode"`
	Odds             int64            `json:"Odds"`
	CodeRule         string           `json:"CodeRule"`
	Rules            map[string][]int `json:"Rules"`
	CodeLength       int              `json:"CodeLength"`
	MaxBetNumber     int              `json:"MaxBetNumber"`
	UnitBetAmount    int64            `json:"UnitBetAmount"` // 注单单价
	UnitBetCodeCount int64            `json:"UnitBetCodeCount"`
	OpenCodeCount    int64            `json:"OpenCodeCount"` // 满足开奖码个数
	Description      string           `json:"Description"`
}

// 预设 PresetOpenCode
type PresetOpenCode struct {
	Oid         primitive.ObjectID      `bson:"_id,omitempty" json:"Oid"`
	Uid         string                  `bson:"Uid" json:"Uid"`                 // Setter user id
	LotteryCode string                  `bson:"LotteryCode" json:"LotteryCode"` // code
	Number      string                  `bson:"Number" json:"Number"`
	OpenCode    map[PrizeLevel][]string `bson:"OpenCode" json:"OpenCode"`
	CreateAt    time.Time               `bson:"CreateAt"`
	UpdateAt    time.Time               `bson:"UpdateAt"`
}

type BetStats struct {
	Oid            primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Number         string             `bson:"Number" json:"Number"`
	AreaCode       string             `bson:"AreaCode" json:"AreaCode"`       //
	LotteryCode    string             `bson:"LotteryCode" json:"LotteryCode"` // code
	PlayCode       string             `bson:"PlayCode" json:"PlayCode"`
	SubPlayCode    string             `bson:"SubPlayCode" json:"SubPlayCode"`
	UnitBetCode    string             `bson:"UnitBetCode" json:"UnitBetCode"`
	TotalBetAmount string             `bson:"TotalBetAmount" json:"TotalBetAmount"`
	TotalPatAmount string             `bson:"TotalPatAmount" json:"TotalPatAmount"`
	BetCodeCount   string             `bson:"BetCodeCount" json:"BetCodeCount"`
	Odds           int64              `bson:"Odds" json:"Odds"`
}

type rule struct {
	Level   PrizeLevel
	Count   int
	Max     int
	CodeLen int
}

var (
	OfficialLottery = 1
	SystemLottery   = 2
	Enable          = 1
	Disable         = 0
	North           = "North"
	Central         = "Central"
	South           = "South"
	CodeRuleMap     = map[string]map[PrizeLevel]*rule{
		South: map[PrizeLevel]*rule{
			PrizeLevel0: &rule{Level: PrizeLevel0, CodeLen: 6, Count: 1, Max: 999999},
			PrizeLevel1: &rule{Level: PrizeLevel1, CodeLen: 5, Count: 1, Max: 99999},
			PrizeLevel2: &rule{Level: PrizeLevel2, CodeLen: 5, Count: 1, Max: 99999},
			PrizeLevel3: &rule{Level: PrizeLevel3, CodeLen: 5, Count: 2, Max: 99999},
			PrizeLevel4: &rule{Level: PrizeLevel4, CodeLen: 5, Count: 7, Max: 99999},
			PrizeLevel5: &rule{Level: PrizeLevel5, CodeLen: 4, Count: 1, Max: 9999},
			PrizeLevel6: &rule{Level: PrizeLevel6, CodeLen: 4, Count: 3, Max: 9999},
			PrizeLevel7: &rule{Level: PrizeLevel7, CodeLen: 3, Count: 1, Max: 999},
			PrizeLevel8: &rule{Level: PrizeLevel8, CodeLen: 2, Count: 1, Max: 99},
		},
		North: map[PrizeLevel]*rule{
			PrizeLevel0: &rule{Level: PrizeLevel0, CodeLen: 5, Count: 1, Max: 99999},
			PrizeLevel1: &rule{Level: PrizeLevel1, CodeLen: 5, Count: 1, Max: 99999},
			PrizeLevel2: &rule{Level: PrizeLevel2, CodeLen: 5, Count: 2, Max: 99999},
			PrizeLevel3: &rule{Level: PrizeLevel3, CodeLen: 5, Count: 6, Max: 99999},
			PrizeLevel4: &rule{Level: PrizeLevel4, CodeLen: 4, Count: 4, Max: 9999},
			PrizeLevel5: &rule{Level: PrizeLevel5, CodeLen: 4, Count: 5, Max: 9999},
			PrizeLevel6: &rule{Level: PrizeLevel6, CodeLen: 3, Count: 3, Max: 999},
			PrizeLevel7: &rule{Level: PrizeLevel7, CodeLen: 2, Count: 4, Max: 99},
		},
		Central: map[PrizeLevel]*rule{
			PrizeLevel0: &rule{Level: PrizeLevel0, CodeLen: 6, Count: 1, Max: 999999},
			PrizeLevel1: &rule{Level: PrizeLevel1, CodeLen: 5, Count: 1, Max: 99999},
			PrizeLevel2: &rule{Level: PrizeLevel2, CodeLen: 5, Count: 1, Max: 99999},
			PrizeLevel3: &rule{Level: PrizeLevel3, CodeLen: 5, Count: 2, Max: 99999},
			PrizeLevel4: &rule{Level: PrizeLevel4, CodeLen: 5, Count: 7, Max: 99999},
			PrizeLevel5: &rule{Level: PrizeLevel5, CodeLen: 4, Count: 1, Max: 9999},
			PrizeLevel6: &rule{Level: PrizeLevel6, CodeLen: 4, Count: 3, Max: 9999},
			PrizeLevel7: &rule{Level: PrizeLevel7, CodeLen: 3, Count: 1, Max: 999},
			PrizeLevel8: &rule{Level: PrizeLevel8, CodeLen: 2, Count: 1, Max: 99},
		},
	}
)

const (
	PrizeLevel0 PrizeLevel = "L0" //
	PrizeLevel1 PrizeLevel = "L1" //
	PrizeLevel2 PrizeLevel = "L2" //
	PrizeLevel3 PrizeLevel = "L3" //
	PrizeLevel4 PrizeLevel = "L4" //
	PrizeLevel5 PrizeLevel = "L5" //
	PrizeLevel6 PrizeLevel = "L6" //
	PrizeLevel7 PrizeLevel = "L7" //
	PrizeLevel8 PrizeLevel = "L8" //
)

func CheckOpenCode(areaCode string, openCode map[PrizeLevel][]string) (err error) {
	if CodeRule, exist := CodeRuleMap[areaCode]; !exist {
		err = fmt.Errorf("CodeRule key:%s not find value", areaCode)
	} else {
		for level, codes := range openCode {
			for _, code := range codes {
				if len(code) != CodeRule[level].CodeLen || !utils.IsNumber(code) {
					err = fmt.Errorf("level:%s code:%s not number", level, code)
					break
				}
			}
		}
	}
	return
}
