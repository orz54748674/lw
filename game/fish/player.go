package fish

import (
	"sync"
	"time"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/storage/fishStorage"
	"vn/storage/userStorage"
)

type DrillCannon struct {
	KillTime int
	State    bool
}

type FuDaiMsg struct {
	Status bool
}

type LeiTingMsg struct {
	Status      bool
	BulletCount int
	BulletGolds int64
	LastSecond  int
}

type DianZuanMsg struct {
	Status     bool
	LastSecond int
}

type Player struct {
	room.BasePlayerImp
	UserID             string
	Seat               int8
	Golds              int64
	Head               string
	Name               string
	CannonType         int
	CannonGolds        int64
	FireAmount         int64
	TotalBet           int64
	Score              int64
	Drill              DrillCannon
	LaserBulletCount   int64
	SpecialFishType    int
	SpecialCannonGolds int64
	FuDai              FuDaiMsg
	LeiTing            LeiTingMsg
	DianZuan           DianZuanMsg
	bLeiShe            bool
	bZhaDan            bool
	bShanDian          bool
	bLunZhou           bool
	fireTimes          int
	isBlock            bool
	scoreWriteLock     sync.Mutex
	goldsWriteLock     sync.Mutex
	UserType           int8
	Account            string
}

func NewPlayer(info map[string]interface{}, tableType int) *Player {
	this := new(Player)
	this.UserID = info["userID"].(string)
	this.Seat = info["seat"].(int8)
	this.Golds = info["golds"].(int64)
	this.Head = info["head"].(string)
	this.Name = info["name"].(string)
	this.CannonType = info["cannonType"].(int)
	this.CannonGolds = info["cannonGolds"].(int64)
	this.Account = info["account"].(string)
	this.FireAmount = 0
	this.TotalBet = 0
	this.Score = 0
	playerConf := fishStorage.GetFishPlayerConf(this.UserID, this.Account)
	this.isBlock = playerConf.IsBlock
	this.Drill.KillTime = 0
	this.Drill.State = false
	this.fireTimes = this.GetFireTimes(tableType)
	user := userStorage.QueryUserId(utils.ConvertOID(this.UserID))
	this.UserType = user.Type
	go func() {
		for {
			time.Sleep(10 * time.Second)
			playerConf = fishStorage.GetFishPlayerConf(this.UserID, this.Account)
			this.isBlock = playerConf.IsBlock
		}
	}()
	return this
}

func (this *Player) IsDrillShooting() bool {
	if !this.Drill.State && this.Drill.KillTime > 0 {
		return true
	}
	return false
}

func (this *Player) GetFireTimes(tableType int) int {
	return fishStorage.GetFireTimes(this.UserID, tableType)
}

func (this *Player) UpsertFireTimes(tableType int) {
	fishStorage.UpsertFireTimes(this.UserID, tableType, this.fireTimes)
}

func (this *Player) UpdateScore(num int64) {
	this.scoreWriteLock.Lock()
	this.scoreWriteLock.Unlock()
	this.Score += num
}

func (this *Player) UpdateGolds(num int64) {
	this.goldsWriteLock.Lock()
	this.goldsWriteLock.Unlock()
	this.Golds += num
}
