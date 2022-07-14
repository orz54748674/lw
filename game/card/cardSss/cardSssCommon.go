package cardSss

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
	"vn/common"
	"vn/common/protocol"
	"vn/common/utils"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
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

	Ready       bool //是否准备
	AutoReady   bool //是否自动准备
	NotReadyCnt int  `bson:"NotReadyCnt" json:"NotReadyCnt"` //累计连续不准备次数
	Hosting     bool //托管
	QuitRoom    bool

	StraightType StraightType

	//Poker []int //三道
	PokerVal     []int
	PokerType    []PokerType
	Oolong       bool    //乌龙
	ResultScore  []int64 //每道得分
	FinalScore   int64   //最终得分
	ShotScore    int64   //打枪得分
	HomeRunScore int64   //全垒打的分

	IsHavePeople bool  //是不是有人
	HandPoker    []int //手牌
}

//type PlayerShowPoker struct {
//	First []int
//	Second []int
//	Third []int
//
//	FirstVal int
//	SecondVal int
//	ThirdVal int
//
//	FirstType PokerType
//	SecondType PokerType
//	ThirdType PokerType
//
//	Oolong bool //乌龙
//}
type JiesuanData struct {
	RoomState Room_v `json:"RoomState"`
	CountDown int    `json:"CountDown"`

	ShooterList []int //打枪者
	ShotList    []int //被打枪者
	HomeRun     int   //全垒打
	//NotComp   bool

	PlayerInfo map[int]PlayerInfo
}
type PlayerInfo struct {
	StraightType StraightType

	Poker        []int //三道
	PokerType    []PokerType
	Oolong       bool    //乌龙
	ResultScore  []int64 //每道得分
	FinalScore   int64   //最终得分
	ShotScore    int64   //打枪的分
	HomeRunScore int64   //全垒打得分
	TotalBackYxb int64   //返回
}

var BaseScoreList = []int64{
	100, 500, 1000, 2000, 5000, 10000, 20000, 50000, 100000, 200000, 500000, 1000000,
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
	ROOM_WAITING_START     Room_v = 1
	ROOM_WAITING_READY     Room_v = 2 //准备
	ROOM_WAITING_XIAZHU    Room_v = 3 //下注阶段
	ROOM_END               Room_v = 4
	ROOM_WAITING_ENTER     Room_v = 5
	ROOM_WAITING_JIESUAN   Room_v = 6 //结算
	ROOM_WAITING_RESTART   Room_v = 7
	ROOM_WAITING_CLEAR     Room_v = 8
	ROOM_WAITING_SHOWPOKER Room_v = 9 //亮牌
)

type HallType string

const (
	All  HallType = "ALL"
	FOUR HallType = "FOUR"
	TWO  HallType = "TWO"
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
		GameType: game.CardSss,
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
	for _, v := range this.PlayerList {
		if v.UserID == uid {
			return true
		}
	}
	return false
}
func (this *MyTable) SetPlayerHosting(uid string, flag bool) {
	for k, v := range this.PlayerList {
		if v.UserID == uid {
			this.PlayerList[k].Hosting = flag
			break
		}
	}
}
func (this *MyTable) GetPlayerIdx(userID string) int { //获取玩家Idx
	for k, v := range this.PlayerList {
		if v.UserID == userID {
			return k
		}
	}
	return -1
}
func (this *MyTable) GetReadyPlayerNum() int {
	num := 0
	for _, v := range this.PlayerList {
		if v.IsHavePeople && v.Ready {
			num++
		}
	}
	return num
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
func (this *MyTable) GetRobotNum() int {
	num := 0
	for _, v := range this.PlayerList {
		if v.IsHavePeople && v.Role == ROBOT {
			num++
		}
	}
	return num
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
	tableInfo["ShowTime"] = this.GameConf.ShowPokerTime
	playerInfo := make(map[int]interface{})

	for k, v := range this.PlayerList {
		if k != idx {
			if v.IsHavePeople {
				info := make(map[string]interface{})
				game := make(map[string]interface{})
				game["Ready"] = v.Ready

				playerData := make(map[string]interface{})
				if (this.RoomState == ROOM_WAITING_SHOWPOKER || this.RoomState == ROOM_WAITING_JIESUAN) && this.PlayerList[idx].Ready && v.Ready {
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
				//if this.WaitingList[k]{
				//	game["StraightType"] = this.PlayerList[idx].StraightType
				//	game["Poker"] = this.PlayerList[idx].HandPoker
				//}else{
				//	game["StraightType"] = StraightType(0)
				//	game["Poker"] = nil
				//}
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
	game["StraightType"] = this.PlayerList[idx].StraightType
	if (this.RoomState == ROOM_WAITING_SHOWPOKER || this.RoomState == ROOM_WAITING_JIESUAN) && this.PlayerList[idx].Ready {
		waitingList, _ := this.WaitingList.Load(idx)
		game["ShowPoker"] = waitingList
		game["Poker"] = this.PlayerList[idx].HandPoker
	}
	if userID == this.Master && this.RoomState == ROOM_WAITING_READY {
		if this.GetReadyPlayerNum() >= 2 {
			game["CanStartGame"] = true
		} else {
			game["CanStartGame"] = false
		}
	}

	selfInfo["Info"] = info
	selfInfo["Game"] = game

	res["TableInfo"] = tableInfo
	res["PlayerInfo"] = playerInfo
	res["SelfInfo"] = selfInfo
	if this.RoomState == ROOM_WAITING_JIESUAN && this.PlayerList[idx].Ready {
		res["JieSuanData"] = this.JieSuanData
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
				if (this.RoomState == ROOM_WAITING_SHOWPOKER || this.RoomState == ROOM_WAITING_JIESUAN) && this.PlayerList[idx].Ready && v.Ready {
					info["UserID"] = v.UserID
					info["Head"] = v.Head
					info["Name"] = v.Name
					info["Yxb"] = v.Yxb
					waitingList, _ := this.WaitingList.Load(k)
					game["ShowPoker"] = waitingList

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
	for i := 0; i < 13; i++ {
		for k, v := range this.PlayerList {
			if v.Ready {
				this.PlayerList[k].HandPoker = append(this.PlayerList[k].HandPoker, pool[len(pool)-1])
				pool = append(pool[:len(pool)-1])
			}
		}
	}

	if this.BaseScore > 100 && this.GetRealReadyPlayerNum() > 0 {
		this.ControlResults()
	}

	//this.PlayerList[0].HandPoker = []int{36,55,24,66,35,20,69,70,74,59,28,45,49}
	//this.PlayerList[0].HandPoker = []int{0x3b,0x2c,0x31,0x15,0x45,0x23,0x26,0x3a,0x43,0x44,0x48,0x49,0x4c}
	//this.PlayerList[3].HandPoker = []int{0x19,0x2a,0x37,0x38,0x21,0x12,0x23,0x31,0x15,0x24,0x2b,0x4c,0x2d}
	//this.PlayerList[2].HandPoker = []int{0x19,0x2a,0x37,0x38,0x21,0x12,0x23,0x31,0x15,0x24,0x2b,0x4c,0x2d}

	//for k,v := range this.PlayerList{
	//	if v.Ready{
	//		//log.Info("----------------------------deal pooker--idx = %d----------------------- ",k)
	//		res,_ := json.Marshal(v.HandPoker)
	//		log.Info("10进制 idx = %d deal poker %s tableId = %s",k,res,this.tableID)
	//		//pk := ""
	//		//x := fmt.Sprintf("%x",v.HandPoker)
	//		//pk += x
	//		//log.Info("16进制 deal poker %s",pk)
	//
	//		//poker := this.SortPokerFunc(v.UserID,v.HandPoker)
	//		//
	//		//pk = ""
	//		//for _,v1 := range poker{
	//		//	x := fmt.Sprintf("%x ",v1)
	//		//	pk += x
	//		//}
	//		//log.Info(" poker %s",pk)
	//	}
	//
	//}
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
	//time.Sleep(time.Millisecond * 500)
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
func (this *MyTable) CopyMap(src map[int]int) map[int]int {
	dst := map[int]int{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
func (this *MyTable) SortPoker(uid string) []int {
	idx := this.GetPlayerIdx(uid)
	if this.RoomState != ROOM_WAITING_SHOWPOKER {
		log.Info("-------Err room state--- %s", this.RoomState)
		return nil
	}
	if idx < 0 {
		return nil
	}
	if !this.PlayerList[idx].Ready {
		return nil
	}
	return this.SortPokerFunc(uid, this.PlayerList[idx].HandPoker)
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
func (this *MyTable) GetTableRealPlayerNum() int {
	playerNum := 0
	for _, v := range this.PlayerList {
		if v.IsHavePeople && v.Role != ROBOT {
			playerNum++
		}
	}
	return playerNum
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
func (this *MyTable) GetThreeKindPoker(pk []int) [][]int { //获取三道
	if len(pk) != 13 {
		return nil
	}
	res := make([][]int, 0)
	res = append(res, []int{pk[0], pk[1], pk[2]})
	res = append(res, []int{pk[3], pk[4], pk[5], pk[6], pk[7]})
	res = append(res, []int{pk[8], pk[9], pk[10], pk[11], pk[12]})
	return res
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

type PokerTypeData struct {
	StraightType StraightType
	OoLong       bool
	PokerType    []PokerType
}

func (this *MyTable) GetPokerTypeInterface(userID string, pk []int) PokerTypeData {
	idx := this.GetPlayerIdx(userID)
	waitingList, _ := this.WaitingList.Load(idx)
	if this.RoomState == ROOM_WAITING_SHOWPOKER && !waitingList.(bool) {
		this.PlayerList[idx].HandPoker = pk
	}
	pokerType := PokerTypeData{
		StraightType: StraightType(0),
	}
	pokerType.StraightType = this.CheckStraightScore(pk)
	ThreeShowPoker := this.GetThreeKindPoker(pk)
	pokerVal := make([]int, 3)
	pokerType.PokerType = make([]PokerType, 3)
	for k, v := range ThreeShowPoker {
		pokerType.PokerType[k], pokerVal[k] = this.CheckPokerType(v)
	}

	if !(pokerVal[2] > pokerVal[1] && pokerVal[1] > pokerVal[0]) {
		pokerType.OoLong = true
	}

	return pokerType
}
func (this *MyTable) SortShowPokerFunc(showPoker []int, pokerType PokerType) []int {
	sortPoker := make([]int, 0)
	modList := make([]int, len(showPoker))
	mapList := map[int]int{}
	for i := len(showPoker) - 1; i >= 0; i-- {
		modList[i] = showPoker[i] % 0x10
		mapList[modList[i]] += 1
	}
	switch pokerType {
	case FourOfAKind:
		four := make([]int, 0) //
		single := make([]int, 0)
		for k, v := range mapList {
			if v == 4 {
				for _, v1 := range showPoker {
					if v1%0x10 == k {
						four = append(four, v1)
					} else {
						single = append(single, v1)
					}
				}
				break
			}
		}
		for _, v := range four {
			sortPoker = append(sortPoker, v)
		}
		for _, v := range single {
			sortPoker = append(sortPoker, v)
		}
		break
	case FullHouse:
		three := make([]int, 0) //
		two := make([]int, 0)
		for k, v := range mapList {
			if v == 3 {
				for _, v1 := range showPoker {
					if v1%0x10 == k {
						three = append(three, v1)
					}
				}
			} else if v == 2 {
				for _, v1 := range showPoker {
					if v1%0x10 == k {
						two = append(two, v1)
					}
				}
			}
		}
		for _, v := range three {
			sortPoker = append(sortPoker, v)
		}
		for _, v := range two {
			sortPoker = append(sortPoker, v)
		}
		break
	case ThreeOfAKind:
		three := make([]int, 0) //
		single := make([]int, 0)
		for k, v := range mapList {
			if v == 3 {
				for _, v1 := range showPoker {
					if v1%0x10 == k {
						three = append(three, v1)
					} else {
						single = append(single, v1)
					}
				}
				break
			}
		}
		for _, v := range three {
			sortPoker = append(sortPoker, v)
		}
		for _, v := range single {
			sortPoker = append(sortPoker, v)
		}
		break
	case TwoPair:
		two := make([]int, 0) //
		single := make([]int, 0)
		for k, v := range mapList {
			if v == 2 {
				for _, v1 := range showPoker {
					if v1%0x10 == k {
						two = append(two, v1)
					}
				}
			}
		}
		for _, v := range showPoker {
			find := false
			for _, v1 := range two {
				if v1 == v {
					find = true
				}
			}
			if !find {
				single = append(single, v)
				break
			}
		}
		sort.Slice(two, func(i, j int) bool { //升序排序
			if two[i]%0x10 == two[j]%0x10 {
				return two[i] < two[j]
			}
			return two[i]%0x10 < two[j]%0x10
		})
		for _, v := range two {
			sortPoker = append(sortPoker, v)
		}
		for _, v := range single {
			sortPoker = append(sortPoker, v)
		}
		break
	case Pair:
		two := make([]int, 0) //
		single := make([]int, 0)
		for k, v := range mapList {
			if v == 2 {
				for _, v1 := range showPoker {
					if v1%0x10 == k {
						two = append(two, v1)
					} else {
						single = append(single, v1)
					}
				}
				break
			}
		}
		for _, v := range two {
			sortPoker = append(sortPoker, v)
		}
		for _, v := range single {
			sortPoker = append(sortPoker, v)
		}
		break
	}

	if len(sortPoker) == 0 {
		return showPoker
	}
	return sortPoker
}
func (this *MyTable) SortShowPoker(threeShowPoker [][]int, pokerType []PokerType) []int {
	sortPoker := make([]int, 0)

	sort.Slice(threeShowPoker[0], func(i, j int) bool { //升序排序
		modI := threeShowPoker[0][i] % 0x10
		modJ := threeShowPoker[0][j] % 0x10
		if modI == 1 {
			modI += 13
		}
		if modJ == 1 {
			modJ += 13
		}
		if modI == modJ {
			return threeShowPoker[0][i] < threeShowPoker[0][j]
		}
		return modI < modJ
	})
	sort.Slice(threeShowPoker[1], func(i, j int) bool { //升序排序
		modI := threeShowPoker[1][i] % 0x10
		modJ := threeShowPoker[1][j] % 0x10
		if modI == 1 {
			modI += 13
		}
		if modJ == 1 {
			modJ += 13
		}
		if modI == modJ {
			return threeShowPoker[1][i] < threeShowPoker[1][j]
		}
		return modI < modJ
	})
	sort.Slice(threeShowPoker[2], func(i, j int) bool { //升序排序
		modI := threeShowPoker[2][i] % 0x10
		modJ := threeShowPoker[2][j] % 0x10
		if modI == 1 {
			modI += 13
		}
		if modJ == 1 {
			modJ += 13
		}
		if modI == modJ {
			return threeShowPoker[2][i] < threeShowPoker[2][j]
		}
		return modI < modJ
	})

	pk1 := this.SortShowPokerFunc(threeShowPoker[0], pokerType[0])
	for _, v := range pk1 {
		sortPoker = append(sortPoker, v)
	}
	pk2 := this.SortShowPokerFunc(threeShowPoker[1], pokerType[1])
	for _, v := range pk2 {
		sortPoker = append(sortPoker, v)
	}
	pk3 := this.SortShowPokerFunc(threeShowPoker[2], pokerType[2])
	for _, v := range pk3 {
		sortPoker = append(sortPoker, v)
	}
	return sortPoker
}

type CardNum struct {
	size int
	pk   []int
	mod  int
}

func (this *MyTable) CalCardNum(pk []int) []CardNum {
	tmp := make([]CardNum, 0)
	for _, v := range pk {
		mod := v % 0x10
		find := false
		for k1, v1 := range tmp {
			if v1.mod == mod {
				tmp[k1].size++
				tmp[k1].pk = append(tmp[k1].pk, v)
				find = true
				break
			}
		}
		if !find {
			tmp = append(tmp, CardNum{
				size: 1,
				pk:   []int{v},
				mod:  mod,
			})
		}
	}
	return tmp
}

type CompList struct {
	maxPk int
	num   int
	List  []int
}

func (this *MyTable) CompStraight(pk []int) (int, []int) { //顺子
	list := make([]int, len(pk))
	copy(list, pk)
	tmp := this.CalCardNum(list)
	tlist := make([]struct {
		k    int
		card []int
	}, 0)
	for _, v := range tmp {
		if v.mod%0x10 != 0x0f && v.size >= 1 && len(v.pk) >= 1 {
			tlist = append(tlist, struct {
				k    int
				card []int
			}{k: v.mod, card: v.pk})
		}
	}

	sort.Slice(tlist, func(i, j int) bool { //升序排序
		return tlist[i].k < tlist[j].k
	})
	s := tlist[0].k
	i := 1
	modRes := make([]int, 0)
	modRes = append(modRes, s)
	for i < len(tlist) {
		s1 := tlist[i].k
		if i == s1-s {
			modRes = append(modRes, s1)
			i += 1
		} else {
			break
		}
	}

	return i, modRes
}
func (this *MyTable) SortStraight3(mapList map[int]int, sanDao bool, cnt int, sortMod []int) (bool, []int) {
	if cnt <= 0 {
		return true, sortMod
	}
	if !sanDao {
		aFlag := false
		start := 0
		lenS := 0
		for i := 2; i < 15; i++ {
			if mapList[i] > 0 {
				if start == 0 {
					if i == 2 && mapList[14] > 0 {
						mapList[14] -= 1
						lenS += 1
						aFlag = true
					}
					start = i
				}
				lenS += 1
				mapList[i] -= 1
			} else if start > 0 {
				break
			}
			if lenS == 3 {
				pos := start + lenS
				mod := make([]int, len(sortMod))
				copy(mod, sortMod)
				if aFlag {
					pos -= 1
					mod = append(mod, 1)
				}
				for i := start; i < pos; i++ {
					mod = append([]int{i}, mod...)
				}
				ret, mod := this.SortStraight3(mapList, true, cnt-3, mod)
				if ret {
					sortMod = mod
					return true, sortMod
				}
			}

			if lenS == 4 && aFlag {
				aFlag = false
				lenS -= 1
				mapList[14] += 1

				pos := start + lenS
				mod := make([]int, len(sortMod))
				copy(mod, sortMod)
				for i := start; i < pos; i++ {
					mod = append([]int{i}, mod...)
				}
				ret, modN := this.SortStraight3(mapList, true, cnt-3, mod)
				if ret {
					sortMod = modN
					return true, sortMod
				}
			}
		}
		if aFlag {
			mapList[14] += 1
			lenS -= 1
		}
		if start > 0 {
			pos := start + lenS
			for i := start; i < pos; i++ {
				mapList[i] += 1
			}
		}
	}
	aFlag := false
	start := 0
	lenS := 0
	for i := 2; i < 15; i++ {
		if mapList[i] > 0 {
			if start == 0 {
				if i == 2 && mapList[14] > 0 {
					mapList[14] -= 1
					lenS += 1
					aFlag = true
				}
				start = i
			}
			lenS += 1
			mapList[i] -= 1
		} else if start > 0 {
			break
		}

		if lenS == 5 {
			pos := start + lenS
			mod := make([]int, len(sortMod))
			copy(mod, sortMod)
			if aFlag {
				pos -= 1
				mod = append(mod, 1)
			}
			for i := start; i < pos; i++ {
				mod = append(mod, i)
			}
			ret, modN := this.SortStraight3(mapList, sanDao, cnt-5, mod)
			if ret {
				sortMod = modN
				return true, sortMod
			}
		}

		if lenS == 6 && aFlag {
			aFlag = false
			lenS -= 1
			mapList[14] += 1

			pos := start + lenS
			mod := make([]int, len(sortMod))
			copy(mod, sortMod)
			for i := start; i < pos; i++ {
				mod = append(mod, i)
			}
			ret, modN := this.SortStraight3(mapList, sanDao, cnt-5, mod)
			if ret {
				sortMod = modN
				return true, sortMod
			}
		}
	}
	if aFlag {
		mapList[14] += 1
		lenS -= 1
	}
	if start > 0 {
		pos := start + lenS
		for i := start; i < pos; i++ {
			mapList[i] += 1
		}
	}
	return false, sortMod
}
func (this *MyTable) SortStraightPoker(idx int) {
	handPk := make([]int, len(this.PlayerList[idx].HandPoker))
	copy(handPk, this.PlayerList[idx].HandPoker)
	modList := make([]int, len(this.PlayerList[idx].HandPoker))
	color := make([]int, len(this.PlayerList[idx].HandPoker))
	mapList := map[int]int{}
	sortPk := make([]int, 0)
	sort.Slice(handPk, func(i, j int) bool { //升序排序
		modI := handPk[i] % 0x10
		modJ := handPk[j] % 0x10
		if modI == 1 {
			modI += 13
		}
		if modJ == 1 {
			modJ += 13
		}
		if modI == modJ {
			return handPk[i] < handPk[j]
		}
		return modI < modJ
	})
	for i := len(handPk) - 1; i >= 0; i-- {
		modList[i] = handPk[i] % 0x10
		color[i] = handPk[i] / 0x10
		mapList[modList[i]] += 1
	}
	if this.PlayerList[idx].StraightType == QingLong ||
		this.PlayerList[idx].StraightType == YiTiaoLong ||
		this.PlayerList[idx].StraightType == Pair6 ||
		this.PlayerList[idx].StraightType == SameColor {
		sortPk = handPk
	} else if this.PlayerList[idx].StraightType == Pair5Three1 {
		three := make([]int, 0)
		for k, v := range mapList {
			if v == 3 {
				for _, v1 := range handPk {
					if k == v1%0x10 {
						three = append(three, v1)
					}
				}
				break
			}
		}
		two := this.FindNotInArrayList(three, handPk)
		for _, v := range three {
			sortPk = append(sortPk, v)
		}
		for _, v := range two {
			sortPk = append(sortPk, v)
		}
	} else if this.PlayerList[idx].StraightType == Flush3 || this.PlayerList[idx].StraightType == StraightFlush3 {
		colorNum := map[int]int{
			1: 0,
			2: 0,
			3: 0,
			4: 0,
		}
		for i := 0; i < 13; i++ {
			colorNum[color[i]]++
		}

		for k, v := range colorNum {
			if v == 3 || v == 8 {
				i := 0
				for _, v1 := range handPk {
					if v1/0x10 == k {
						sortPk = append(sortPk, v1)
						i++
						if i >= 3 {
							break
						}
					}
				}
				break
			}
		}
		remainPk := this.FindNotInArrayList(sortPk, handPk)
		sort.Slice(remainPk, func(i, j int) bool { //升序排序
			return remainPk[i] < remainPk[j]
		})
		for _, v := range remainPk {
			sortPk = append(sortPk, v)
		}
	} else if this.PlayerList[idx].StraightType == Straight3 {
		for k, v := range mapList {
			if k == 1 {
				mapList[k] -= v
				mapList[k+13] += v
			}
		}
		//for k,v := range modList{
		//	if v == 1{
		//		modList[k] += 13
		//	}
		//}
		sortMod := make([]int, 0)
		flag, ret := this.SortStraight3(mapList, false, 13, sortMod)
		if flag {
			for _, v := range ret {
				if v == 14 {
					v -= 13
				}
				remainPk := this.FindNotInArrayList(sortPk, handPk)
				for _, v1 := range remainPk {
					if v1%0x10 == v {
						sortPk = append(sortPk, v1)
						break
					}
				}
			}
		} else {
			sortPk = handPk
		}
	}

	this.PlayerList[idx].HandPoker = sortPk
}
