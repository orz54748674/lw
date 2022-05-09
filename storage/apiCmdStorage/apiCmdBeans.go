package apiCmdStorage

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
	MTCode    string `bson:"Mtcode" json:"mtcode"`
	Amount    int64  `bson:"Amount" json:"amount"`
	RoundID   string `bson:"RoundID" json:"roundid"`
	EventTime string `bson:"EventTime" json:"eventtime"`
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
	_Id      string      `bson:"_id" json:"_id`
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

type UserToken struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Account  string             `bson:"account" json:"account"`
	Token    string             `bson:"token" json:"token"`
	UpdateAt int64              `bson:"updateAt" json:"updateAt"`
}

type AmendData struct {
	MTCode    string
	Amount    int64
	Action    string
	RoundID   string
	EventTime string
	ValidBet  int64
}

type Amend struct {
	Account    string
	GameHall   string
	GameCode   string
	Action     string
	Amount     int64
	CreateTime string
	Data       []AmendData
}

type WinsEvent struct {
	MTCode    string
	Amount    int64
	ValidBet  int64
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
	Amount    int64
	ValidBet  int64
	RoundID   string
	EventTime string
	GameCode  string
	Action    string
}

type AmendsData struct {
	Account   string
	Event     []AmendsEvent
	EventTime string
	Amount    int64
	Action    string
	UCode     string
}

type Amends struct {
	List []AmendsData
}

type TicketDetails struct {
	SourceName        string  `json:"SourceName"`
	ReferenceNo       string  `json:"ReferenceNo"`
	TransactionAmount float64 `json:"TransactionAmount"`
}

type UpdateBalanceMsg struct {
	ActionId      int             `json:"ActionId"`
	MatchID       int64           `json:"MatchID"`
	TicketDetails []TicketDetails `json:"TicketDetails"`
}

type RecordUpdateBalance struct {
	Data       string `bson:"data" json:"data"`
	UpdateTime int64  `bson:"updateTime" json:"updateTime"`
}

type ApiCmdConf struct {
	PartnerKey string `bson:"partnerKey" json:"partnerKey"`
	VersionID  int64  `bson:"versionID" json:"versionID"`
}

type BetRecord struct {
	ID          int64  `bson:"Id" json:"Id"`
	SourceName  string `bson:"SourceName" json:"SourceName"`
	ReferenceNo string `bson:"ReferenceNo" json:"ReferenceNo"`
	SocTransId  int    `bson:"SocTransId" json:"SocTransId"`
	IsFirstHalf bool   `bson:"IsFirstHalf" json:"IsFirstHalf"`
	TransDate   int64  `bson:"TransDate" json:"TransDate"`
	IsHomeGive  bool   `bson:"IsHomeGive" json:"IsHomeGive"`
}

type ReferenceMsg struct {
	Uid         string    `bson:"Uid"`
	ReferenceNo string    `bson:"ReferenceNo"`
	BetAmount   int64     `bson:"BetAmount"`
	CreateAt    time.Time `bson:"CreateAt" json:"CreateAt"`
}

type CmdTeamLeagueInfo struct {
	InfoType int
	InfoID int
	InfoName string
}
