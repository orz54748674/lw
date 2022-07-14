package http

import (
	"net/http"
	"net/url"
	"strconv"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/storage/dataStorage"
)

type DataController struct {
	BaseController
}

func (s *DataController) start(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	p := r.Form
	start := dataStorage.DataStart{}
	start.Uuid = getParam(p, "uuid")
	start.UuidWeb = getParam(p, "uuid_web")
	start.Channel = getParam(p, "channel")
	start.UserAgent = r.UserAgent()
	start.Platform = getParam(p, "platform")
	start.Brand = getParam(p, "brand")
	start.Model = getParam(p, "model")
	start.SystemVersion = getParam(p, "system_version")
	start.Language = getParam(p, "language")
	start.CellularProvider = getParam(p, "cellular_provider")
	start.IsRoot, _ = strconv.ParseInt(getParam(p, "is_root"), 0, 64)
	start.CreateAt = utils.Now()
	start.Ip = utils.GetIP(r)
	common.ExecQueueFunc(func() {
		ipInfo := dataStorage.IpInfo{Ip: start.Ip}
		if err := ipInfo.RequestIpInfo(); err == nil {
			ipInfo.Save()
		}
	})

	if start.Uuid == "undefined" || start.Uuid == "" {
		log.Error("uuid is empty.")
		return
	}
	start.Save()
	s.response(w, errCode.Success(nil))
}

func getParam(param url.Values, key string) string {
	if v, ok := param[key]; ok {
		return v[0]
	}
	return ""
}
