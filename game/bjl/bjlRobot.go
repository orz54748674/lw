package bjl

import (
	"fmt"
	"math/rand"
	"time"
	"vn/common/protocol"
	"vn/game"
	common2 "vn/game/common"
)

func (s *Table) CreateRobot(num int) {
	for {
		if num > 0 {
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			robot := common2.RandBotN(num, r)
			for _, v := range robot {
				find := false
				s.Players.Range(func(k, val interface{}) bool {
					tmp := val.(*Player)
					if v.Oid.Hex() == tmp.UserID {
						find = true
						return false
					}
					return true
				})
				if !find {
					tmpInfo := map[string]interface{}{
						"userID":    v.Oid.Hex(),
						"golds":     rand.Int63n(50000000-1000) + 1000,
						"head":      v.Avatar,
						"name":      v.NickName,
						"robot":     true,
						"gameCount": rand.Intn(10) + 1,
					}
					robotPlayer := NewPlayer(tmpInfo)
					if robotPlayer != nil {
						num -= 1
						s.Players.Store(v.Oid.Hex(), robotPlayer)
					}
					ret := make(map[string]interface{})
					ret["uid"] = tmpInfo["userID"]
					ret["nickName"] = tmpInfo["name"]
					ret["head"] = tmpInfo["head"]
					ret["golds"] = tmpInfo["golds"]
					s.sendPackToAll(game.Push, ret, protocol.Enter, nil)
				}
			}
		} else {
			break
		}
	}
}

func (s *Table) getRobotBetPos() int {
	randNum := rand.Intn(100)
	if randNum <= 38 {
		return 0
	} else if randNum <= 76 {
		return 1
	} else if randNum <= 86 {
		return 2
	} else if randNum <= 93 {
		return 3
	} else {
		return 4
	}

	return 0
}

func (s *Table) getRobotBetCoin() int64 {
	randNum := rand.Intn(100)
	coinIdx := 0
	if randNum <= 44 {
		coinIdx = 0
	} else if randNum <= 64 {
		coinIdx = 1
	} else if randNum <= 84 {
		coinIdx = 2
	} else if randNum <= 90 {
		coinIdx = 3
	} else if randNum <= 93 {
		coinIdx = 4
	} else if randNum <= 95 {
		coinIdx = 5
	} else if randNum <= 96 {
		coinIdx = 6
	} else if randNum <= 97 {
		coinIdx = 7
	} else if randNum <= 98 {
		coinIdx = 8
	} else if randNum <= 99 {
		coinIdx = 0
	}
	return s.betCoins[coinIdx]
}

func (s *Table) robotBet() {
	go func() {
		sec := 0
		for {
			s.Players.Range(func(k, val interface{}) bool {
				tmp := val.(*Player)
				if tmp.robotMsg.IsRobot {
					if rand.Intn(100) >= 80 {
						pos := s.getRobotBetPos()
						if pos == 0 || pos == 1 {
							if tmp.BetInfo[0] > 0 {
								pos = 0
							} else if tmp.BetInfo[1] > 0 {
								pos = 1
							}
						}
						coin := s.getRobotBetCoin()
						if tmp.Golds-coin > 0 {
							err := s.betHandle(tmp.UserID, pos, coin)
							if err != nil {
								fmt.Println("robot bet err", err.ErrMsg)
							}
						}
					}
				}
				return true
			})
			sec += 1
			if sec == 20 {
				return
			}
			time.Sleep(1 * time.Second)
		}

	}()
}

func (s *Table) robotCheckout() {
	s.Players.Range(func(k, val interface{}) bool {
		v := val.(*Player)
		if v.robotMsg.IsRobot {
			v.robotMsg.GameCount -= 1
		}
		return true
	})
}

func (s *Table) robotEnter() {
	for {
		tmp := time.Duration(rand.Int63n(10000))
		time.Sleep(tmp * time.Millisecond)
		var lenNum int
		s.Players.Range(func(k, val interface{}) bool {
			lenNum = lenNum + 1
			return true
		})
		if lenNum < 100 {
			s.CreateRobot(1)
		}
	}
}

func (s *Table) robotLeave() {
	for {
		tmp := time.Duration(rand.Int63n(10000))
		time.Sleep(tmp * time.Millisecond)
		s.Players.Range(func(k, val interface{}) bool {
			v := val.(*Player)
			if v.robotMsg.IsRobot && v.robotMsg.GameCount == 0 {
				s.leaveHandle(v.UserID)
			}
			return true
		})
	}
}

func (s *Table) robotEnterAndLeave() {
	go s.robotEnter()
	go s.robotLeave()
}
