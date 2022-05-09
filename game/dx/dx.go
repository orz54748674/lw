package dx

import (
	"fmt"
	"time"
	"vn/common"
	"vn/common/errCode"
	"vn/common/utils"
	"vn/framework/mqant-modules/room"
	"vn/framework/mqant/conf"
	"vn/framework/mqant/gate"
	"vn/framework/mqant/log"
	"vn/framework/mqant/module"
	basemodule "vn/framework/mqant/module/base"
	"vn/framework/mqant/server"
	"vn/game"
	"vn/storage/dxStorage"
	"vn/storage/gameStorage"

	"github.com/yireyun/go-queue"
)

var Module = func() module.Module {
	this := new(dxRoom)
	return this
}

type dxRoom struct {
	basemodule.BaseModule
	room *room.Room
}

const (
	tableId              = "1"
	actionInfo           = "HD_info"
	actionHistory        = "HD_history"
	actionJackpotHistory = "HD_jackpotHistory"
	actionDetails        = "HD_details"
	actionPlay           = "HD_play"
	actionDxInfo         = "/dx/dxInfo"
	actionCurRoundBet    = "HD_CurRoundBet"
)

func (self *dxRoom) GetType() string {
	//很关键,需要与配置文件中的Module配置对应
	return string(game.BiDaXiao)
}
func (self *dxRoom) Version() string {
	//可以在监控时了解代码版本
	return "1.0.0"
}
func (s *dxRoom) OnInit(app module.App, settings *conf.ModuleSettings) {
	s.BaseModule.OnInit(s, app, settings,
		server.RegisterInterval(15*time.Second),
		server.RegisterTTL(30*time.Second),
	)
	s.room = room.NewRoom(s.App)

	hook := game.NewHook(s.GetType())
	hook.RegisterAndCheckLogin(s.GetServer(), actionInfo, s.info)
	hook.RegisterAndCheckLogin(s.GetServer(), actionHistory, s.history)
	hook.RegisterAndCheckLogin(s.GetServer(), actionJackpotHistory, s.jackpotHistory2)
	hook.RegisterAndCheckLogin(s.GetServer(), actionDetails, s.details)
	hook.RegisterAndCheckLogin(s.GetServer(), actionPlay, s.play)
	hook.RegisterAndCheckLogin(s.GetServer(), actionCurRoundBet, s.CurRoundBet)
	s.GetServer().Register(actionDxInfo, s.dxInfo)

	_, err := s.room.CreateById(s.App, "1", s.NewTable)
	if err != nil {
		log.Error(err.Error())
	}
	incDataExpireDay := time.Duration(
		app.GetSettings().Settings["mongoIncDataExpireDay"].(float64)) * 24 * time.Hour
	dxStorage.Init(incDataExpireDay)
	gameStorage.UpsertGameReboot(game.BiDaXiao, "false")

	s.GetServer().RegisterGO("/dx/onDisconnect", s.onDisconnect)
	common.AddListener(s.GetServerID(), common.EventDisconnect, "/dx/onDisconnect")
}

func (self *dxRoom) Run(closeSig chan bool) {
	log.Info("%v模块运行中...", self.GetType())
	<-closeSig
}

func (self *dxRoom) OnDestroy() {
	//一定别忘了继承
	self.BaseModule.OnDestroy()
}

func (self *dxRoom) NewTable(app module.App, tableId string) (room.BaseTable, error) {
	table := NewTable(
		self, app,
		room.TableId(tableId),
		room.Router(func(TableId string) string {
			return fmt.Sprintf("%v://%v/%v", self.GetType(), self.GetServerId(), tableId)
		}),
		room.Capaciity(2048),
		//room.DestroyCallbacks(func(table room.BaseTable) error {
		//	log.Info("回收了房间: %v", table.TableId())
		//	_ = self.room.DestroyTable(table.TableId())
		//	return nil
		//}),
	)
	return table, nil
}
func (self *dxRoom) info(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	uid := session.GetUserID()
	res := make(map[string]interface{}, 2)
	res["dxServerId"] = self.GetServerID()
	res["chipList"] = []int{1000, 10000, 50000, 100000, 500000, 1000000, 10000000, 50000000}
	table := self.room.GetTable(tableId)
	if table == nil {
		table, _ = self.room.CreateById(self.App, tableId, self.NewTable)
	}
	mytable := table.(*DxTable)
	mytable.Start()
	if mytable.dxRun.curDx == nil {
		time.Sleep(300 * time.Millisecond)

	}
	res["dx"] = mytable.dxRun.curDx.Notify
	res["timeLeft"] = mytable.dxRun.TimeLeft
	big, small, amount := dxStorage.QueryMyBet(mytable.dxRun.curDx.ShowId, uid)
	res["myBet"] = map[string]int64{"big": big, "small": small, "amount": amount}
	if mytable.dxRun.TimeLeft < 1 {
		res["curJackpot"] = mytable.dxRun.GetNextJackpot()
	} else {
		res["curJackpot"] = mytable.dxRun.curDx.Jackpot
	}
	erro := table.PutQueue("empty", session, msg)
	if erro != nil {
		//return errCode.ServerError.SetErr(erro.Error()).GetMap(), erro
		log.Warning(erro.Error())
	}
	return errCode.Success(res).GetMap(), nil
}
func (self *dxRoom) history(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"size"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), ok
	}
	size, _ := utils.ConvertInt(params["size"])
	if size > 101 {
		return errCode.PageSizeErr.SetKey().GetMap(), nil
	}
	res := make(map[string]interface{}, 2)
	res["history"] = dxStorage.GetHistory(int(size))
	return errCode.Success(res).GetMap(), nil
}

//func (self *dxRoom) jackpotHistory(session gate.Session, params map[string]interface{}) (map[string]interface{},error) {
//	//size := 100
//	log.Info("start")
//	res := dxStorage.QueryJackpotLog(100)
//	bigCount := 0; smallCount :=0
//	for index,data := range res{
//		dx := dxStorage.QueryDx(data.GameId)
//		res[index].Jackpot = dx.Jackpot
//		res[index].Result = dx.Result
//		res[index].CreateAt = dx.CreateAt
//		res[index].ResultAmount = dx.BetSmall
//		res[index].LogCount = len(res[index].JackpotLog)
//		var newJackpotLog []dxStorage.DxJackpotDetails
//		for i,_ := range res[index].JackpotLog{
//			res[index].JackpotLog[i].UserType = ""
//			newJackpotLog = append(newJackpotLog,res[index].JackpotLog[i])
//			if i == 20 {
//				break
//			}
//		}
//		res[index].JackpotLog = newJackpotLog
//		if dx.Result == dxStorage.ResultBig{
//			bigCount++
//		}else{
//			smallCount++
//		}
//	}
//	result := make(map[string]interface{}, 3)
//	result["bigCount"] = bigCount
//	result["smallCount"] = smallCount
//	result["list"] = res
//	log.Info("end")
//	return errCode.Success(result).GetMap(), nil
//}
func (self *dxRoom) jackpotHistory2(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	if check, ok := utils.CheckParams2(params,
		[]string{"offset", "limit"}); ok != nil {
		return errCode.ErrParams.SetKey(check).GetMap(), ok
	}
	offset, _ := utils.ConvertInt(params["offset"])
	limit, _ := utils.ConvertInt(params["limit"])
	if limit > 100 {
		return errCode.ErrParams.SetKey("limit").GetI18nMap(), nil
	}
	res := dxStorage.QueryJackpotLog2(int(offset), int(limit))
	smallCount, bigCount := dxStorage.QueryJackpotCount()
	result := make(map[string]interface{}, 3)
	result["bigCount"] = bigCount
	result["smallCount"] = smallCount
	result["TotalNum"] = bigCount + smallCount
	result["list"] = res

	return errCode.Success(result).GetMap(), nil

}
func (self *dxRoom) details(session gate.Session, params map[string]interface{}) (map[string]interface{}, error) {
	var gameId int64 = 0
	if _, ok := params["gameId"]; ok {
		gameId, _ = utils.ConvertInt(params["gameId"])
	}
	if gameId == 0 {
		table := self.room.GetTable(tableId)
		mytable := table.(*DxTable)
		gameId = mytable.dxRun.curDx.ShowId - 1
	}
	details := dxStorage.GetGameDetails(gameId)
	return errCode.Success(details).GetMap(), nil
}
func (self *dxRoom) dxInfo() (map[string]interface{}, error) {
	table := self.room.GetTable(tableId)
	if table == nil {
		table, _ = self.room.CreateById(self.App, tableId, self.NewTable)
	}
	mytable := table.(*DxTable)
	if mytable.dxRun.curDx == nil {
		mytable.Start()
		time.Sleep(300 * time.Millisecond)
	}
	res := map[string]interface{}{
		"Jackpot": mytable.dxRun.curDx.Jackpot,
		"Big":     mytable.dxRun.curDx.BetBig,
		"Small":   mytable.dxRun.curDx.BetSmall,
	}
	return res, nil
}
func (self *dxRoom) play(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	//table_id := msg["table_id"].(string)
	action := msg["action"].(string)
	table := self.room.GetTable(tableId)
	if table == nil {
		table, _ = self.room.CreateById(self.App, tableId, self.NewTable)
	}
	//erro := table.PutQueue(action, session, msg)
	//if erro != nil {
	//	log.Warning(erro.Error())
	//	return errCode.ServerError.GetI18nMap(),erro
	//}

	uid := session.GetUserID()
	userSyncExec := getQueue(uid)
	userSyncExec.queue.Put(func() {
		_ = utils.CallReflect(table, action, session, msg)
		//if out != nil{
		//	if len(out) == 2{
		//		errCode := out[0].Interface().(*common.Err)
		//		return errCode.SetAction(game.Nothing).GetMap(),nil
		//	}
		//}
	})
	//log.Info("userSyncExec: %v,userQueue:%v",&userSyncExec,&userQueue)
	userSyncExec.exec()
	return errCode.Success(nil).SetAction(game.Nothing).GetMap(), nil
}
func (self *dxRoom) CurRoundBet(session gate.Session, msg map[string]interface{}) (map[string]interface{}, error) {
	table := self.room.GetTable(tableId)
	if table == nil {
		table, _ = self.room.CreateById(self.App, tableId, self.NewTable)
	}
	myTable := table.(*DxTable)
	res := myTable.GetCurRoundBet()
	return errCode.Success(res).GetMap(), nil
}
var userQueue = common.NewMapRWMutex()

type userSyncExec struct {
	queue   *queue.EsQueue
	running bool
}

func (s *userSyncExec) exec() {
	if s.running {
		return
	}
	s.running = true
	ok := true
	for ok {
		val, _ok, _ := s.queue.Get()
		if _ok {
			f := val.(func())
			f()
		}
		ok = _ok
	}
	s.running = false
}
func getQueue(uid string) *userSyncExec {
	q := userQueue.Get(uid)
	if q != nil {
		return q.(*userSyncExec)
	} else {
		exec := &userSyncExec{
			queue: queue.NewQueue(6),
		}
		userQueue.Set(uid, exec)
		return exec
	}
}
func (s *dxRoom) onDisconnect(uid string) (interface{}, error) {
	userQueue.Remove(uid)
	log.Info("cur userQueue len: %v, uid: %s", len(userQueue.Data), uid)
	return nil, nil
}
