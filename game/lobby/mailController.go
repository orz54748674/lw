package lobby

import (
	"encoding/json"
	"time"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mqant/gate"
	"vn/game"
	gate2 "vn/gate"
	"vn/storage/gameStorage"
	"vn/storage/lobbyStorage"
	"vn/storage/userStorage"
)

func (self *Lobby)DealInfoFormat() map[string]interface{} {
	res := make(map[string]interface{})
	res["Code"] = 0
	res["Action"] = "HD_info"
	res["ErrMsg"] = "操作成功"
	res["GameType"] = "lobby"
	return res
}
func (self *Lobby) AdminMailSend(session gate.Session,msg map[string]interface{}) (map[string]interface{},error){
	mailType,_:= msg["Type"].(string)
	mail := gameStorage.Mail{
		Type:         gameStorage.MailType(mailType),
		SendTime:     time.Now(),
		Title:        msg["Title"].(string),
		ContentTitle: msg["ContentTitle"].(string),
		Content:      msg["Content"].(string),
		Account:      msg["Account"].(string),
	}
	if mail.Type == "private"{ //私发邮件
		if user := userStorage.QueryUser(bson.M{"Account":mail.Account});user != nil{
			mailRecord := gameStorage.MailRecord{
				Type:         mail.Type,
				SendTime:     mail.SendTime,
				Title:        mail.Title,
				ContentTitle: mail.ContentTitle,
				Content:      mail.Content,
				Account:      mail.Account,
				ReadState:    gameStorage.UnRead,
			}
			gameStorage.InsertMailRecord(&mailRecord)
			sb := gate2.QuerySessionBean(user.Oid.Hex())
			if sb != nil{
				res := self.DealInfoFormat()
				unread := make(map[string]interface{},1)
				num := gameStorage.QueryMailUnreadNum(mail.Account,gameStorage.MailAll)
				unread["mailUnreadNum"] = num
				res["Data"] = unread

				ret,_ := json.Marshal(res)
				self.push.SendCallBackMsgNR([]string{sb.SessionId}, game.Push,ret)
				lobbyStorage.UpsertLobbyBubble(lobbyStorage.LobbyBubble{
					Uid: user.Oid.Hex(),
					BubbleType: lobbyStorage.Mail,
					Num: num,
					UpdateAt: utils.Now(),
				})
			}
		}else{
			return errCode.AccountNotExist.GetI18nMap(),nil
		}
	}else if users := userStorage.QueryUsers();users != nil {
		res := self.DealInfoFormat()
		for _,v := range users{
			mailRecord := gameStorage.MailRecord{
				Type:         mail.Type,
				SendTime:     mail.SendTime,
				Title:        mail.Title,
				ContentTitle: mail.ContentTitle,
				Content:      mail.Content,
				Account:      v.Account,
				ReadState:    gameStorage.UnRead,
			}
			gameStorage.InsertMailRecord(&mailRecord)
			sb := gate2.QuerySessionBean(v.Oid.Hex())
			if sb != nil{
				unread := make(map[string]interface{},1)
				num := gameStorage.QueryMailUnreadNum(v.Account,gameStorage.MailAll)
				unread["mailUnreadNum"] = num
				res["Data"] = unread
				ret,_ := json.Marshal(res)
				self.push.SendCallBackMsgNR([]string{sb.SessionId}, game.Push,ret)
				lobbyStorage.UpsertLobbyBubble(lobbyStorage.LobbyBubble{
					Uid: v.Oid.Hex(),
					BubbleType: lobbyStorage.Mail,
					Num: num,
					UpdateAt: utils.Now(),
				})
			}
		}
	}
	gameStorage.InsertMail(&mail)
	return errCode.Success(nil).GetI18nMap(),nil
}
func (self *Impl) GetMailList(session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
	userID := session.GetUserID()
	user := userStorage.QueryUserId(utils.ConvertOID(userID))
	res := make(map[string]interface{})
	res["SystemUnreadNum"] = gameStorage.QueryMailUnreadNum(user.Account,gameStorage.Group)
	res["PrivateUnreadNum"] = gameStorage.QueryMailUnreadNum(user.Account,gameStorage.Private)
	res["SystemMail"] = gameStorage.QueryMailRecord(user.Account,gameStorage.Group)
	res["PrivateMail"] = gameStorage.QueryMailRecord(user.Account,gameStorage.Private)
	return errCode.Success(res).GetMap(), nil
}
func (self *Lobby) UpdateMailState(session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
	oid := utils.ConvertOID(msg["Oid"].(string))
	uid := session.GetUserID()
	uOid := utils.ConvertOID(uid)
	user := userStorage.QueryUserId(uOid)
	gameStorage.UpdateMailRecordReadState(oid,gameStorage.Read)
	sb := gate2.QuerySessionBean(uOid.Hex())
	if sb != nil{
		res := self.DealInfoFormat()
		unread := make(map[string]interface{},1)
		num := gameStorage.QueryMailUnreadNum(user.Account,gameStorage.MailAll)
		unread["mailUnreadNum"] = num
		res["Data"] = unread

		ret,_ := json.Marshal(res)
		self.push.SendCallBackMsgNR([]string{sb.SessionId}, game.Push,ret)
		lobbyStorage.UpsertLobbyBubble(lobbyStorage.LobbyBubble{
			Uid: user.Oid.Hex(),
			BubbleType: lobbyStorage.Mail,
			Num: num,
			UpdateAt: utils.Now(),
		})
	}
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
}
func (self *Impl) DeleteMail(session gate.Session, msg map[string]interface{}) (map[string]interface{},error){
	oid := utils.ConvertOID(msg["Oid"].(string))
	gameStorage.DeleteMail(oid)
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(),nil
}