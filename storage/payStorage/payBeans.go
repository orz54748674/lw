package payStorage

import (
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mqant/log"
)

type PayConf struct {
	Oid            primitive.ObjectID `bson:"_id,omitempty" json:"MethodId"`
	Name           string             `bson:"Name"`
	Merchant       string             `bson:"Merchant"`
	MethodType     string             `bson:"MethodType"`
	Max            int                `bson:"Max"`
	Mini           int                `bson:"Mini"`
	FeePerThousand int                `bson:"FeePerThousand"`
	IsClose        int                `bson:"IsClose"`
	Priority       int                `bson:"Priority"`
	Remark         string             `bson:"Remark"`
	UpdateAt       time.Time          `bson:"UpdateAt"`
}

type CompanyBank struct {
	Oid         primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	BankName    string             `bson:"BtName"`
	AccountName string             `bson:"AccountName"`
	CardNumber  string             `bson:"CardNumber"`
	BankBranch  string             `bson:"BankBranch"`
	Phone       string             `bson:"Phone"`
	IsAuto      int                `bson:"IsAuto"`
	IsClosed    int                `bson:"IsClosed"`
}

var (
	cPayConf = "payConf"
	//cPayActivityConf = "payActivityConf"
	cCompanyBankConf = "companyBankConf"
	cOrder           = "order"
	cCallBackLog     = "callBackLog"
	cOrderTransfer   = "orderTransfer"
	cDouDouBt        = "doudouBt"
	cDouDou          = "doudou"
	cChargeCode      = "chargeCode"
	cChargeFromPhone = "chargeFromPhone"
	cPhoneChargeConf = "phoneChargeConf"
	cUserReceiveBt   = "userReceiveBt"
)

const (
	StatusInit          = 0
	StatusUsed          = 1
	StatusCallBack      = 5
	StatusProcess       = 3
	StatusSuccess       = 9
	DouDouStatusSuccess = 9
	DouDouStatusReject  = 4
	StatusFailed        = 4
)

func newPayConf(name string, merchant string, methodType string,
	max int, mini int, priority int, remark string, feePerThousand int) *PayConf {
	return &PayConf{
		Name:           name,
		Merchant:       merchant,
		MethodType:     methodType,
		Max:            max,
		Mini:           mini,
		IsClose:        0,
		Priority:       priority,
		Remark:         remark,
		FeePerThousand: feePerThousand,
		UpdateAt:       utils.Now(),
	}
}

type Order struct {
	ID        int64              `bson:"-" json:"-"`
	Oid       primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	UserId    primitive.ObjectID `bson:"UserId"`
	MethodId  primitive.ObjectID `bson:"MethodId"`
	Amount    int64              `bson:"Amount"`
	GotAmount int64              `bson:"GotAmount"`
	Status    int                `bson:"Status"`
	NotifyUrl string             `bson:"NotifyUrl"`
	ThirdId   string             `bson:"ThirdId"`
	Fee       int64              `bson:"Fee"`
	Ip        string             `bson:"Ip"`
	Remark    string             `bson:"Remark"`
	//Reason    string             `bson:"Reason"`
	AdminId  uint      `bson:"AdminId"`
	UpdateAt time.Time `bson:"UpdateAt"`
	CreateAt time.Time `bson:"CreateAt"`
}

func (Order) TableName() string {
	return "order"
}

func NewOrder(userId primitive.ObjectID, methodId primitive.ObjectID, amount int64, ip string) *Order {
	var fee int64 = 0
	if payConf := QueryPayConf(methodId); payConf != nil {
		fee = amount * int64(payConf.FeePerThousand) / 1000
	}
	return &Order{
		Oid:      primitive.NewObjectID(),
		UserId:   userId,
		MethodId: methodId,
		Amount:   amount,
		Ip:       ip,
		Fee:      fee,
		UpdateAt: utils.Now(),
		CreateAt: utils.Now(),
	}
}

type CallBackLog struct {
	ID       int64     `bson:"-" json:"-"`
	Type     string    `bson:"Type"` //charge, doudou
	Content  string    `bson:"Content"`
	Merchant string    `bson:"Merchant"`
	ThirdId  string    `bson:"ThirdId"`
	CreateAt time.Time `bson:"CreateAt"`
}

func (CallBackLog) TableName() string {
	return "call_back_log"
}
func NewCallBack(Type string, content string, merchant string, thirdId string) {
	cb := CallBackLog{
		Type:     Type,
		Content:  content,
		Merchant: merchant,
		ThirdId:  thirdId,
		CreateAt: utils.Now(),
	}
	c := common.GetMongoDB().C(cCallBackLog)
	if err := c.Insert(&cb); err != nil {
		log.Error(err.Error())
	}
	common.ExecQueueFunc(func() {
		common.GetMysql().Create(&cb)
	})
}

type OrderTransfer struct {
	ID          int64              `bson:"-" json:"-"`
	OrderId     primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	ReceiveId   primitive.ObjectID `bson:"ReceiveId"`
	AccountName string             `bson:"AccountName"`
	SaveType    string             `bson:"SaveType"`
	Code        string             `bson:"Code"`
	CreateAt    time.Time          `bson:"CreateAt"`
}

func (OrderTransfer) TableName() string {
	return "order_transfer"
}

type DouDouBt struct {
	BtName   string `bson:"BtName"`
	Max      int    `bson:"Max"`
	Mini     int    `bson:"Mini"`
	Priority int    `bson:"Priority"`
	IsClosed int    `bson:"IsClosed"`
}

type DouDou struct {
	ID          int64              `bson:"-" json:"-"`
	Oid         primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	UserId      string             `bson:"UserId"`
	BtName      string             `bson:"BtName"`
	AccountName string             `bson:"AccountName"`
	CardNum     string             `bson:"CardNum"`
	Amount      int64              `bson:"Amount"`
	Status      int                `bson:"Status"`
	Remark      string             `bson:"Remark"`
	AdminId     uint               `bson:"AdminId"`
	Ip          string             `bson:"Ip"`
	UpdateAt    time.Time          `bson:"UpdateAt"`
	CreateAt    time.Time          `bson:"CreateAt"`
}

func (DouDou) TableName() string {
	return "doudou"
}

func NewDouDou(userId, btName, accountName, cardNum, ip string, amount int64) *DouDou {
	return &DouDou{
		Oid:         primitive.NewObjectID(),
		UserId:      userId,
		BtName:      btName,
		AccountName: accountName,
		CardNum:     cardNum,
		Amount:      amount,
		Ip:          ip,
		CreateAt:    utils.Now(),
		UpdateAt:    utils.Now(),
	}
}

type ChargeCode struct {
	Oid      primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Code     string             `bson:"Code"`
	Amount   int                `bson:"Amount"`
	Status   int                `bson:"Status"` // 0 未使用。 1，已使用
	Uid      string             `bson:"Uid"`
	Belong   string             `bson:"Belong"`
	UpdateAt time.Time          `bson:"UpdateAt"`
	CreateAt time.Time          `bson:"CreateAt"`
}

type PhoneCharge struct {
	ID         uint64
	Oid        primitive.ObjectID `bson:"_id,omitempty" json:"Oid"` //orderId
	Seri       string             `bson:"Seri"`                     //序列号
	Password   string             `bson:"Password"`                 //卡密
	Amount     int64              `bson:"Amount"`                   //面值
	RealAmount int64              `bson:"RealAmount"`
	CreateAt   time.Time          `bson:"CreateAt"`
}

func (PhoneCharge) TableName() string {
	return "phone_charge"
}

type PhoneChargeConf struct {
	Name           string    `bson:"Name"`
	FeePerThousand int       `bson:"FeePerThousand"`
	Amount         int       `bson:"Amount"`
	UpdateAt       time.Time `bson:"UpdateAt"`
}

type UserReceiveBt struct {
	ID          int64              `bson:"-" json:"-"`
	Oid         primitive.ObjectID `bson:"_id,omitempty" json:"Oid"` //uid
	BtName      string             `bson:"BtName"`
	AccountName string             `bson:"AccountName"`
	CardNum     string             `bson:"CardNum"`
	CreateAt    time.Time          `bson:"CreateAt"`
}

func (UserReceiveBt) TableName() string {
	return "user_receive_bt"
}

type BalanceChangeLog struct {
	ID           int64     `gorm:"primarykey"`
	AgentAccount string    `gorm:"column:agent_account" form:"agentAccount" json:"agentAccount"`
	Account      string    `gorm:"column:account" form:"account" json:"account"`
	Uid          string    `gorm:"column:uid" form:"uid" json:"uid"`
	UserType     int       `gorm:"column:user_type" form:"userType" json:"userType"`
	Type         int       `gorm:"column:type" form:"opType" json:"opType"`
	Amount       int64     `gorm:"column:amount" form:"amount" json:"amount"`
	Remark       string    `gorm:"column:remark" form:"remark" json:"remark"`
	AdminId      uint      `gorm:"column:admin_id" form:"admin_id" json:"admin_id"`
	CreateAt     time.Time `gorm:"column:create_at" form:"create_at" json:"create_at"`
	VndBalance   int64     `gorm:"vnd_balance" form:"vnd_balance" json:"vndBalance"`
}

func (BalanceChangeLog) TableName() string {
	return "balance_change_log"
}
