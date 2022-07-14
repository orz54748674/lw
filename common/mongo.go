package common

//
//var onceMongo sync.Once
//var instance *myMongo
//
type DBConf struct {
	Host     string
	DbName   string
	User     string
	Password string
	MysqlDns string
}

var MongoConfig *DBConf

//var mgoSession *mgo.Session
//func InitMongo(conf *DBConf)  {
//	onceMongo.Do(func() {
//		if instance != nil {
//			return
//		}
//		queueSession = queue.NewQueue(256)
//		mgoSession, MgoError = mgo.Dial(conf.Host)
//		if MgoError != nil {
//			log.Error("myMongo db connect error")
//			os.Exit(1)
//		}
//		// 选择DB
//		instance = &myMongo{mgo.Database{Session: mgoSession, Name: conf.DbName}}
//		// 登陆
//		MgoError = instance.Login(conf.User, conf.Password)
//		if MgoError != nil {
//			log.Error("myMongo db login error")
//			os.Exit(1)
//		}
//		log.Info("myMongo db init success")
//		go func() {
//			runQueue()
//		}()
//	})
//	initListener()
//}
//var queueSession *queue.EsQueue
//
//type sessionFlag struct {
//	session *mgo.Session
//	createTime time.Time
//}
//func GetMgo() *myMongo {
//	if instance != nil {
//		newSession := mgoSession.Clone()
//		queueSession.Put(&sessionFlag{
//			session: newSession,
//			createTime: time.Now(),
//		})
//		instance.Session = newSession
//		return instance
//	}
//	panic("myMongo db is not init.")
//	return instance
//}
//
//type myMongo struct {
//	mgo.Database
//}
//
//func runQueue(){
//	for{
//		time.Sleep(1* time.Second)
//		ok := true
//		for ok{
//			val,_ok,_ := queueSession.Get()
//			if _ok{
//				sessionFlag := val.(*sessionFlag)
//				now := time.Now().Unix()
//				target := sessionFlag.createTime.Unix()
//				d := now - target
//				if d > 5{
//					sessionFlag.session.Close()
//					//log.Info("session close: %v", &sessionFlag.session)
//				}else{
//					queueSession.Put(sessionFlag)
//				}
//			}
//			ok =_ok
//		}
//
//	}
//}
