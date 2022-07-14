package gameStorage

import (
	"time"
	"vn/framework/mongo-driver/bson/primitive"
)

type MailType string

var (
	MailAll MailType = "All"
	Group   MailType = "group"
	Private MailType = "private"
)

type ReadStatus string

var (
	ReadAll ReadStatus = "All"
	Read    ReadStatus = "read"
	UnRead  ReadStatus = "unread"
)

type Mail struct {
	ID           int64              `bson:"-" json:"-"`
	Oid          primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Account      string             `bson:"Account" json:"Account"`
	Type         MailType           `bson:"Type" json:"Type"`                 //group群发 private私发
	SendTime     time.Time          `bson:"SendTime" json:"SendTime"`         //发送时间
	Title        string             `bson:"Title" json:"Title"`               //标题
	ContentTitle string             `bson:"ContentTitle" json:"ContentTitle"` //内容里的标题
	Content      string             `bson:"Content" json:"Content"`           //内容
}

type MailRecord struct {
	ID           int64              `bson:"-" json:"-"`
	Oid          primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Account      string             `bson:"Account" json:"Account"`
	Type         MailType           `bson:"Type" json:"Type"`                 //group 系统邮件  private 普通邮件
	SendTime     time.Time          `bson:"SendTime" json:"SendTime"`         //发送时间
	Title        string             `bson:"Title" json:"Title"`               //标题
	ContentTitle string             `bson:"ContentTitle" json:"ContentTitle"` //内容里的标题
	Content      string             `bson:"Content" json:"Content"`           //内容
	ReadState    ReadStatus         `bson:"ReadState" json:"ReadState"`       //已读状态 read 未读 unread 已读
}
