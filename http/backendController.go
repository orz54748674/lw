package http

type BackendController struct {
	BaseController
}

//func (s *BackendController)profitUpdate(w http.ResponseWriter, r *http.Request){
//	_ = r.ParseForm()
//	p := r.Form
//	if _,ok := utils.CheckParams(p, []string{"admin_id","bot_balance","game_type"});ok != nil{
//		return
//	}
//	gameType := p["game_type"][0]
//	adminID := p["admin_id"][0]
//	botBalance,_ := strconv.ParseInt(p["bot_balance"][0],10,64)
//	gameStorage.IncProfit(game.Type(gameType),0,botBalance,-botBalance)
//	profitLog := gameStorage.GameProfitLog{
//		AdminID:adminID,
//		GameType: game.Type(gameType),
//		BotBalance: botBalance,
//		CreateAt:time.Now(),
//	}
//	gameStorage.InsertProfitLog(profitLog)
//}
//
//func (s *BackendController)walletOperate(w http.ResponseWriter, r *http.Request){
//	_ = r.ParseForm()
//	p := r.Form
//	if _,ok := utils.CheckParams(p, []string{"userID","operateV"});ok != nil{
//		return
//	}
//	userID := p["userID"][0]
//	operateV,_ := strconv.ParseInt(p["operateV"][0],10,64)
//
//	billType := walletStorage.TypeIncome
//	if operateV < 0 {
//		billType = walletStorage.TypeExpenses
//	}
//	bill := walletStorage.NewBill(userID,billType,walletStorage.EventAdmin,
//		utils.GetDateStr(utils.Now()),operateV)
//	if err := walletStorage.OperateVndBalance(bill);err== nil {
//		walletStorage.NotifyUserWallet(userID)
//		agentStorage.OnWalletChange(userID)
//	}
//}
