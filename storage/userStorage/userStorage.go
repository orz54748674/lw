package userStorage

import (
	"net/http"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/bson/primitive"
	"vn/framework/mongo-driver/mongo/options"
	"vn/framework/mongo-driver/x/bsonx"
	"vn/framework/mqant/log"
	"vn/storage"
)

var (
	cGlobal   = "global"
	cUser     = "user"
	cLogin    = "userLogin"
	cToken    = "userToken"
	cUserInfo = "userInfo"
)

func NewUserGlobalId() int64 {
	return storage.NewGlobalId(storage.KeyUser)
}

func GetLastUserId() int64 {
	return storage.GetLastId(storage.KeyUser)
}
func InitUserMongo(tokenExpire time.Duration) {
	createUserIndex()
	createTokenIndex(tokenExpire)
	//createLoginIndex()
	_ = common.GetMysql().AutoMigrate(&User{})
	_ = common.GetMysql().AutoMigrate(&Login{})
	_ = common.GetMysql().AutoMigrate(&UserInfo{})
	_ = common.GetMysql().AutoMigrate(&LoginLog{})
}

func createUserIndex() {
	c := common.GetMongoDB().C(cUser)
	key := bsonx.Doc{{Key: "Account", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetUnique(true)); err != nil {
		log.Error("create token Index: %s", err)
	}
	log.Info("init user of mongo db")
}
func createTokenIndex(tokenExpire time.Duration) {
	c := common.GetMongoDB().C(cToken)
	key := bsonx.Doc{{Key: "UpdateAt", Value: bsonx.Int32(1)}}
	if err := c.CreateIndex(key, options.Index().
		SetExpireAfterSeconds(int32(tokenExpire/time.Second))); err != nil {
		log.Error("create token Index: %s", err)
	}
	log.Info("init token of mongo db")
}
func InsertUser(user *User) (*User, *common.Err) {
	find := bson.M{"$or": []bson.M{{"Account": user.Account}, {"NickName": user.Account}}}
	if user := QueryUser(find); user != nil {
		return nil, errCode.AccountExisted.SetKey()
	}
	c := common.GetMongoDB().C(cUser)
	if count, _ := c.FindId(user.Oid).Count(); count > 0 {
		return nil, errCode.ServerBusy.SetKey()
	}
	c = common.GetMongoDB().C(cUser)
	user.ProfitPerThousand = -1
	if err := c.Insert(user); err != nil {
		log.Error("insert user err: %s", err)
		return nil, errCode.AccountExisted.SetKey()
	}
	u := *user
	common.GetMysql().Create(&u)
	//log.Info("new user: %s ", utils.GetStrFromObj(user))
	//common.ExecQueueFunc(func() {
	//
	//})
	return user, nil
}
func UpsertLogin(login *Login) *common.Err {
	//login.LastTime = utils.Now()
	save := *login
	c := common.GetMongoDB().C(cLogin)
	selector := bson.M{"_id": login.Oid}
	update, err := utils.ToMap(login, "json")
	if err != nil {
		log.Error("json to map crash: %s", err)
	}
	_, err2 := c.Upsert(selector, update)
	if err2 != nil {
		log.Error("Upsert user login error: %s", err2)
		return errCode.ServerError.SetErr(err2.Error())
	}
	common.ExecQueueFunc(func() {
		var res Login
		common.GetMysql().Where("oid=?", save.Oid.Hex()).
			First(&res)
		save.ID = res.ID
		common.GetMysql().Save(&save)
	})
	return nil
}
func NewLoginLog(r *http.Request, login Login) {
	ua := r.Header.Get("User-Agent")
	//common.ExecQueueFunc(func() {
	//
	//})
	loginLog := &LoginLog{
		Uid:      login.Oid.Hex(),
		Platform: login.LastPlatform,
		Ip:       login.LastIp,
		Uuid:     login.Uuid,
		UuidWeb:  login.UuidWeb,
		Ua:       ua,
		CreateAt: utils.Now(),
	}
	common.GetMysql().Create(loginLog)
}
func UpsertToken(token *Token) *common.Err {
	c := common.GetMongoDB().C(cToken)
	selector := bson.M{"_id": token.Oid}
	_, err := c.Upsert(selector, token)
	if err != nil {
		log.Error("Upsert user token error: %s", err)
		return errCode.ServerError.SetErr(err.Error())
	}
	return nil
}
func QueryUser(query map[string]interface{}) *User {
	c := common.GetMongoDB().C(cUser)
	var user User
	if err := c.Find(query).One(&user); err != nil {
		//log.Info("not found user: %s ,err: %v", query, err)
		return nil
	}

	return &user
}
func QueryUsers() []User {
	c := common.GetMongoDB().C(cUser)
	var user []User
	if err := c.Find(nil).All(&user); err != nil {
		log.Info("not found users: ,err: %v", err)
		return nil
	}
	return user
}
func QueryUserNums(query map[string]interface{}) int64 {//获取用户数量
	c := common.GetMongoDB().C(cUser)
	num,err := c.Find(query).Count()
	if err != nil {
		return 0
	}
	return num
}
func QueryUserId(id primitive.ObjectID) User {
	c := common.GetMongoDB().C(cUser)
	var user User
	if err := c.FindId(id).One(&user); err != nil {
		//log.Error("not found user: %s ", query)
		//return nil
	}
	return user
}

func QueryTokenByAccount(account string) User{
	c := common.GetMongoDB().C(cUser)
	var user User
	if err := c.Find(bson.M{"Account": account}).One(&user); err != nil {
		log.Info("not found user by account: %s ", account)
	}
	return user
}
//func UpdateTokenTime(id primitive.ObjectID) error{
//	c := common.GetMgo().C(cToken)
//	selector := bson.M{"_id":id}
//	var update map[string]interface{}
//	if err := c.FindId(id).One(&update); err != nil{
//		log.Info("UpdateTokenTime error query uid: %s ", id)
//		return nil
//	}
//	err :=c.Update(selector,update)
//	if err != nil{
//		log.Error("UpdateTokenTime error: %s", err)
//		return err
//	}
//	return nil
//}
func QueryLogin(id primitive.ObjectID) *Login {
	c := common.GetMongoDB().C(cLogin)
	var login Login
	if err := c.FindId(id).One(&login); err != nil {
		log.Info("not found login: %s ", id)
		return nil
	}
	return &login
}
func QueryToken(token string) *Token {
	c := common.GetMongoDB().C(cToken)
	var query Token
	if err := c.Find(bson.M{"AccessToken": token}).One(&query); err != nil {
		log.Info("not found token: %s ", token)
		return nil
	}
	return &query
}
func QueryTokenBySession(sessionId string) *Token {
	c := common.GetMongoDB().C(cToken)
	var query Token
	if err := c.Find(bson.M{"SessionId": sessionId}).One(&query); err != nil {
		log.Info("not found sessionId: %s ", sessionId)
		return nil
	}
	return &query
}
func QueryTokenByUid(uid primitive.ObjectID) *Token {
	c := common.GetMongoDB().C(cToken)
	var query Token
	if err := c.Find(bson.M{"_id": uid}).One(&query); err != nil {
		log.Info("not found token by uid: %s ", uid)
		return nil
	}
	return &query
}

func UpdateUser(user User) {
	c := common.GetMongoDB().C(cUser)
	query := bson.M{"_id": user.Oid}
	if err := c.Update(query, &user); err != nil {
		log.Error(err.Error())
	}
	//common.ExecQueueFunc(func() {
	//
	//})
	common.GetMysql().Where("oid=?", user.Oid.Hex()).Updates(&user)
}

//押注
func IncUserBet(uid primitive.ObjectID, income int64, betAmount int64) {
	c := common.GetMongoDB().C(cUserInfo)
	userInfo := QueryUserInfo(uid)
	douDouBet := utils.Abs(betAmount)
	if userInfo.DouDouBet <= betAmount {
		douDouBet = userInfo.DouDouBet
	}
	count := 1
	if betAmount < 0 { //回滚操作，api游戏可回滚
		count = -1
	}
	query := bson.M{"_id": uid}
	update := bson.M{"$inc": bson.M{
		"SumBet": betAmount, "DouDouBet": -1 * douDouBet, "WinAndLost": income,
		"SumBetCount": count,
	}}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
	userInfo = QueryUserInfo(uid)
	updateUserInfo2mysql(userInfo)
}

//充值
func IncUserCharge(uid primitive.ObjectID, amount int64) {
	c := common.GetMongoDB().C(cUserInfo)
	query := bson.M{"_id": uid}
	update := bson.M{"$inc": bson.M{"SumCharge": amount}}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
	userInfo := QueryUserInfo(uid)
	updateUserInfo2mysql(userInfo)
}

//doudou
func IncUserDoudou(uid primitive.ObjectID, doudou int64) {
	c := common.GetMongoDB().C(cUserInfo)
	query := bson.M{"_id": uid}
	update := bson.M{"$inc": bson.M{"SumDouDou": doudou}}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
	userInfo := QueryUserInfo(uid)
	updateUserInfo2mysql(userInfo)
}

//增加用户换豆豆需要的流水
func IncUserDouDouBet(uid primitive.ObjectID, douDouBet int64) {
	c := common.GetMongoDB().C(cUserInfo)
	query := bson.M{"_id": uid}
	update := bson.M{"$inc": bson.M{"DouDouBet": douDouBet}}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
	userInfo := QueryUserInfo(uid)
	updateUserInfo2mysql(userInfo)
}

//增加用户累计在线时长
func IncUserSumOnlineSec(uid primitive.ObjectID, onlineSec int64) {
	c := common.GetMongoDB().C(cUserInfo)
	query := bson.M{"_id": uid}
	update := bson.M{"$inc": bson.M{"SumOnlineSec": onlineSec}}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
	userInfo := QueryUserInfo(uid)
	updateUserInfo2mysql(userInfo)
}

//增加用户累计佣金
func IncUserSumAgentBalance(uid primitive.ObjectID, agentBalance int64) {
	c := common.GetMongoDB().C(cUserInfo)
	query := bson.M{"_id": uid}
	update := bson.M{"$inc": bson.M{"SumAgentBalance": agentBalance}}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
	userInfo := QueryUserInfo(uid)
	updateUserInfo2mysql(userInfo)
}

//增加活动总和
func IncUserActivityTotal(uid primitive.ObjectID, amount int64) {
	c := common.GetMongoDB().C(cUserInfo)
	query := bson.M{"_id": uid}
	update := bson.M{"$inc": bson.M{"ActivityTotal": amount}}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
	userInfo := QueryUserInfo(uid)
	updateUserInfo2mysql(userInfo)
}
//增加GiftCode总和
func IncUserGiftCode(uid primitive.ObjectID, amount int64) {
	c := common.GetMongoDB().C(cUserInfo)
	query := bson.M{"_id": uid}
	update := bson.M{"$inc": bson.M{"GiftCode": amount}}
	if _, err := c.Upsert(query, update); err != nil {
		log.Error(err.Error())
	}
	userInfo := QueryUserInfo(uid)
	updateUserInfo2mysql(userInfo)
}
func UpsertUserInfo(uid primitive.ObjectID, userInfo UserInfo) *common.Err {
	c := common.GetMongoDB().C(cUserInfo)
	selector := bson.M{"_id": uid}
	_, err := c.Upsert(selector, userInfo)
	if err != nil {
		log.Error("Upsert user userInfo error: %s", err)
		return errCode.ServerError.SetErr(err.Error())
	}
	updateUserInfo2mysql(userInfo)
	return nil
}
func updateUserInfo2mysql(userInfo UserInfo) {
	common.ExecQueueFunc(func() {
		var u UserInfo
		common.GetMysql().Where("oid=?", userInfo.Oid.Hex()).First(&u)
		userInfo.ID = u.ID
		if userInfo.FistChargeTime.IsZero() {
			userInfo.FistChargeTime = time.Unix(0, 0)
		}
		common.GetMysql().Save(&userInfo)
	})
}
func QueryUserInfo(uid primitive.ObjectID) UserInfo {
	c := common.GetMongoDB().C(cUserInfo)
	var userInfo UserInfo
	if err := c.FindId(uid).One(&userInfo); err != nil {
	}
	return userInfo
}
func SetUserInfoVipLevel(uid primitive.ObjectID,level int) *common.Err {
	c := common.GetMongoDB().C(cUserInfo)
	selector := bson.M{"_id": uid}
	update := bson.M{"$set":bson.M{"VipLevel":level}}
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("Upsert user userInfo VipLevel error: %s", err)
		return errCode.ServerError.SetErr(err.Error())
	}
	userInfo := QueryUserInfo(uid)
	updateUserInfo2mysql(userInfo)
	return nil
}
func SetUserInfoSafeStatus(uid primitive.ObjectID,status int) *common.Err {
	c := common.GetMongoDB().C(cUserInfo)
	selector := bson.M{"_id": uid}
	update := bson.M{"$set":bson.M{"SafeStatus":status}}
	_, err := c.Upsert(selector, update)
	if err != nil {
		log.Error("Upsert user userInfo SafeStatus error: %s", err)
		return errCode.ServerError.SetErr(err.Error())
	}
	userInfo := QueryUserInfo(uid)
	updateUserInfo2mysql(userInfo)
	return nil
}
func QueryAllUserInfo(query bson.M) []UserInfo {
	c := common.GetMongoDB().C(cUserInfo)
	var userInfo []UserInfo
	if err := c.Find(query).All(&userInfo); err != nil {
	}
	return userInfo
}