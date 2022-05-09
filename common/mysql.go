package common

import (
	"github.com/panjf2000/ants/v2"
	"github.com/yireyun/go-queue"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	slog "log"
	"os"
	"sync"
	"time"
	"vn/common/myLog"
	"vn/framework/mqant/log"
)

var mysqlDB *gorm.DB
var onceMysql sync.Once
var queueFunc *queue.EsQueue

type QueueFuc func()

func InitMysql(conf *DBConf)  {
	onceMysql.Do(func() {
		newLogger := myLog.New(
			slog.New(os.Stdout, "\r\n", slog.LstdFlags), // io writer
			myLog.Config{
				SlowThreshold: time.Second/2,   // Slow SQL threshold
				LogLevel:      logger.Error, // Log level
				Colorful:      true,         // Disable color
			},
		)
		dsn := conf.MysqlDns
		MysqlDB, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
			Logger: newLogger,
		})
		db,_ := MysqlDB.DB()
		db.SetMaxIdleConns(512)
		db.SetMaxOpenConns(512)
		db.SetConnMaxLifetime(30*time.Second)
		mysqlDB = MysqlDB
		if err != nil {
			log.Error("failed to connect database:", err)
			os.Exit(1)
		}else{
			log.Info("connect mysql database success")
		}
		//queueFunc = queue.NewQueue(20480)
		//go func() {
		//	runFuncQueue()
		//}()
		pool, _ = ants.NewPool(200)
		pool.Running()
	})
}
var pool *ants.Pool
func GetMysql() *gorm.DB {
	if mysqlDB != nil {
		return mysqlDB
	}
	panic("mysql db is not init.")
	return mysqlDB
}



func ExecQueueFunc(f QueueFuc){
	//ok,quantity := queueFunc.Put(f)
	//if !ok{
	//	log.Error("mysql queue is full, cur quantity: %v",quantity)
	//}

	if err := pool.Submit(func() {
		f()
	});err != nil{
		log.Error("mysql queue error: %s", err)
	}
	//log.Info("Running: %v", length)
	//w.Stop()
	//go func() {
	//	defer func() {
	//		if r := recover(); r != nil {
	//			buff := make([]byte, 1024)
	//			runtime.Stack(buff, false)
	//			log.Error("mysql panic(%v)\n info:%s", r, string(buff))
	//		}
	//	}()
	//	f()
	//}()
}

//func runFuncQueue()  {
//	defer func() {
//		if r := recover(); r != nil {
//			buff := make([]byte, 1024)
//			runtime.Stack(buff, false)
//			log.Error("mysql queue panic(%v)\n info:%s", r, string(buff))
//		}
//		runFuncQueue()
//	}()
//	for{
//		time.Sleep(100* time.Millisecond)
//		ok := true
//		for ok{
//			val,_ok,_ := queueFunc.Get()
//			if _ok{
//				f := val.(QueueFuc)
//				go f()
//			}
//			ok = _ok
//		}
//	}
//}
//type MysqlBean struct {}
//func (s *MysqlBean) BeforeCreate(tx *gorm.DB) error{
//	self := reflect.ValueOf(s)
//	value := self.FieldByName("Oid")
//	fmt.Println(value)
//
//	return nil
//}