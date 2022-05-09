package spadeGame

import (
	"time"
	"vn/framework/mqant/log"
)

type SpadeRpc struct {
}

// token 验证
func (s *SpadeRpc) Authorize(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SpadeRpc authorize time:%v data:%v", time.Now().Unix(), data)
	return
}

// 入口
func (s *SpadeRpc) Enter(data map[string]interface{}) (resp map[string]interface{}, err error) {
	log.Debug("SpadeRpc Enter time:%v data:%v", time.Now().Unix(), data)
	return
}
