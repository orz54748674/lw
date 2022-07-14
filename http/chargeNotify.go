package http

import (
	"net/http"
	"strings"
	"vn/game/pay/payWay"
)

type chargeNotify struct {
	vgPay     *payWay.VgPay
	official  *payWay.Official
	NapTuDong *payWay.NapTuDong
}

func (s *chargeNotify) init() {
	s.vgPay = &payWay.VgPay{}
	s.official = &payWay.Official{}
	s.NapTuDong = &payWay.NapTuDong{}
}

func (s *chargeNotify) Dispatch(w http.ResponseWriter, r *http.Request) {
	action := strings.Replace(r.URL.Path, "/charge/", "", 1)
	switch action {
	case "vgpay":
		s.vgPay.NotifyCharge(w, r)
	case "naptudong":
		s.NapTuDong.NotifyCharge(w, r)
	case "official":
		s.official.NotifyCharge(w, r)
	default:
		_, _ = w.Write([]byte("404"))
	}
}
