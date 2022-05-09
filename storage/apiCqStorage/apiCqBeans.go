package apiCqStorage

import (
	"time"
	"vn/framework/mongo-driver/bson/primitive"
)

type ApiUser struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	CreateAt time.Time          `bson:"CreateAt" json:"CreateAt"`
	UpdateAt time.Time          `bson:"UpdateAt" json:"UpdateAt"`
	Uid      string             `bson:"Uid" json:"Uid"`
	Account  string             `bson:"Account" json:"Account"`
	Type     int8               `bson:"Type" json:"Type"` // 1 xg
}

type BetInfo struct {
	MTCode    string  `bson:"Mtcode" json:"mtcode"`
	Amount    float64 `bson:"Amount" json:"amount"`
	RoundID   string  `bson:"RoundID" json:"roundid"`
	EventTime string  `bson:"EventTime" json:"eventtime"`
}

type Bets struct {
	Account    string    `bson:"Account" json:"account"`
	Session    string    `bson:"Session" json:"session"`
	GameHall   string    `bson:"GameHall" json:"gamehall"`
	GameCode   string    `bson:"GameCode" json:"gamecode"`
	Data       []BetInfo `bson:"Data" json:"data"`
	CreateTime string    `bson:"CreateTime" json:"createTime"`
}

type MTCode struct {
	Oid       primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	RecordOid string             `bson:"recordOid" json:"recordOid"`
	Mtcode    string             `bson:"mtcode" json:"mtcode"`
	Account   string             `bson:"account" json:"account"`
}

type TargetInfo struct {
	Account string `bson:"account" json:"account"`
}

type StatusInfo struct {
	CreateTime string `bson:"createtime" json:"createtime"`
	EndTime    string `bson:"endtime" json:"endtime"`
	Status     string `bson:"status" json:"status"`
	Message    string `bson:"message" json:"message"`
}

type EventInfo struct {
	Mtcode    string `bson:"mtcode" json:"mtcode"`
	Amount    int64  `bson:"amount" json:"amount"`
	EventTime string `bson:"eventtime" json:"eventtime"`
	Status    string `bson:"status" json:"status"`
	Action    string `bson:"action" json:"action"`
}

type StatusMsg struct {
	Code     string `bson:"code" json:"code"`
	Message  string `bson:"message" json:"message"`
	Datatime string `bson:"datatime" json:"datatime"`
}

type DataInfo struct {
	DataID   string      `bson:"dataID" json:"_id`
	Action   string      `bson:"bets" json:"bets"`
	Target   TargetInfo  `bson:"target" json:"target"`
	Status   StatusInfo  `bson:"status" json:"status"`
	Before   int64       `bson:"before" json:"before"`
	Balance  int64       `bson:"balance" json:"balance"`
	Currency string      `bson:"currency" json:"currency"`
	Events   []EventInfo `bson:"event" json:"event"`
}

type BetRecords struct {
	Oid    primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Data   DataInfo           `bson:"data" json:"data"`
	Status StatusMsg          `bson:"status" json:"status"`
}

type AmendData struct {
	MTCode    string  `json:"mtcode"`
	Amount    float64 `json:"amount"`
	ValidBet  float64 `json:"validbet"`
	Action    string  `json:"action"`
	RoundID   string  `json:"roundid"`
	EventTime string  `json:"eventtime"`
}

type Amend struct {
	Account    string      `json:"account"`
	GameHall   string      `json:"gamehall"`
	GameCode   string      `json:"gamecode"`
	Action     string      `json:"action"`
	Amount     float64     `json:"amount"`
	CreateTime string      `json:"createTime"`
	Data       []AmendData `json:"data"`
}

type Bet struct {
	Account   string  `json:"account"`
	EventTime string  `json:"eventTime"`
	GameHall  string  `json:"gamehall"`
	GameCode  string  `json:"gamecode"`
	RoundID   string  `json:"roundid"`
	Amount    float64 `json:"amount"`
	Mtcode    string  `json:"mtcode"`
	Session   string  `json:"session"`
	Platform  string  `json:"platform"`
}

type Rollout struct {
	Account   string  `json:"account"`
	EventTime string  `json:"eventTime"`
	GameHall  string  `json:"gamehall"`
	GameCode  string  `json:"gamecode"`
	RoundID   string  `json:"roundid"`
	Amount    float64 `json:"amount"`
	Mtcode    string  `json:"mtcode"`
	Session   string  `json:"session"`
}

type Rollin struct {
	Account    string  `json:"account"`
	EventTime  string  `json:"eventTime"`
	GameHall   string  `json:"gamehall"`
	GameCode   string  `json:"gamecode"`
	RoundID    string  `json:"roundid"`
	ValidBet   float64 `json:"validbet"`
	Bet        float64 `json:"bet"`
	Win        float64 `json:"win"`
	Roomfee    float64 `json:"roomfee"`
	Amount     float64 `json:"amount"`
	Mtcode     string  `json:"mtcode"`
	CreateTime string  `json:"createTime"`
	Rake       float64 `json:"rake"`
	GameType   string  `json:"gametype"`
}

type TakeAll struct {
	Account   string `json:"account"`
	EventTime string `json:"eventTime"`
	GameHall  string `json:"gamehall"`
	GameCode  string `json:"gamecode"`
	RoundID   string `json:"roundid"`
	Mtcode    string `json:"mtcode"`
	Session   string `json:"session"`
}

type EndRoundData struct {
	Mtcode    string  `json:"mtcode"`
	Amount    float64 `json:"amount"`
	EventTime string  `json:"eventtime"`
	ValidBet  float64 `json:"validbet"`
}

type EndRound struct {
	Account             string         `json:"account"`
	GameHall            string         `json:"gamehall"`
	GameCode            string         `json:"gamecode"`
	RoundID             string         `json:"roundid"`
	Data                []EndRoundData `json:"data"`
	CreateTime          string         `json:"createTime"`
	FreeTime            float64        `json:"freegame"`
	Jackpot             float64        `json:"jackpot"`
	Jackpotcontribution float64        `json:"jackpotcontribution"`
}

type WinsEvent struct {
	MTCode    string
	Amount    float64
	ValidBet  float64
	RoundID   string
	EventTime string
	GameCode  string
	GameHall  string
}

type WinsData struct {
	Account   string
	EventTime string
	UCode     string
	Event     []WinsEvent
}

type Wins struct {
	List []WinsData
}

type AmendsEvent struct {
	MTCode    string
	Amount    float64
	ValidBet  float64
	RoundID   string
	EventTime string
	GameCode  string
	Action    string
}

type AmendsData struct {
	Account   string
	Event     []AmendsEvent
	EventTime string
	Amount    float64
	Action    string
	UCode     string
}

type Amends struct {
	List []AmendsData
}
