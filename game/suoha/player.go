package suoha

import (
	"vn/common/utils"
	"vn/storage/userStorage"
)

type RobotMsg struct {
	IsRobot   bool
	GameCount int
}

type RoundMsg struct {
	IsAdd  bool
	CurBet int64
	Flg    bool
}

type Player struct {
	UserID         string
	Golds          int64
	Head           string
	Nickname       string
	TotalBet       int64
	Score          int64
	robotMsg       RobotMsg
	IsOnline       bool
	UserType       int8
	LastChatTime   int64
	IsReady        bool
	SelectShowCard int
	Seat           int
	IsUp           bool
	RoundMsg       RoundMsg
	IsAllIn        bool
	AllInRound     int
	RoundBet       []int64
	HandCards      []int
}

func NewPlayer(info map[string]interface{}) *Player {
	this := new(Player)
	this.UserID = info["userID"].(string)
	this.Golds = info["golds"].(int64)
	this.Head = info["head"].(string)
	this.Nickname = info["name"].(string)
	this.TotalBet = 0
	this.Score = 0
	this.robotMsg.IsRobot = false
	this.IsOnline = true
	if info["robot"] != nil {
		this.robotMsg.IsRobot = true
		this.robotMsg.GameCount = info["gameCount"].(int)
	} else {
		user := userStorage.QueryUserId(utils.ConvertOID(this.UserID))
		this.UserType = user.Type
	}
	return this
}
