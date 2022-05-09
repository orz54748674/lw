package fish

import (
	"encoding/json"
	"math/rand"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/game"
	"vn/game/fish/fishConf"
	"vn/storage/fishStorage"
	"vn/storage/userStorage"
	"vn/storage/walletStorage"
)

func (s *Table) DealProtocolFormat(in interface{}, action string, error *common.Err) []byte {
	info := struct {
		Data     interface{}
		GameType game.Type
		Action   string
		ErrMsg   string
		Code     int
	}{
		Data:     in,
		GameType: game.Fish,
		Action:   action,
	}
	if error == nil {
		info.Code = 0
		info.ErrMsg = "操作成功"
	} else {
		info.Code = error.Code
		info.ErrMsg = error.SetKey().ErrMsg
	}

	ret, _ := json.Marshal(info)
	return ret
}

func (s *Table) sendPackToAll(topic string, in interface{}, action string, err *common.Err) {
	body := s.DealProtocolFormat(in, action, err)
	if tmpErr := s.NotifyCallBackMsgNR(topic, body); tmpErr != nil {
		log.Error("sendPackToAll err:%s, topic:%s, action:%s", tmpErr.Error(), topic, action)
	}
}

func (s *Table) sendPack(session string, topic string, in interface{}, action string, err *common.Err) {
	body := s.DealProtocolFormat(in, action, err)
	if tmpErr := s.SendCallBackMsgNR([]string{session}, topic, body); tmpErr != nil {
		log.Error("sendPack err:%s, topic:%s, action:%s", tmpErr.Error(), topic, action)
	}
}

//玩家加入桌子
func (s *Table) SitDown(session gate.Session, tableType int) {
	if tableType != 0 {
		s.tableType = tableType
		s.cannonGolds = s.fishConf.CannonConf[s.tableType]
	}

	userID := session.GetUserID()
	wallet := walletStorage.QueryWallet(utils.ConvertOID(userID))
	user := userStorage.QueryUserId(utils.ConvertOID(userID))

	tmpIndex := rand.Intn(len(s.seatArr))
	tmpInfo := map[string]interface{}{
		"seat":        s.seatArr[tmpIndex],
		"userID":      userID,
		"golds":       wallet.VndBalance,
		"head":        user.Avatar,
		"name":        user.NickName,
		"account":     user.Account,
		"cannonType":  1,
		"cannonGolds": s.cannonGolds[0],
	}
	s.seatArr = append(s.seatArr[:tmpIndex], s.seatArr[tmpIndex+1:]...)

	player := NewPlayer(tmpInfo, tableType)
	player.Bind(session)
	player.OnRequest(session)
	s.Players[userID] = player

	ret := make(map[string]interface{})
	ret["uid"] = player.UserID
	ret["name"] = player.Name
	ret["headId"] = player.Head
	ret["seat"] = player.Seat
	ret["golds"] = player.Golds
	ret["cannonType"] = player.CannonType
	ret["cannonGolds"] = player.CannonGolds
	s.sendPackToAll(game.Push, ret, protocol.Enter, nil)

	return
}

func (s *Table) PlayerFire(session gate.Session, msg map[string]interface{}) error {
	userID := session.GetUserID()
	param1, ok1 := msg["offset"].(float64)
	if !ok1 {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.PlayerFire, error)
		return nil
	}
	offset := param1

	param2, ok2 := msg["fishID"].(float64)
	if !ok2 {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.PlayerFire, error)
		return nil
	}
	fishID := int32(param2)

	param3, ok := msg["cannonType"].(float64)
	if !ok {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.PlayerFire, error)
		return nil
	}
	cannonType := int(param3)

	var pl = s.Players[userID]
	if &pl == nil {
		error := errCode.NotInRoomError
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.PlayerFire, error)
		return nil
	}

	//钻头炮
	if cannonType == 101 {
		if pl.DianZuan.Status {
			go func() {
				for {
					time.Sleep(time.Second)
					pl.DianZuan.LastSecond -= 1
					if pl.DianZuan.LastSecond == 0 {
						s.NoticeChangeCannon(userID)
						pl.SpecialFishType = 0
						break
					}
				}
			}()
		} else {
			s.sendPack(session.GetSessionID(), game.Push, "", protocol.PlayerFire, errCode.Illegal)
			return nil
		}
		//	雷霆火炮
	} else if cannonType == 105 {
		if pl.LeiTing.Status {
			pl.LeiTing.BulletCount -= 1
			pl.FireAmount = pl.FireAmount + pl.LeiTing.BulletGolds
			if pl.LeiTing.BulletCount == 0 {
				pl.LeiTing.Status = false
				s.NoticeChangeCannon(userID)
			}
		} else {
			s.sendPack(session.GetSessionID(), game.Push, "", protocol.PlayerFire, errCode.Illegal)
			return nil
		}
	} else {
		if pl.Golds-pl.CannonGolds < 0 {
			s.sendPack(session.GetSessionID(), game.Push, "", protocol.PlayerFire, errCode.BalanceNotEnough)
			return nil
		}
		pl.UpdateGolds(-pl.CannonGolds)
		pl.UpdateScore(-pl.CannonGolds)
		pl.FireAmount = pl.FireAmount + pl.CannonGolds
		pl.TotalBet = pl.TotalBet + pl.CannonGolds
		pl.fireTimes = pl.fireTimes + 1
	}

	info := struct {
		UserID      string  `json:"uid"`
		Offset      float64 `json:"offset"`
		PlayerGolds int64   `json:"playerGolds"`
		FishID      int32   `json:"fishID"`
		BulletGolds int64   `json:"bulletGolds"`
		BulletType  int     `json:"bulletType"`
	}{
		UserID:      userID,
		Offset:      offset,
		PlayerGolds: pl.Golds,
		FishID:      fishID,
		BulletGolds: pl.CannonGolds,
		BulletType:  cannonType,
	}

	s.sendPackToAll(game.Push, info, protocol.PlayerFire, nil)

	return nil
}

func (s *Table) HandleKillFish(uid string, fishType fishConf.FishType, bulletGolds int64) (bool, int64) {
	tmpOdds := RandInt64(int64(fishType.RewardMin), int64(fishType.RewardMax))
	rewardGolds := bulletGolds * tmpOdds
	if fishType.ID >= FuDaiID {
		rewardGolds = rewardGolds / 10 * 8
	}
	pl := s.Players[uid]
	var resFlg bool

	randNum := rand.Int63n(10000)
	randPro := 10000 / tmpOdds

	rate := 10000
	for _, v := range s.fishConf.RateByFireArr {
		if pl.fireTimes >= v.FireMin && pl.fireTimes <= v.FireMax {
			rate = rate + int(v.Rate*10000)
		}
	}

	var effectTable fishStorage.EffectBetRate
	var tableRate float64
	if s.tableType == 1 {
		tableRate = s.fishConf.RateRoom1
		effectTable = s.fishConf.EffectBetRoom1
	}
	if s.tableType == 2 {
		tableRate = s.fishConf.RateRoom2
		effectTable = s.fishConf.EffectBetRoom2
	}
	if s.tableType == 3 {
		tableRate = s.fishConf.RateRoom3
		effectTable = s.fishConf.EffectBetRoom3
	}
	rate = rate + int(tableRate*10000)

	if float64(s.botBalance)/float64(s.effectBet) < effectTable.MinValue {
		rate = rate + int(effectTable.MinRate*10000)
	} else if float64(s.botBalance)/float64(s.effectBet) > effectTable.MaxValue {
		rate = rate + int(effectTable.MaxRate*10000)
	}

	if pl.isBlock {
		rate = rate + int(s.fishConf.BlockRate*10000)
	}

	randPro = randPro * int64(rate) / 10000
	if randNum <= randPro {
		resFlg = true
	}

	return resFlg, rewardGolds
}

//通知玩家换回普通炮
func (s *Table) NoticeChangeCannon(uid string) {
	var pl = s.Players[uid]
	if pl != nil {
		info := struct {
			UserID     string `json:"uid"`
			CannonType int    `json:"cannonType"`
		}{
			UserID:     uid,
			CannonType: pl.CannonType,
		}
		s.sendPackToAll(game.Push, info, protocol.ChangeCannon, nil)
	}
}

func (s *Table) KillFish(session gate.Session, msg map[string]interface{}) error {
	userID := session.GetUserID()
	var pl = s.Players[userID]
	if pl == nil {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.KillFish, errCode.NotInRoomError)
		return nil
	}

	param1, ok1 := msg["golds"].(float64)
	if !ok1 {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.KillFish, errCode.ErrParams)
		return nil
	}

	golds := int64(param1)
	if golds <= 0 {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.KillFish, errCode.ErrParams)
		return nil
	}

	param2, ok2 := msg["fishID"].(float64)
	if !ok2 {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.KillFish, errCode.ErrParams)
		return nil
	}
	fishID := int(param2)

	param3, ok3 := msg["bulletType"].(float64)
	if !ok3 {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.KillFish, errCode.ErrParams)
		return nil
	}
	bulletType := int(param3)

	selectFish, bContain := s.findFish(fishID)
	if !bContain {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.KillFish, errCode.NotInRoomError)
		return nil
	}

	//if pl.SpecialFishType > 0 && selectFish.FishType > 200 {
	//	s.sendPack(session.GetSessionID(), game.Push, "", protocol.KillFish, errCode.Illegal)
	//	return nil
	//}

	var fishType fishConf.FishType
	fishType = fishConf.FishTypeConf[selectFish.FishType]
	if bulletType != 101 {
		if pl.FireAmount-golds < 0 {
			s.sendPack(session.GetSessionID(), game.Push, "", protocol.KillFish, errCode.NotInRoomError)
			return nil
		}
		pl.FireAmount = pl.FireAmount - golds
	}

	resFlg, rewardGolds := s.HandleKillFish(userID, fishType, golds)
	if resFlg {
		if selectFish.FishType != LongWangID {
			s.deleteFish(selectFish.FishID)
		}
		tmpRewardGolds := rewardGolds
		if fishType.ID > 200 && fishType.ID != LongWangID {
			pl.SpecialFishType = fishType.ID
			pl.SpecialCannonGolds = pl.CannonGolds
			rewardGolds = 0
		} else {
			pl.UpdateGolds(rewardGolds)
			pl.UpdateScore(rewardGolds)
		}

		info := struct {
			UserID             string                 `json:"uid"`
			FishID             int                    `json:"fishID"`
			FType              int                    `json:"fType"`
			RewardGolds        int64                  `json:"rewardGolds"`
			PlayerGolds        int64                  `json:"playerGolds"`
			RewardNum          int                    `json:"rewardNum"`
			LeiTingBulletCount int                    `json:"leiTingBulletCount"`
			FuDaiRewardArr     []int64                `json:"fuDaiRewardArr"`
			LunZhou            fishStorage.LunZhouMsg `json:"lunZhou"`
		}{
			UserID:      userID,
			FishID:      fishID,
			FType:       fishType.FType,
			RewardGolds: rewardGolds,
			PlayerGolds: pl.Golds,
			RewardNum:   int(rewardGolds / golds),
		}

		if fishType.ID == LeiSheID {
			pl.bLeiShe = true
		}
		if fishType.ID == DianZuanID {
			pl.DianZuan.Status = true
			pl.DianZuan.LastSecond = 5
		}
		if fishType.ID == ZhaDanID {
			pl.bZhaDan = true
		}
		if fishType.ID == LunZhouID {
			info.LunZhou.FirstArr = []int{30, 30, 40, 50}
			info.LunZhou.SecondArr = []int{60, 70, 80, 90}
			info.LunZhou.ThirdArr = []int{100, 200, 300}
			info.LunZhou.SelectReward = s.fishConf.LunZhouRewardArr[rand.Intn(len(s.fishConf.LunZhouRewardArr))]
			info.RewardGolds = int64(info.LunZhou.SelectReward) * pl.CannonGolds
			info.RewardNum = info.LunZhou.SelectReward

			pl.bLunZhou = true
			pl.UpdateGolds(info.RewardGolds)
			pl.UpdateScore(info.RewardGolds)
			info.PlayerGolds = pl.Golds
			go func() {
				time.Sleep(5 * time.Second)
				pl.bLunZhou = false
				pl.SpecialFishType = 0
			}()
		}
		if fishType.ID == ShanDianID {
			pl.bShanDian = true
		}
		if fishType.ID == LeiTingID {
			pl.LeiTing.Status = true
			pl.LeiTing.BulletCount = RandInt(1, 10) * 10
			pl.LeiTing.LastSecond = 120
			info.LeiTingBulletCount = pl.LeiTing.BulletCount
			go func() {
				for {
					time.Sleep(1 * time.Second)
					pl.LeiTing.LastSecond -= 1
					if pl.LeiTing.LastSecond == 0 {
						pl.LeiTing.Status = false
						s.NoticeChangeCannon(pl.UserID)
						pl.SpecialFishType = 0
						break
					}
				}
			}()
		}
		if fishType.ID == FuDaiID {
			var rewardOdds []int64
			for i := 0; i < 5; i++ {
				rewardOdds = append(rewardOdds, int64(RandInt(fishType.RewardMin, fishType.RewardMax))*pl.CannonGolds)
				if i == 0 {
					rewardOdds[0] = tmpRewardGolds
				}
			}
			info.FuDaiRewardArr = rewardOdds
			info.RewardGolds = info.FuDaiRewardArr[0]
			info.PlayerGolds = info.PlayerGolds + info.RewardGolds
			info.RewardNum = int(info.RewardGolds / pl.CannonGolds)

			pl.FuDai.Status = true
			pl.UpdateGolds(info.RewardGolds)
			pl.UpdateScore(info.RewardGolds)
			go func() {
				time.Sleep(5 * time.Second)
				pl.FuDai.Status = false
				pl.SpecialFishType = 0
			}()
		}
		s.sendPackToAll(game.Push, info, protocol.KillFish, nil)
	}

	return nil
}

//用技能击中鱼
func (s *Table) SpecialKillFish(session gate.Session, msg map[string]interface{}) error {
	uid := session.GetUserID()
	var pl = s.Players[uid]
	if pl == nil {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, errCode.NotInRoomError)
		return nil
	}
	pl.SpecialFishType = 0

	param1, ok := msg["fishType"].(float64)
	if !ok {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, errCode.ErrParams)
		return nil
	}
	specialFishType := int(param1)

	if (specialFishType == LeiSheID && !pl.bLeiShe) || (specialFishType == ZhaDanID && !pl.bZhaDan) || (specialFishType == ShanDianID && !pl.bShanDian) {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, errCode.ErrParams)
		return nil
	}
	if specialFishType == LeiSheID {
		pl.bLeiShe = false
		s.NoticeChangeCannon(uid)
	}
	if specialFishType == ZhaDanID {
		pl.bZhaDan = false
	}
	if specialFishType == ShanDianID {
		pl.bShanDian = false
	}

	posX, ok := msg["fishPosX"].(float64)
	if !ok {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, errCode.ErrParams)
		return nil
	}
	posY, ok := msg["fishPosY"].(float64)
	if !ok {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, errCode.ErrParams)
		return nil
	}

	fishType := fishConf.FishTypeConf[specialFishType]
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	rewardNum := utils.RandInt64(int64(fishType.RewardMin), int64(fishType.RewardMax), r)
	var playerReward int64

	type fishRewardInfo struct {
		Uid         string `json:"uid"`
		FishID      int    `json:"fishID"`
		FType       int    `json:"fType"`
		RewardGolds int64  `json:"rewardGolds"`
		RewardNum   int    `json:"rewardNum"`
	}
	info := struct {
		Uid         string           //玩家Uid
		CannonType  int              //炮台类型
		RewardInfos []fishRewardInfo //击杀的鱼信息
		TotalReward int64            //总的奖励金额
		FishType    int
		FishPosX    float64
		FishPosY    float64
		PlayerGolds int64
	}{
		Uid:         uid,
		CannonType:  pl.CannonType,
		RewardInfos: []fishRewardInfo{},
		FishType:    specialFishType,
		FishPosX:    posX,
		FishPosY:    posY,
	}

	fishIDArr, ok := msg["fishIDArr"].([]interface{})
	for _, v := range fishIDArr {
		fishID := int(v.(float64))
		fishItem, bContain := s.findFish(fishID)
		tmpFishType := fishConf.FishTypeConf[fishItem.FishType]
		if !bContain {
			continue
		} else if fishItem.FishType > 200 {
			continue
		}

		if rewardNum > 0 {
			rate := rand.Intn(10000)
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			tmpRand := int(utils.RandInt64(int64(tmpFishType.KillProMin), int64(tmpFishType.KillProMax), r))
			if rate <= tmpRand {
				tmpReward := RandInt64(int64(tmpFishType.RewardMin), int64(tmpFishType.RewardMax))
				rewardNum = rewardNum - tmpReward
				playerReward = playerReward + tmpReward
				pl.UpdateGolds(tmpReward * pl.SpecialCannonGolds)
				pl.UpdateScore(tmpReward * pl.SpecialCannonGolds)
				tmpInfo := fishRewardInfo{
					Uid:         uid,
					FishID:      fishID,
					FType:       fishItem.FishType,
					RewardGolds: tmpReward * pl.SpecialCannonGolds,
					RewardNum:   int(tmpReward),
				}
				info.RewardInfos = append(info.RewardInfos, tmpInfo)

				s.deleteFish(fishID)
			}
		}
	}
	info.TotalReward = playerReward * pl.SpecialCannonGolds
	info.PlayerGolds = pl.Golds
	s.sendPackToAll(game.Push, info, protocol.SpecialKillFish, nil)

	return nil
}

func (s *Table) GetFishByID(fishID int) (bool, fishStorage.Fish) {
	containFishID := false
	var selectFish fishStorage.Fish
	for _, v := range s.AllFish {
		if v.FishID == fishID {
			containFishID = true
			selectFish = v
			break
		}
	}
	return containFishID, selectFish
}

func (s *Table) ChangeCannon(session gate.Session, msg map[string]interface{}) error {
	userID := session.GetUserID()
	param1, ok := msg["cannonType"].(float64)
	if !ok {
		error := errCode.ErrParams
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		return nil
	}
	cannonType := int(param1)

	var pl = s.Players[userID]
	if pl == nil {
		error := errCode.NotInRoomError
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.XiaZhu, error)
		return nil
	}

	pl.CannonType = cannonType
	pl.CannonGolds = s.cannonGolds[cannonType-1]

	info := struct {
		UserID     string `json:"uid"`
		CannonType int    `json:"cannonType"`
	}{
		UserID:     userID,
		CannonType: cannonType,
	}
	s.sendPackToAll(game.Push, info, protocol.ChangeCannon, nil)
	return nil
}

func (s *Table) PlayerLeave(session gate.Session, msg map[string]interface{}) error {
	userID := session.GetUserID()
	s.PlayerDisconnect(userID)

	info := struct {
		UserID string `json:"uid"`
	}{
		UserID: userID,
	}
	s.sendPackToAll(game.Push, info, protocol.PlayerLeave, nil)

	return nil
}

func (s *Table) ChangeSeat(session gate.Session, msg map[string]interface{}) error {
	uid := session.GetUserID()
	pl := s.Players[uid]
	if pl == nil {
		error := errCode.NotInRoomError
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.ChangeSeat, error)
		return nil
	}

	param, ok := msg["seat"].(float64)
	if !ok {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.ChangeSeat, errCode.ErrParams)
		return nil
	}
	seat := int8(param)
	if seat == pl.Seat {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.ChangeSeat, errCode.ErrParams)
		return nil
	}

	bHave := false
	idx := 0
	for k, v := range s.seatArr {
		if v == seat {
			bHave = true
			idx = k
			break
		}
	}
	if !bHave {
		s.sendPack(session.GetSessionID(), game.Push, "", protocol.ChangeSeat, errCode.ErrParams)
		return nil
	}

	beforeSeat := pl.Seat
	s.seatArr[idx] = pl.Seat
	pl.Seat = seat

	info := struct {
		Uid        string `json:"Uid"`
		BeforeSeat int8   `json:"BeforeSeat"`
		AfterSeat  int8   `json:"AfterSeat"`
	}{
		Uid:        uid,
		BeforeSeat: beforeSeat,
		AfterSeat:  seat,
	}

	s.sendPackToAll(game.Push, info, protocol.ChangeSeat, nil)
	return nil
}
