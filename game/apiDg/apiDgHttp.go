package apiDg

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"vn/common/utils"
	"vn/framework/mqant/log"
	"vn/storage/apiCmdStorage"
	"vn/storage/apiDgStorage"
	"vn/storage/walletStorage"
)

type DgHttp struct {
}

func (h *DgHttp) CheckParams(r *http.Request, params []string) bool {
	for _, v := range params {
		tmp := r.URL.Query().Get(v)
		if tmp == "" {
			return false
		}
	}
	return true
}

func (h *DgHttp) VerifyToken(w http.ResponseWriter, r *http.Request) {
	type Authenticate struct {
		XMLName    xml.Name `xml:"authenticate"`
		MemberID   string   `xml:"member_id"`
		StatusCode int      `xml:"status_code"`
		Message    string   `xml:"message"`
	}

	var rep Authenticate
	tmpToken := r.URL.Query().Get("token")
	if tmpToken == "" {
		rep.StatusCode = 1
		rep.Message = "Fail"
	} else {
		bExist, account := apiCmdStorage.CheckToken(tmpToken)
		if !bExist {
			rep.StatusCode = 2
			rep.Message = "Fail"
		} else {
			rep.StatusCode = 0
			rep.MemberID = account
			rep.Message = "Success"
		}
	}

	fmt.Println("------VerifyToken--------", rep)
	data, _ := xml.MarshalIndent(&rep, "", "  ")
	//加入XML头
	headerBytes := []byte(xml.Header)
	//拼接XML头和实际XML内容
	xmlData := append(headerBytes, data...)
	if _, err := w.Write(xmlData); err != nil {
		log.Error("response: ioWrite json format err:%s", err.Error())
		return
	}
}

func (h *DgHttp) encryptResponse(w http.ResponseWriter, rep interface{}) {
	repByte, _ := json.Marshal(rep)
	fmt.Println("dg response:", string(repByte))
	if _, err := w.Write(repByte); err != nil {
		log.Error("response: ioWrite json format err:%s", err.Error())
		return
	}
}

func (h *DgHttp) GetBalance(w http.ResponseWriter, r *http.Request) {
	fmt.Println("getd.gdgewgdfsdwrewr")
	type MemberInfo struct {
		UserName string  `json:"username"`
		Balance  float64 `json:"balance"`
	}
	rep := struct {
		CodeId int        `json:"codeId"`
		Token  string     `json:"token"`
		Member MemberInfo `json:"member"`
	}{}

	paramMap := make(map[string]interface{})
	if err := h.parse(r, &paramMap); err != nil {
		rep.CodeId = 1
		h.encryptResponse(w, rep)
		return
	}

	memberMap := paramMap["member"].(map[string]interface{})
	tmpToken := paramMap["token"].(string)
	username := memberMap["username"].(string)
	dgUserInfo, err := apiDgStorage.GetDgUserInfoByUsername(username)
	if err != nil {
		fmt.Println("dg account no exist....", username)
		rep.CodeId = 102
		rep.Token = tmpToken
		rep.Member.UserName = username
		h.encryptResponse(w, rep)
		return
	}

	wallet := walletStorage.QueryWallet(utils.ConvertOID(dgUserInfo.Uid))
	rep.CodeId = 0
	rep.Member.UserName = username
	rep.Member.Balance = float64(wallet.VndBalance)/1000
	rep.Token = tmpToken
	h.encryptResponse(w, rep)
	fmt.Println("getBalance..........", rep)
	return
}

func (h *DgHttp) Transfer(w http.ResponseWriter, r *http.Request) {
	type MemberInfo struct {
		UserName string  `json:"username"`
		Amount float64 `json:"amount"`
		Balance  float64 `json:"balance"`
	}
	rep := struct {
		CodeId int        `json:"codeId"`
		Token  string     `json:"token"`
		Data string `json:"data"`
		Member MemberInfo `json:"member"`
	}{}
	paramMap := make(map[string]interface{})
	if err := h.parse(r, &paramMap); err != nil {
		rep.CodeId = 1
		h.encryptResponse(w, rep)
		return
	}
	tmpToken := paramMap["token"].(string)
	ticketId := paramMap["ticketId"].(float64)
	serialNo := paramMap["data"].(string)
	memberMap := paramMap["member"].(map[string]interface{})
	username := memberMap["username"].(string)
	amount := memberMap["amount"].(float64)

	_, err := apiDgStorage.GetTransferRecordBySerialNo(serialNo)
	if err == nil {
		rep.CodeId = 1
		rep.Token = tmpToken
		rep.Member.UserName = username
		h.encryptResponse(w, rep)
		return
	}

	usrInfo, err := apiDgStorage.GetDgUserInfoByUsername(username)
	if err != nil {
		rep.CodeId = 102
		rep.Token = tmpToken
		rep.Member.UserName = username
		h.encryptResponse(w, rep)
		return
	}
	wallet := walletStorage.QueryWallet(utils.ConvertOID(usrInfo.Uid))
	balance := float64(wallet.VndBalance)/1000
	transferAmount := int64(amount * 1000)
	rep.Token = tmpToken
	rep.Member.UserName = username
	rep.Member.Amount = amount
	rep.Data = serialNo
	if transferAmount < 0 && wallet.VndBalance + transferAmount < 0 {
		rep.CodeId = 1
		rep.Member.Balance = balance
		h.encryptResponse(w, rep)
		return
	}
	billType := walletStorage.TypeIncome
	if transferAmount < 0 {
		billType = walletStorage.TypeExpenses
	}

	bill := walletStorage.NewBill(usrInfo.Uid, billType, walletStorage.EventApiDg, serialNo, transferAmount)
	walletStorage.OperateVndBalanceV1(bill)

	apiDgStorage.InsertTransferRecord(serialNo, strconv.Itoa(int(ticketId)), username, tmpToken, int64(balance * 1000), transferAmount)

	rep.CodeId = 0
	rep.Member.Balance = float64(wallet.VndBalance)/1000
	h.encryptResponse(w, rep)
	return
}

func (h *DgHttp) CheckTransfer(w http.ResponseWriter, r *http.Request) {
	paramMap := make(map[string]interface{})
	rep := struct {
		CodeId int        `json:"codeId"`
		Token  string     `json:"token"`
	}{}
	if err := h.parse(r, &paramMap); err != nil {
		rep.CodeId = 1
		h.encryptResponse(w, rep)
		return
	}

	tmpToken := paramMap["token"].(string)
	serialNo := paramMap["data"].(string)


	_, err := apiDgStorage.GetTransferRecordBySerialNo(serialNo)
	if err != nil {
		rep.CodeId = 98
		rep.Token = tmpToken
	} else {
		rep.CodeId = 0
		rep.Token = tmpToken
	}

	h.encryptResponse(w, rep)
}

func (h *DgHttp) Inform(w http.ResponseWriter, r *http.Request) {
	paramMap := make(map[string]interface{})
	type MemberInfo struct {
		Username string `json:"username"`
		Balance float64 `json:"balance"`
	}
	rep := struct {
		CodeId int        `json:"codeId"`
		Token  string     `json:"token"`
		Data string `json:"data"`
		Member MemberInfo `json:"member"`
	}{}
	if err := h.parse(r, &paramMap); err != nil {
		rep.CodeId = 1
		h.encryptResponse(w, rep)
		return
	}

	tmpToken := paramMap["token"].(string)
	serialNo := paramMap["data"].(string)
	ticketId := paramMap["ticketId"].(float64)
	memberMap := paramMap["member"].(map[string]interface{})
	username := memberMap["username"].(string)
	amount := memberMap["amount"].(float64)

	usrInfo, err := apiDgStorage.GetDgUserInfoByUsername(username)
	if err != nil {
		fmt.Println("inform account not exist....", username)
		rep.CodeId = 102
		rep.Token = tmpToken
		rep.Member.Username = username
		h.encryptResponse(w, rep)
		return
	}
	uid := usrInfo.Uid

	rep.Token = tmpToken
	rep.Data = serialNo
	rep.Member.Username = username
	wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
	balance := float64(wallet.VndBalance)/1000
	rep.Member.Balance = balance
	if amount < 0 {
		_, err := apiDgStorage.GetTransferRecordBySerialNo(serialNo)
		if err != nil {
			rep.CodeId = 0
		} else {
			if err := apiDgStorage.RemoveTransferRecordBySerialNo(serialNo); err != nil {
				rep.CodeId = 98
			} else {
				bill := walletStorage.NewBill(uid, walletStorage.TypeIncome, walletStorage.EventApiDg, serialNo, int64(-amount * 1000))
				walletStorage.OperateVndBalanceV1(bill)
			}

		}
	} else {
		_, err := apiDgStorage.GetTransferRecordBySerialNo(serialNo)
		if err != nil {
			rep.CodeId = 0
			bill := walletStorage.NewBill(uid, walletStorage.TypeIncome, walletStorage.EventApiDg, serialNo, int64(amount * 1000))
			walletStorage.OperateVndBalanceV1(bill)
			wallet := walletStorage.QueryWallet(utils.ConvertOID(uid))
			tmpBalance := wallet.VndBalance + int64(amount * 1000)
			apiDgStorage.InsertTransferRecord(serialNo, strconv.Itoa(int(ticketId)), username, tmpToken, tmpBalance, int64(amount * 1000))
		} else {
			rep.CodeId = 0
		}
	}

	h.encryptResponse(w, rep)
}

func (h *DgHttp) Order(w http.ResponseWriter, r *http.Request) {
	paramMap := make(map[string]interface{})
	type ListInfo struct {
		Username string `json:"username"`
		TicketId float64 `json:"ticketId"`
		Serial string `json:"serial"`
		Amount float64 `json:"amount"`
	}
	rep := struct {
		CodeId int        `json:"codeId"`
		Token  string     `json:"token"`
		TicketId float64 `json:"ticketId"`
		List []ListInfo `json:"list"`
	}{}
	if err := h.parse(r, &paramMap); err != nil {
		rep.CodeId = 1
		h.encryptResponse(w, rep)
		return
	}

	tmpToken := paramMap["token"].(string)
	ticketId := paramMap["ticketId"].(float64)
	strTicketId := strconv.Itoa(int(ticketId))

	transferRecords, err := apiDgStorage.GetTransferRecordByTicketId(strTicketId)
	rep.Token = tmpToken
	rep.TicketId = ticketId
	if err != nil {
		rep.CodeId = 98
		h.encryptResponse(w, rep)
		return
 	} else {
 		rep.CodeId = 0
 		for _, v := range transferRecords {
 			var tmpList ListInfo
 			tmpList.TicketId = ticketId
 			tmpList.Username = v.Username
 			tmpList.Amount = float64(v.Amount)/1000
 			tmpList.Serial = v.SerialNo
 			rep.List = append(rep.List, tmpList)
		}
		h.encryptResponse(w, rep)
 		return
	}
}

func (h *DgHttp) Unsettle(w http.ResponseWriter, r *http.Request) {
	paramMap := make(map[string]interface{})
	type ListInfo struct {
		Username string `json:"username"`
		TicketId float64 `json:"ticketId"`
		Serial string `json:"serial"`
		Amount float64 `json:"amount"`
	}
	rep := struct {
		CodeId int        `json:"codeId"`
		Token  string     `json:"token"`
		List []ListInfo `json:"list"`
	}{}
	if err := h.parse(r, &paramMap); err != nil {
		rep.CodeId = 1
		h.encryptResponse(w, rep)
		return
	}
	tmpToken := paramMap["token"].(string)
	transferRecords, err := apiDgStorage.GetTransferRecordByToken(tmpToken)
	rep.Token = tmpToken
	if err != nil {
		rep.CodeId = 98
		h.encryptResponse(w, rep)
		return
	} else {
		for _, v := range transferRecords {
			var tmpList ListInfo
			tmpList.TicketId, _ = strconv.ParseFloat(v.TicketId, 64)
			tmpList.Username = v.Username
			tmpList.Amount = float64(v.Amount) / 1000
			tmpList.Serial = v.SerialNo
			rep.List = append(rep.List, tmpList)
		}
		h.encryptResponse(w, rep)
		return
	}
}

func (h *DgHttp) parse(r *http.Request, param interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		return fmt.Errorf("Content-Type err")
	}

	if err = json.Unmarshal(body, &param); err != nil {
		return err
	}

	return nil
}
