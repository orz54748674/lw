package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fatih/structs"
	"vn/framework/mqant/log"
)

type Err struct {
	Code   int
	ErrMsg string
	errKey string
	Data   interface{}
	Action string
}

func (s *Err) Init()  {
	s.errKey = s.ErrMsg
}
func (s *Err) GetErr(args ...interface{}) string {
	s.ErrMsg = s.getErrMsg(args)
	return s.ErrMsg
}
func (s *Err) GetError(args ...interface{}) error {
	s.ErrMsg = s.getErrMsg(args)
	return errors.New(fmt.Sprintf(s.ErrMsg))
}
func (s *Err) getErrMsg(args ...interface{}) string {
	msg := ""
	switch len(args) {
	case 1:
		msg = I18str(s.errKey, args[0])
	case 2:
		msg = I18str(s.errKey, args[0], args[1])
	case 3:
		msg = I18str(s.errKey, args[0], args[1], args[2])
	case 4:
		msg = I18str(s.errKey, args[0], args[1], args[2], args[3])
	}
	return msg
}

func (s *Err) SetData(data interface{}) *Err {
	s.Data = data
	return s
}
func (s *Err) SetAction(action string) *Err {
	s.Action = action
	return s
}
func (s *Err) SetErr(err string) *Err {
	s.ErrMsg = err
	return s
}
func (s *Err) SetErrCode(err *Err) *Err {
	s.ErrMsg = err.ErrMsg
	s.Code = err.Code
	s.Data = err.Data
	return s
}
func (s *Err) SetKey(args ...interface{}) *Err {
	s.ErrMsg = s.getErrMsg(args)
	return s
}
func (s *Err) String() string {
	byte, _ := s.Json()
	return string(byte)
}
func (s *Err) Json() ([]byte, error) {
	if s.Data == nil {
		s.Data = map[string]string{}
	}
	return json.Marshal(s)
}
func (s *Err) JsonStr() string {
	if s.Data == nil {
		s.Data = map[string]string{}
	}
	b, err := json.Marshal(s)
	if err != nil {
		log.Error("response json format is err: %s", err.Error())
	}
	return string(b)
}
func (s *Err) GetMap() map[string]interface{} {
	ss := structs.New(s)
	ss.TagName = "json"
	return ss.Map()
}
func (s *Err) GetI18nMap() map[string]interface{} {
	ss := structs.New(s.SetKey())
	ss.TagName = "json"
	return ss.Map()
}
