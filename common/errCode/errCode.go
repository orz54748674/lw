package errCode

import "vn/common"

var (
	//Success = NewCode(0,"Success").SetKey()
	// ------------  系统错误  -----------
	Forbidden              = errCode(401, "Forbidden")
	AccountChanged         = errCode(302, "AccountChanged")
	Illegal                = errCode(402, "Illegal")
	ServerError            = errCode(500, "ServerError")
	ServerBusy             = errCode(501, "ServerBusy")
	ErrParams              = errCode(10001, "ErrParams")
	FileOverMax            = errCode(10002, "FileOverMax")
	ActionNotFound         = errCode(10003, "ActionNotFound")
	PayMethodIdErr         = errCode(10004, "PayMethodIdErr")
	DoudouAmountErr        = errCode(10005, "DoudouAmountErr")
	ConnectCustomerService = errCode(10006, "ConnectCustomerService")
	AccountNotAllow        = errCode(10007, "AccountNotAllow")

	//------------- 一般功能错误  ------------
	AccountExisted   = errCode(20001, "AccountExisted")
	AccountNotExist  = errCode(20002, "AccountNotExist")
	PasswordErr      = errCode(20003, "PasswordErr")
	PageSizeErr      = errCode(20004, "PageSizeErr")
	SmsSentTooFast   = errCode(20005, "SmsSentTooFast")
	SmsCodeErr       = errCode(20006, "SmsCodeErr")
	PhoneAlreadyBind = errCode(20007, "PhoneAlreadyBind")
	PhoneFormatErr   = errCode(20008, "PhoneFormatErr")
	UserAlreadyBind  = errCode(20009, "UserAlreadyBind")
	BetNotEnough     = errCode(20010, "BetNotEnough")
	AmountNotAllow   = errCode(20011, "AmountNotAllow")
	DataNotFind      = errCode(20012, "DataNotFind")
	WalletPayErr     = errCode(20013, "WalletPayErr")
	ChatGroupNotFind = errCode(20014, "ChatGroupNotFind")
	NotAvailableSeat = errCode(20015, "NotAvailableSeat")

	GameStopBet             = errCode(30001, "GameStopBet")
	DxGameBetErr            = errCode(30002, "dxGameBetErr")
	BalanceNotEnough        = errCode(30003, "BalanceNotEnough")
	XiaZhuCantQuit          = errCode(30004, "XiaZhuCantQuit")
	NameFormatError         = errCode(30005, "NameFormatError")
	PwdFormatError          = errCode(30006, "PwdFormatError")
	CurCanXiaZhuError       = errCode(30008, "CurCanXiaZhuError")
	NotInRoomError          = errCode(30009, "NotInRoomError")
	CantCreateTableError    = errCode(30010, "CantCreateTableError")
	OldPwdError             = errCode(30011, "OldPwdError")
	PwdNotSameError         = errCode(30012, "PwdNotSameError")
	DouDouCountOverLimit    = errCode(30013, "DouDouCountOverLimit")
	ActivityReceiveError    = errCode(30014, "ActivityReceiveError")
	TimeIntervalError       = errCode(30015, "TimeIntervalError")
	ChatYxbLimitError       = errCode(30016, "ChatYxbLimitError")
	AgentBalanceNotEnough   = errCode(30017, "AgentBalanceNotEnough")
	GiftCodeErr             = errCode(30018, "GiftCodeErr")
	GiftCodeUsed            = errCode(30019, "GiftCodeUsed")
	NickNameExistError      = errCode(30020, "NickNameExistError")
	AlreadyBindBankCard     = errCode(30021, "AlreadyBindBankCard")
	NotBindBtCard           = errCode(30022, "NotBindBtCard")
	BtCardAlreadyBind       = errCode(30023, "BtCardAlreadyBind")
	DouDouNumNotAllow       = errCode(30024, "DouDouNumNotAllow")
	UncompleteOrder         = errCode(30025, "UncompleteOrder")
	InvalidPutPoker         = errCode(30026, "InvalidPutPoker")
	InvalidPhomPoker        = errCode(30027, "InvalidPhomPoker")
	InvalidGivePoker        = errCode(30028, "InvalidGivePoker")
	FreeGameCantQuit        = errCode(300029, "FreeGameCantQuit")
	CurGrabDealerError      = errCode(30030, "CurGrabDealerError")
	Order5MinuteOnce        = errCode(30031, "Order5MinuteOnce")
	PlayAccountNotAllow     = errCode(30032, "PlayAccountNotAllow")
	BindBankNotExist        = errCode(30033, "BindBankNotExist")
	NapTuDongError          = errCode(30034, "NapTuDongError")
	ChargeProtectError      = errCode(30035, "ChargeProtectError")
	ActivityNeedChargeError = errCode(30036, "ActivityNeedChargeError")
	DouDouAccountNotAllow   = errCode(30037, "DouDouAccountNotAllow")
	DouDouSameBt            = errCode(30038, "DouDouSameBt")
	DouDouMinAmount         = errCode(30039, "DouDouMinAmount")

	InvalidLotteryCode = errCode(30040, "InvalidLotteryCode")
	LotteryNumberErr   = errCode(30041, "LotteryNumberErr")
	LotteryPlayErr     = errCode(30042, "LotteryPlayErr")
	BetCodeErr         = errCode(30043, "BetCodeErr")
	BetLimit           = errCode(30044, "BetLimit")

	ApiCreateUserErr = errCode(30101, "ApiCreateUserErr")
	ApiErr           = errCode(30102, "ApiErr")
	ApiLoginErr      = errCode(30103, "ApiLoginErr")

	RoomPlayerNumLimit   = errCode(30200, "RoomPlayerNumLimit")
	PointsNotEnough      = errCode(30201, "PointsNotEnough")
	PleaseUnlockSafe     = errCode(30202, "PleaseUnlockSafe")
	PleaseActivationSafe = errCode(30203, "PleaseActivationSafe")
	RegisterLimit        = errCode(30204, "RegisterLimit")
	QuitRoomAfterOver    = errCode(30205, "QuitRoomAfterOver")
	QuitRoomCancel       = errCode(30206, "QuitRoomCancel")
)

func errCode(code int, errMsg string) *common.Err {
	err := &common.Err{Code: code, ErrMsg: errMsg}
	err.Init()
	return err
}
func Success(data interface{}) *common.Err {
	if data == nil {
		data = map[string]string{}
	}
	return errCode(0, "Success").SetKey().SetData(data)
}
func New() *common.Err {
	return Success(nil)
}
