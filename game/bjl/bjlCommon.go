package bjl

import (
	"encoding/json"
	"vn/common"
	"vn/game"
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
		GameType: game.Bjl,
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

func (s *Table) sendPackToAll(topic string, in interface{}, action string, err *common.Err) error {
	body := s.DealProtocolFormat(in, action, err)
	error := s.NotifyCallBackMsgNR(topic, body)
	return error
}

func (s *Table) sendPack(session string, topic string, in interface{}, action string, err *common.Err) error {
	body := s.DealProtocolFormat(in, action, err)
	error := s.SendCallBackMsgNR([]string{session}, topic, body)
	return error
}