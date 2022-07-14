package cardQzsg

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
	"vn/common"
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant/gate"
	basegate "vn/framework/mqant/gate/base"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/agentStorage"
	"vn/storage/walletStorage"
)

type PlayerList struct {
	Yxb     int64  `bson:"Yxb" json:"Yxb"`         //游戏币
	UserID  string `bson:"UserID" json:"UserID"`   //用户id
	Account string `bson:"Account" json:"Account"` //账号
	Name    string `bson:"Name" json:"Name"`       //用户名
	Head    string `bson:"Head" json:"Head"`       //用户头像
	//Sex int8  `bson:"Sex" json:"Sex"`//用户性别
	Role Role `bson:"Role" json:"Role"` //user 真实用户 robot机器人

	TotalBackYxb int64 `bson:"TotalBackYxb" json:"TotalBackYxb"` //总返回金币

	LastChatTime time.Time //最后发送消息时间
	SysProfit    int64
	BotProfit    int64
	Session      gate.Session

	Ready       bool //是否准备
	AutoReady   bool //是否自动准备
	NotReadyCnt int  `bson:"NotReadyCnt" json:"NotReadyCnt"` //累计连续不准备次数
	Hosting     bool //托管
	QuitRoom    bool

	PokerVal  int
	PokerType PokerType

	BetVal int64

	FinalScore   int64 //最终得分
	PressScore   int64
	IsHavePeople bool  //是不是有人
	HandPoker    []int //手牌
}

type JiesuanData struct {
	RoomState  Room_v `json:"RoomState"`
	CountDown  int    `json:"CountDown"`
	PlayerInfo map[int]PlayerInfo
}
type PlayerInfo struct {
	HandPoker    []int
	TotalBackYxb int64
	PokerType    PokerType
}

var XiaZhuTimesList = []int64{
	10, 15, 20,
}
var BaseScoreList = []int64{
	500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000,
}
var ChipsList = []int64{
	5, 10, 15, 20,
}

type HallConfig struct {
	PlayerNum int   //房间类型 4就是4人房
	BaseScore int64 //底分
	BaseNum   int   //总人数
	MaxOffset int   //最大偏移
	StepNum   int
	CurNum    int //当前人数
}

type Role string

const (
	USER  Role = "user"
	ROBOT Role = "robot"
	Agent Role = "agent"
)

type Room_v uint8

const (
	ROOM_WAITING_START      Room_v = 1
	ROOM_WAITING_READY      Room_v = 2 //准备
	ROOM_WAITING_XIAZHU     Room_v = 3 //下注阶段
	ROOM_END                Room_v = 4
	ROOM_WAITING_ENTER      Room_v = 5
	ROOM_WAITING_JIESUAN    Room_v = 6 //结算
	ROOM_WAITING_RESTART    Room_v = 7
	ROOM_WAITING_CLEAR      Room_v = 8
	ROOM_WAITING_SHOWPOKER  Room_v = 9 //亮牌
	ROOM_WAITING_PUTOP      Room_v = 10
	ROOM_WAITING_GRABDEALER Room_v = 11
)

type HallType string

const (
	All HallType = "ALL"
)

func (this *MyTable) RandInt64(min, max int64) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return this.Rand.Int63n(max-min) + min
}
func (this *MyTable) sendPackToAll(topic string, in interface{}, action string, err *common.Err) error {
	if !this.BroadCast { //广播功能
		return nil
	}
	body := this.DealProtocolFormat(in, action, err)
	//error := this.NotifyCallBackMsgNR(topic,body)
	for _, v := range this.PlayerList {
		if v.IsHavePeople && v.Role != ROBOT {
			sb := vGate.QuerySessionBean(v.UserID)
			if sb != nil {
				s, _ := basegate.NewSession(this.app, sb.Session)
				this.SendCallBackMsgByQueue([]string{s.GetSessionID()}, topic, body)
			}
		}
	}
	return nil
}
func (this *MyTable) sendPack(session string, topic string, in interface{}, action string, err *common.Err) error {
	body := this.DealProtocolFormat(in, action, err)
	error := this.SendCallBackMsgByQueue([]string{session}, topic, body)
	return error
}
func (this *MyTable) DealProtocolFormat(in interface{}, action string, error *common.Err) []byte {
	info := struct {
		Data     interface{}
		GameType game.Type
		Action   string
		ErrMsg   string
		Code     int
	}{
		Data:     in,
		GameType: game.CardQzsg,
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
func (this *MyTable) sendPackToLobby(session string, topic string, in interface{}, action string, err *common.Err) error {
	body := this.DealProtocolFormatToLobby(in, action, err)
	error := this.SendCallBackMsgByQueue([]string{session}, topic, body)
	return error
}
func (this *MyTable) DealProtocolFormatToLobby(in interface{}, action string, error *common.Err) []byte {
	info := struct {
		Data     interface{}
		GameType game.Type
		Action   string
		ErrMsg   string
		Code     int
	}{
		Data:     in,
		GameType: game.Lobby,
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
func (this *MyTable) PlayerIsTable(uid string) bool {
	for k, v := range this.PlayerList {
		if v.UserID == uid {
			this.PlayerList[k].Hosting = false
			return true
		}
	}
	return false
}
func (this *MyTable) GetPlayerIdx(userID string) int { //获取玩家Idx
	for k, v := range this.PlayerList {
		if v.UserID == userID {
			return k
		}
	}
	return -1
}
func (this *MyTable) GetTableInfo(userID string) interface{} {
	idx := this.GetPlayerIdx(userID)
	res := make(map[string]interface{})

	tableInfo := make(map[string]interface{})
	tableInfo["RoomState"] = this.RoomState
	tableInfo["CountDown"] = this.CountDown
	tableInfo["EventID"] = this.EventID
	tableInfo["MasterIdx"] = this.GetPlayerIdx(this.Master)
	tableInfo["BaseScore"] = this.BaseScore
	tableInfo["TableID"] = strings.Split(this.tableID, "_")[0]
	tableInfo["ReadyTime"] = this.GameConf.ReadyTime
	tableInfo["TotalPlayerNum"] = this.TotalPlayerNum
	tableInfo["DealerIdx"] = this.DealerIdx
	tableInfo["GrabDealerTime"] = this.GameConf.QiangZhuangTime - 2
	tableInfo["BetTime"] = this.GameConf.XiaZhuTime - 2

	playerInfo := make(map[int]interface{})

	for k, v := range this.PlayerList {
		if k != idx {
			if v.IsHavePeople {
				info := make(map[string]interface{})
				game := make(map[string]interface{})
				game["Ready"] = v.Ready
				game["GrabDealer"] = nil
				game["XiaZhu"] = nil
				playerData := make(map[string]interface{})
				if (this.RoomState == ROOM_WAITING_GRABDEALER || this.RoomState == ROOM_WAITING_XIAZHU || this.RoomState == ROOM_WAITING_JIESUAN) && this.PlayerList[idx].Ready && v.Ready {
					info["UserID"] = v.UserID
					info["Head"] = v.Head
					info["Name"] = v.Name
					info["Yxb"] = v.Yxb

					playerData["Info"] = info
				} else {
					playerData["Info"] = nil
				}
				if this.RoomState == ROOM_WAITING_GRABDEALER {
					waitingList, _ := this.WaitingList.Load(k)
					if waitingList.(bool) {
						if this.IsContainElement(k, this.GrabDealerList) {
							game["GrabDealer"] = 1
						} else {
							game["GrabDealer"] = -1
						}
					}
				}
				if this.RoomState == ROOM_WAITING_XIAZHU || this.RoomState == ROOM_WAITING_JIESUAN {
					waitingList, _ := this.WaitingList.Load(k)
					if waitingList.(bool) {
						game["XiaZhu"] = v.BetVal
					}
				}
				playerData["Game"] = game

				playerInfo[k] = playerData

			}
		}
	}

	selfInfo := make(map[string]interface{})
	info := make(map[string]interface{})
	game := make(map[string]interface{})
	info["UserID"] = this.PlayerList[idx].UserID
	info["Head"] = this.PlayerList[idx].Head
	info["Name"] = this.PlayerList[idx].Name
	info["Yxb"] = this.PlayerList[idx].Yxb
	info["Idx"] = idx

	game["Ready"] = this.PlayerList[idx].Ready
	game["GrabDealer"] = nil
	game["XiaZhu"] = nil
	chipsList := make([]int64, 0)
	for _, v := range ChipsList {
		chipsList = append(chipsList, v*this.BaseScore)
	}
	game["ChipsList"] = chipsList
	if this.RoomState == ROOM_WAITING_JIESUAN && this.PlayerList[idx].Ready {
		game["Poker"] = this.PlayerList[idx].HandPoker
	}
	if userID == this.Master && this.RoomState == ROOM_WAITING_READY {
		if this.GetReadyPlayerNum() >= 2 {
			game["CanStartGame"] = true
		} else {
			game["CanStartGame"] = false
		}
	}
	if this.RoomState == ROOM_WAITING_GRABDEALER {
		waitingList, _ := this.WaitingList.Load(idx)
		if waitingList.(bool) {
			if this.IsContainElement(idx, this.GrabDealerList) {
				game["GrabDealer"] = 1
			} else {
				game["GrabDealer"] = -1
			}
		}
	}
	if this.RoomState == ROOM_WAITING_XIAZHU || this.RoomState == ROOM_WAITING_JIESUAN {
		waitingList, _ := this.WaitingList.Load(idx)
		if waitingList.(bool) {
			game["XiaZhu"] = this.PlayerList[idx].BetVal
		}
	}

	selfInfo["Info"] = info
	selfInfo["Game"] = game

	res["TableInfo"] = tableInfo
	res["PlayerInfo"] = playerInfo
	res["SelfInfo"] = selfInfo
	if this.RoomState == ROOM_WAITING_JIESUAN && this.PlayerList[idx].Ready {
		res["JieSuanData"] = this.JieSuanData
	} else {
		res["JieSuanData"] = nil
	}

	return res

}
func (this *MyTable) GetPlayerInfo(userID string) interface{} {
	idx := this.GetPlayerIdx(userID)
	res := make(map[string]interface{})

	tableInfo := make(map[string]interface{})
	tableInfo["RoomState"] = this.RoomState
	tableInfo["CountDown"] = this.CountDown
	tableInfo["EventID"] = this.EventID
	tableInfo["MasterIdx"] = this.GetPlayerIdx(this.Master)

	playerInfo := make(map[int]interface{})

	for k, v := range this.PlayerList {
		if k != idx {
			if v.IsHavePeople {
				info := make(map[string]interface{})
				game := make(map[string]interface{})
				game["Ready"] = v.Ready

				playerData := make(map[string]interface{})
				if (this.RoomState == ROOM_WAITING_GRABDEALER || this.RoomState == ROOM_WAITING_XIAZHU || this.RoomState == ROOM_WAITING_JIESUAN) && this.PlayerList[idx].Ready && v.Ready {
					info["UserID"] = v.UserID
					info["Head"] = v.Head
					info["Name"] = v.Name
					info["Yxb"] = v.Yxb
					waitingList, _ := this.WaitingList.Load(k)
					game["ShowPoker"] = waitingList.(bool)

					playerData["Info"] = info
				} else {
					playerData["Info"] = nil
				}
				playerData["Game"] = game

				playerInfo[k] = playerData

			}
		}
	}

	res["TableInfo"] = tableInfo
	res["PlayerInfo"] = playerInfo

	return res
}

func (this *MyTable) Shuffle() []int {
	pool := make([]int, len(card))
	copy(pool, card)
	n := len(pool)
	for i := 0; i < n; i++ {
		idx := this.RandInt64(1, int64(n+1)) - 1
		tmpIdx := this.RandInt64(1, int64(n+1)) - 1
		pool[idx], pool[tmpIdx] = pool[tmpIdx], pool[idx]
	}

	for i := 0; i < n; i++ {
		idx := this.RandInt64(1, int64(n+1)) - 1
		tmpIdx := this.RandInt64(1, int64(n+1)) - 1
		pool[idx], pool[tmpIdx] = pool[tmpIdx], pool[idx]
	}

	for i := 0; i < n; i++ {
		idx := this.RandInt64(1, int64(n+1)) - 1
		tmpIdx := this.RandInt64(1, int64(n+1)) - 1
		pool[idx], pool[tmpIdx] = pool[tmpIdx], pool[idx]
	}

	return pool
}
func (this *MyTable) DealPoker(pool []int) {
	for k, _ := range this.PlayerList {
		this.PlayerList[k].HandPoker = []int{}
	}
	for i := 0; i < 3; i++ {
		for k, v := range this.PlayerList {
			if v.Ready {
				this.PlayerList[k].HandPoker = append(this.PlayerList[k].HandPoker, pool[len(pool)-1])
				pool = append(pool[:len(pool)-1])
			}
		}
	}
	//this.PlayerList[0].HandPoker = []int{0x23,0x33,0x14,0x24,0x15,0x25,0x4e,0x1a,0x46,0x39,0x2a,0x3f,0x2e}
	//this.PlayerList[1].HandPoker = []int{0x16,0x26,0x37,0x17,0x27,0x18,0x28,0x38,0x24,0x14,0x43,0x29,0x13}
	//this.PlayerList[2].HandPoker = []int{0x3a,0x4c,0x3e,0x3b,0x15,0x4b,0x48,0x2c,0x33,0x47,0x28,0x1e,0x36}

	for k, v := range this.PlayerList {
		if v.Ready {
			sort.Slice(this.PlayerList[k].HandPoker, func(i, j int) bool { //升序排序
				handPk := make([]int, len(this.PlayerList[k].HandPoker))
				copy(handPk, this.PlayerList[k].HandPoker)
				if handPk[i]%0x10 == 1 {
					handPk[i] += 13
				}
				if handPk[j]%0x10 == 1 {
					handPk[j] += 13
				}
				if handPk[i]%0x10 == handPk[j]%0x10 {
					return handPk[i] < handPk[j]
				}
				return handPk[i]%0x10 < handPk[j]%0x10
			})

			this.CalcPlayerPokerType()

			//log.Info("----------------------------deal pooker--idx = %d----------------------- ",k)
			//res,_ := json.Marshal(v.HandPoker)
			//log.Info("10进制 deal poker %s",res)
			//pk := ""
			//x := fmt.Sprintf("%x",v.HandPoker)
			//pk += x
			//log.Info("16进制 deal poker %s",pk)
		}

	}
}
func (this *MyTable) SwitchRoomState() interface{} { //切换房间状态
	info := struct {
		RoomState Room_v
		CountDown int
		EventID   string
	}{
		RoomState: this.RoomState,
		CountDown: this.CountDown,
		EventID:   this.EventID,
	}
	_ = this.sendPackToAll(game.Push, info, protocol.SwitchRoomState, nil)
	//time.Sleep(200 * time.Millisecond)
	return info
}
func (this *MyTable) notifyWallet(uid string) {
	sb := vGate.QuerySessionBean(uid)
	if sb == nil {
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	msg := make(map[string]interface{})
	msg["Wallet"] = wallet
	msg["Action"] = "wallet"
	msg["GameType"] = game.All
	b, _ := json.Marshal(msg)
	this.onlinePush.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, b)
	this.onlinePush.ExecuteCallBackMsg(this.Trace())
	agentStorage.OnWalletChange(uid)
}
func (this *MyTable) Cartesian(sets [][]int) [][]int {
	lens := func(i int) int { return len(sets[i]) }
	product := make([][]int, 0)
	for ix := make([]int, len(sets)); ix[0] < lens(0); this.NextIndex(ix, lens) {
		var r []int
		for j, k := range ix {
			r = append(r, sets[j][k])
		}
		product = append(product, r)
	}
	return product
}
func (this *MyTable) SliceOutOfOrder(in []int) []int {
	n := len(in)
	for i := 0; i < n; i++ {
		idx := this.RandInt64(1, int64(n+1)) - 1
		tmpIdx := this.RandInt64(1, int64(n+1)) - 1
		in[idx], in[tmpIdx] = in[tmpIdx], in[idx]
	}
	return in
}
func (this *MyTable) GetRealReadyPlayerNum() int {
	playerNum := 0
	for _, v := range this.PlayerList {
		if v.IsHavePeople && v.Role == USER && v.Ready {
			playerNum++
		}
	}
	return playerNum
}
func (this *MyTable) NextIndex(ix []int, lens func(i int) int) {
	for j := len(ix) - 1; j >= 0; j-- {
		ix[j]++
		if j == 0 || ix[j] < lens(j) {
			return
		}
		ix[j] = 0
	}
}
func (this *MyTable) Cartesian2(sets [][][]int) [][][]int {
	lens := func(i int) int { return len(sets[i]) }
	product := make([][][]int, 0)
	for ix := make([]int, len(sets)); ix[0] < lens(0); this.NextIndex(ix, lens) {
		var r [][]int
		for j, k := range ix {
			r = append(r, sets[j][k])
		}
		product = append(product, r)
	}
	return product
}

func (this *MyTable) CalcPlayerPokerType() {
	for k, v := range this.PlayerList {
		if v.IsHavePeople && v.Ready {
			this.PlayerList[k].PokerType = PokerType((v.HandPoker[0]%0x10 + v.HandPoker[1]%0x10 + v.HandPoker[2]%0x10) % 10)
			if this.PlayerList[k].PokerType == 0 {
				this.PlayerList[k].PokerType = Ten
			}
			maxPk := v.HandPoker[2]
			if maxPk%0x10 == 1 {
				maxPk += 13
			}
			this.PlayerList[k].PokerVal = int(this.PlayerList[k].PokerType)*0xFFFF + (maxPk%0x10)*0xFF + maxPk/0x10
		}
	}
}

func (this *MyTable) Combinations(iterable []int, r int) [][]int {
	var res [][]int
	res = [][]int{}
	pool := iterable
	n := len(pool)

	if r > n {
		return res
	}

	indices := make([]int, r)
	for i := range indices {
		indices[i] = i
	}

	result := make([]int, r)
	result1 := make([]int, r)
	for i, el := range indices {
		result[i] = pool[el]
	}
	copy(result1, result)
	res = append(res, result1)
	for {
		i := r - 1
		result2 := make([]int, r)
		for ; i >= 0 && indices[i] == i+n-r; i -= 1 {
		}

		if i < 0 {
			return res
		}

		indices[i] += 1
		for j := i + 1; j < r; j += 1 {
			indices[j] = indices[j-1] + 1
		}

		for ; i < len(indices); i += 1 {
			result[i] = pool[indices[i]]
		}
		copy(result2, result)
		res = append(res, result2)
	}

}
func (this *MyTable) FindNotInArrayList(child []int, parent []int) []int { //
	res := make([]int, 0)
	for _, v := range parent {
		find := false
		for _, v1 := range child {
			if v1 == v {
				find = true
				break
			}
		}
		if !find {
			res = append(res, v)
		}
	}
	return res
}
func (this *MyTable) IsContainArray(child []int, parent []int) bool { //
	for _, v := range child {
		find := false
		for _, v1 := range parent {
			if v1 == v {
				find = true
				break
			}
		}
		if !find {
			return false
		}
	}
	return true
}
func (this *MyTable) IsContainElement(element int, array []int) bool { //
	for _, v := range array {
		if element == v {
			return true
		}
	}
	return false
}
func (this *MyTable) CopyMap(src map[int]int) map[int]int {
	dst := map[int]int{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
func (this *MyTable) GetTablePlayerNum() int {
	playerNum := 0
	for _, v := range this.PlayerList {
		if v.IsHavePeople {
			playerNum++
		}
	}
	return playerNum
}
func (this *MyTable) GetRobotNum() int {
	num := 0
	for _, v := range this.PlayerList {
		if v.IsHavePeople && v.Role == ROBOT {
			num++
		}
	}
	return num
}
func (this *MyTable) GetReadyPlayerNum() int {
	num := 0
	for _, v := range this.PlayerList {
		if v.Ready {
			num++
		}
	}
	return num
}
func (this *MyTable) GetReadyPlayerIdx() []int {
	res := make([]int, 0)
	for k, v := range this.PlayerList {
		if v.Ready {
			res = append(res, k)
		}
	}
	return res
}
func (this *MyTable) GetTableRealPlayerNum() int {
	playerNum := 0
	for _, v := range this.PlayerList {
		if v.IsHavePeople && v.Role != ROBOT {
			playerNum++
		}
	}
	return playerNum
}
func (this *MyTable) GetPlayerNum() int {
	num := 0
	for _, v := range this.PlayerList {
		if v.IsHavePeople {
			num++
		}
	}
	return num
}
func (this *MyTable) RemoveTblInList(list []int, tbl []int) []int { //
	for _, v := range tbl {
		for k1, v1 := range list {
			if v == v1 {
				list = append(list[:k1], list[k1+1:]...)
				break
			}
		}
	}
	return list
}

func (this *MyTable) SliceRemoveDuplicates(slice []int) []int {
	sort.Ints(slice)
	i := 0
	var j int
	for {
		if i >= len(slice)-1 {
			break
		}

		for j = i + 1; j < len(slice) && slice[i] == slice[j]; j++ {
		}
		slice = append(slice[:i+1], slice[j:]...)
		i++
	}
	return slice
}
func (this *MyTable) GetNextIdx(idx int) int { //
	nextIdx := idx
	if this.GetTablePlayerNum() == 0 {
		return nextIdx
	}
	for true {
		nextIdx++
		if nextIdx >= this.TotalPlayerNum {
			nextIdx -= this.TotalPlayerNum
		}
		if this.PlayerList[nextIdx].IsHavePeople {
			break
		}
		if nextIdx == idx {
			break
		}
	}
	return nextIdx
}
