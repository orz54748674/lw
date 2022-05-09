package gate

import (
	"errors"
	"github.com/fatih/structs"
	"time"
	"vn/common"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/gate"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
)

type SessionStorage struct {
	gate basegate.Gate
}
type SessionBean struct {
	Oid       primitive.ObjectID `bson:"_id,omitempty" json:"Oid"`
	Session   []byte             `bson:"Session"`
	ServerId  string             `bson:"ServerId"`
	SessionId string             `bson:"SessionId"`
	CreateAt  time.Time          `bson:"CreateAt"`
	UpdateAt  time.Time          `bson:"UpdateAt"`
	PagePath  string             `bson:"PagePath"`
}

var (
	cSession  = "gateSession"
	PageLobby = "lobby"
	PageYXX   = "YXX"
	PageSD    = "SD"
)

/**
  存储用户的Session信息,触发条件
  1. session.Bind(Userid)
  2. session.SetPush(key,value)
  3. session.SetBatch(map[string]string)
*/
func (s *SessionStorage) Storage(session gate.Session) (err error) {
	//log.Info("save session: %s", &session)
	uid := session.GetUserID()
	if uid == "" {
		err = errors.New("save session error: uid is empty")
		log.Error(err.Error())
		return err
	}
	b, _ := session.Serializable()
	sessionBean := &SessionBean{
		Oid:       utils.ConvertOID(uid),
		Session:   b,
		SessionId: session.GetSessionID(),
		ServerId:  session.GetServerID(),
		CreateAt:  utils.Now(),
		UpdateAt:  utils.Now(),
	}
	return s.UpsertSessionBean(sessionBean)
}

/**
  强制删除Session信息,触发条件
  1. 暂无
*/
func (s *SessionStorage) Delete(session gate.Session) (err error) {
	log.Info("Delete session: %s", &session)
	uid := session.GetUserID()
	if uid == "" {
		err = errors.New("delete session error: uid is empty")
		log.Error(err.Error())
		return err
	}
	c := common.GetMongoDB().C(cSession)
	selector := bson.M{"_id": utils.ConvertOID(uid)}
	err = c.Remove(selector)
	if err != nil {
		//log.Error(err.Error())
	}
	return err
}

/**
  获取用户Session信息,触发条件
  1. session.Bind(Userid)
*/
func (s *SessionStorage) Query(uid string) (data []byte, err error) {
	//log.Info("Query uid: %s", uid)
	sessionBean := QuerySessionBean(uid)
	if sessionBean != nil {
		data = sessionBean.Session
	} else {
		err = errors.New("uid is not found")
	}
	return data, err
}

/**
  用户心跳,触发条件
  1. 每一次客户端心跳
  可以用来延长Session信息过期时间
*/
func (s *SessionStorage) Heartbeat(session gate.Session) {
	uid := session.GetUserID()
	if uid == "" {
		return
	}
	sessionBean := QuerySessionBean(uid)
	//log.Info("Heartbeat %s,SessionBean:%s", &session,*sessionBean)
	if sessionBean == nil {
		log.Info("Heartbeat error, uid: %s is not found", uid)
		session.Close()
		return
	} else {
		sessionBean.UpdateAt = utils.Now()
		s.UpsertSessionBean(sessionBean)
	}

}
func (s *SessionStorage) InitMongo(sessionExpireSecond time.Duration) {
	c := common.GetMongoDB().C(cSession)
	key := bsonx.Doc{{Key: "UpdateAt",Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key,options.Index().
		SetExpireAfterSeconds(int32(sessionExpireSecond/time.Second)));err != nil{
		log.Error("create gateSession Index: %s",err)
	}
	log.Info("init gateSession of mongo db")
}

func (s *SessionStorage) UpsertSessionBean(sessionBean *SessionBean) error {
	c := common.GetMongoDB().C(cSession)
	selector := bson.M{"_id": sessionBean.Oid}
	update := structs.Map(sessionBean)
	_, err2 := c.Upsert(selector, update)
	if err2 != nil {
		log.Error("Upsert user login error: %s", err2)
		return err2
	}
	return nil
}
func QuerySessionBean(uid string) *SessionBean {
	c := common.GetMongoDB().C(cSession)
	var sessionBean SessionBean
	oid := utils.ConvertOID(uid)
	err := c.FindId(oid).One(&sessionBean)
	if err != nil {
		return nil
	}
	return &sessionBean
}

func QueryAllSession() *[]SessionBean {
	c := common.GetMongoDB().C(cSession)
	var allSession []SessionBean
	if err := c.Find(bson.M{}).All(&allSession); err != nil {
		log.Error(err.Error())
	}
	return &allSession
}
func QueryServerId(serverId string) *[]SessionBean {
	c := common.GetMongoDB().C(cSession)
	var allSession []SessionBean
	if err := c.Find(bson.M{"ServerId": serverId}).All(&allSession); err != nil {
		log.Error(err.Error())
	}
	return &allSession
}
func QuerySessionId(sessionId string) *SessionBean {
	c := common.GetMongoDB().C(cSession)
	var session SessionBean
	query := bson.M{"SessionId": sessionId}
	if err := c.Find(query).One(&session); err != nil {
		//log.Error(err.Error())
		return nil
	}
	return &session
}
func RemoveAllSession() {
	c := common.GetMongoDB().C(cSession)
	if _, e := c.RemoveAll(bson.M{}); e != nil {
		log.Error(e.Error())
	}
}
func GetSessionIds(uid []primitive.ObjectID) []string {
	c := common.GetMongoDB().C(cSession)
	query := bson.M{"_id": bson.M{"$in": uid}}
	var sessionBeans []SessionBean
	if err := c.Find(query).All(&sessionBeans); err != nil {
		log.Error(err.Error())
	}
	var res []string
	for _, s := range sessionBeans {
		res = append(res, s.SessionId)
	}
	return res
}
func UpdateSessionPage(uid primitive.ObjectID, page string) {
	c := common.GetMongoDB().C(cSession)
	query := bson.M{"_id": uid}
	update := bson.M{"$set": bson.M{"PagePath": page}}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
}
func QuerySessionByPage(page string) *[]SessionBean {
	c := common.GetMongoDB().C(cSession)
	query := bson.M{"PagePath": bson.M{"$regex":page}}
	var sessionBean []SessionBean
	if err := c.Find(query).All(&sessionBean); err != nil {
		log.Error(err.Error())
	}
	return &sessionBean
}
func GetSessionUids(uid []primitive.ObjectID) []string {
	c := common.GetMongoDB().C(cSession)
	query := bson.M{"_id": bson.M{"$in": uid}}
	var sessionBeans []SessionBean
	if err := c.Find(query).All(&sessionBeans); err != nil {
		log.Error(err.Error())
	}
	var res []string
	for _, s := range sessionBeans {
		res = append(res, s.Oid.Hex())
	}
	return res
}