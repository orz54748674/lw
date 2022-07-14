package chat

import (
	"math/rand"
	"time"
	"vn/game"
	common2 "vn/game/common"
	"vn/storage/chatStorage"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func (s *Chat) RandInt64(min, max int64) int64 {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return r.Int63n(max-min) + min
}
func (s *Chat) OnTimer() {
	for _, v := range game.ChatBotGame {
		r1 := s.RandInt64(1, 3)
		if r1 == 1 {
			if len(s.botsChat[v]) > 0 {
				botR := common2.RandBotN(1, r)
				msgR := s.RandBotChatN(v, 1)
				if len(botR) > 0 && len(msgR) > 0 {
					bot := botR[0]
					msg := msgR[0]
					s.impl.addGroup(bot.Oid.Hex(), string(v))
					s.impl.sendBot(bot.NickName, bot.Oid.Hex()+time.Now().String(), string(v), msg.Msg)
					s.impl.exitGroup(bot.Oid.Hex(), string(v))
				}
			}
		}
	}
}
func (s *Chat) InitBotsChat() {
	s.botsChat = map[game.Type][]chatStorage.ChatBotMsgList{}
	botsChat := chatStorage.QueryBotsChat()
	for _, v := range botsChat {
		for _, v1 := range game.ChatBotGame {
			if v.GameType == v1 {
				s.botsChat[v1] = append(s.botsChat[v1], v)
			}
		}
	}
}

//func (s *Chat) RandBotN(num int) []botStorage.Bot {
//	bots := make([]botStorage.Bot,0)
//	for num > 0{
//		r := s.RandInt64(1,int64(len(s.bots)) + 1) - 1
//		find := false
//		for _,v := range bots{
//			if v.NickName == s.bots[r].NickName{
//				find = true
//				break
//			}
//		}
//		if !find {
//			bots = append(bots,s.bots[r])
//			num--
//		}
//	}
//	return bots
//}
func (s *Chat) RandBotChatN(gameType game.Type, num int) []chatStorage.ChatBotMsgList {
	botsChat := make([]chatStorage.ChatBotMsgList, 0)
	for num > 0 {
		r := s.RandInt64(1, int64(len(s.botsChat[gameType]))+1) - 1
		find := false
		for _, v := range botsChat {
			if v.Msg == s.botsChat[gameType][r].Msg {
				find = true
				break
			}
		}
		if !find {
			botsChat = append(botsChat, s.botsChat[gameType][r])
			num--
		}
	}
	return botsChat
}
