package chat

import (
	"encoding/json"
	"vn/common/utils"
	"vn/game"
	"vn/gate"
	"vn/storage/chatStorage"
	"vn/storage/userStorage"
)

type Impl struct {
	push *gate.OnlinePush
}

var (
	actionNewMsg = "newMessage"
)

func (s *Impl) addGroup(uid string, groupId string) {
	chatStorage.AddGroup(uid, groupId)
}
func (s *Impl) exitGroup(uid string, groupId string) {
	chatStorage.ExitGroup(uid, groupId)
}
func (s *Impl) getGroupMsgList(groupId string, size int64) *[]chatStorage.Message {
	msgList := chatStorage.QueryMsgList(groupId, size)
	return msgList
}
func (s *Impl) disconnect(uid string) {
	chatStorage.Disconnect(uid)
}
func (s *Impl) send(uid string, msgId string, groupId string, content string) {
	fromUser := userStorage.QueryUserId(utils.ConvertOID(uid))
	msg := &chatStorage.Message{
		MsgId:    msgId,
		GroupId:  groupId,
		Content:  content,
		FromUid:  uid,
		FromUser: fromUser,
		CreateAt: utils.Now(),
	}
	if msgId == "" {
	} else {
		chatStorage.SaveMsg(msg)
	}
	uids := chatStorage.QueryGroup(groupId)
	userIds := utils.ConvertUidToOid(uids)
	sessionIds := gate.GetSessionIds(userIds)
	notify := make(map[string]interface{}, 4)
	notify["msg"] = msg
	notify["Action"] = actionNewMsg
	s.notify(sessionIds, notify)
}
func (s *Impl) notify(sessionIds []string, notify map[string]interface{}) {
	notify["GameType"] = game.Chat
	body, _ := json.Marshal(notify)
	_ = s.push.SendCallBackMsgNR(sessionIds, game.Push, body)
}
func (s *Impl) sendBot(nickName string, msgId string, groupId string, content string) {
	fromUser := &userStorage.User{NickName: nickName}
	msg := &chatStorage.Message{
		MsgId:    msgId,
		GroupId:  groupId,
		Content:  content,
		FromUid:  "",
		FromUser: *fromUser,
		CreateAt: utils.Now(),
	}
	if msgId == "" {
	} else {
		chatStorage.SaveMsg(msg)
	}
	uids := chatStorage.QueryGroup(groupId)
	userIds := utils.ConvertUidToOid(uids)
	sessionIds := gate.GetSessionIds(userIds)
	notify := make(map[string]interface{}, 4)
	notify["msg"] = msg
	notify["Action"] = actionNewMsg
	s.notify(sessionIds, notify)
}
