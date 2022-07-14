package slotDance

import (
	"vn/common/protocol"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/game"
	vGate "vn/gate"
	"vn/storage/userStorage"
)

func (this *MyTable) Enter(session gate.Session, msg map[string]interface{}) (err error) {
	player := &room.BasePlayerImp{}
	player.Bind(session)

	player.OnRequest(session)
	userID := session.GetUserID()
	if !this.BroadCast {
		this.BroadCast = true
	}
	if userID == "" {
		log.Info("your userid is empty")
		return nil
	}
	this.Players[userID] = player
	this.UserID = userID
	user := userStorage.QueryUserId(utils.ConvertOID(userID))
	if user.Type == userStorage.TypeNormal {
		this.Role = USER
	}
	this.Name = user.NickName

	tableInfoRet := this.GetTableInfo()
	this.ModeType = NORMAL
	_ = this.sendPack(session.GetSessionID(), game.Push, tableInfoRet, protocol.Enter, nil)
	//end := time.Now().UnixNano()
	//log.Info("cost time = %d",time.Duration(end -start) / time.Millisecond)
	return nil
}
func (this *MyTable) QuitTable(session gate.Session) (err error) {
	userID := session.GetUserID()
	sb := vGate.QuerySessionBean(userID)
	//if this.IsInFreeGame(){
	//	if sb != nil {
	//		this.sendPack(session.GetSessionID(), game.Push, "", protocol.QuitTable, errCode.XiaZhuCantQuit)
	//	}
	//	return nil
	//}
	ret := this.DealProtocolFormat("", protocol.QuitTable, nil)
	this.onlinePush.SendCallBackMsgNR([]string{sb.SessionId}, game.Push, ret)
	this.onlinePush.ExecuteCallBackMsg(this.Trace())
	if !this.IsInCheckout && (!this.IsInFreeGame() || this.ModeType == TRIAL) {
		this.PutQueue(protocol.ClearTable)
	}
	return nil
}
