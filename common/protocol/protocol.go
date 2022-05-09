package protocol

import (
	"encoding/json"
	"vn/common"
	basegate "vn/framework/mqant/gate/base"
	"vn/framework/mqant/log"
	"vn/game"
	gate2 "vn/gate"
)

//----客户端访问
const Empty 			string = "HD_Empty"   //空协议

const Enter 			string = "HD_Enter"   //进入房间
const AutoEnter 		string = "HD_AutoEnter"   //自动进入房间
const XiaZhu 			string = "HD_XiaZhu" //下注
const LastXiaZhu 		string = "HD_LastXiaZhu" //上一轮下注
const DoubleXiaZhu 		string = "HD_DoubleXiaZhu" //加倍下注
const GetPlayerList 	string = "HD_GetPlayerList" //获取玩家列表
const GetResultsRecord 	string = "HD_GetResultsRecord" //获取开奖结果
const GetPrizeRecord 	string = "HD_GetPrizeRecord" //获取奖池记录
const QuitTable 		string = "HD_QuitTable" //退出房间
const Info 				string = "HD_info" //大厅信息
const GetTableList 		string = "HD_GetTableList" //获取桌子列表
const GetBaseScoreList	string = "HD_GetBaseScoreList" //获取底分列表
const CreateTableReq	string = "HD_CreateTableReq" //创建房间请求
const GetShortCutList 	string = "HD_GetShortCutList" //获取快捷语
const SendShortCut 		string = "HD_SendShortCut" //发送快捷语
const GetWinLoseRank		string = "HD_GetWinLoseRank" //获取输赢排行榜
const CheckPlayerInGame     string = "HD_CheckPlayerInGame" //玩家是否在房间中
const Spin 			string = "HD_Spin" //
const SpinFree 			string = "HD_SpinFree" //
const GetResults 			string = "HD_GetResults" //
const SelectBonusSymbol 			string = "HD_SelectBonusSymbol" //
const SelectBonusTimes 			string = "HD_SelectBonusTimes" //
const BonusTimeOut 			string = "HD_BonusTimeOut" //
const MiniTimeOut 			string = "HD_MiniTimeOut" //
const EnterBonusGame 			string = "HD_EnterBonusGame" //
const EnterMiniGame 			string = "HD_EnterMiniGame" //

const SelectMiniSymbol 			string = "HD_SelectMiniSymbol" //
const SpinTrial 			string = "HD_SpinTrial" //
const SpinTrialFree 			string = "HD_SpinTrialFree" //
const GetJackpot 			string = "HD_GetJackpot" //
const SelectFreeGame 			string = "HD_SelectFreeGame" //
const SelectTrialFree			string = "HD_SelectTrialFreeGame" //
const GetHallInfo 			string = "HD_GetHallInfo" //
const Ready 			string = "HD_Ready"   //
const RobotReady 			string = "HD_RobotReady"   //
const AutoReady 			string = "HD_AutoReady"   //
const MasterStartGame 			string = "HD_MasterStartGame" //开始游戏
const ShowPoker 			string = "HD_ShowPoker"   //亮牌
const CancelShowPoker 			string = "HD_CancelShowPoker"   //亮牌
const DealPoker 			string = "HD_DealPoker"   //发牌
const DealShowPoker 			string = "HD_DealShowPoker"   //
const SortPoker 			string = "HD_SortPoker"   //整理牌
const HintPoker 			string = "HD_HintPoker"   //提示牌
const SwitchMode 			string = "HD_SwitchMode"   //切换模式
const PutPoker 			string = "HD_PutPoker"   //出牌
const MovePoker 			string = "HD_MovePoker"   //移牌
const DrawPoker 			string = "HD_DrawPoker"   //摸牌
const EatPoker 			string = "HD_EatPoker"   //
const PhomPoker 			string = "HD_PhomPoker"   //
const GivePoker 			string = "HD_GivePoker"   //
const GetPhomPoker 			string = "HD_GetPhomPoker"   //
const CheckPoker 			string = "HD_CheckPoker"   //过牌
const GetPokerType 			string = "HD_GetPokerType"   //获取牌型
const GrabDealer			string = "HD_GrabDealer"   //抢庄
const NotifyDealerResult			string = "HD_NotifyDealerResult"   //
const GetJackpotRecord     string = "HD_GetJackpotRecord" //开奖记录
const GameInvite	string = "HD_GameInvite" //游戏邀请
const GameInviteRecord	string = "HD_GameInviteRecord" //游戏邀请记录
const InviteEnter 			string = "HD_InviteEnter"   //邀请进入房间
//服务端主动通知
const JieSuan 		string = "HD_JieSuan" //结算

const UpdatePlayerList 		string = "HD_UpdatePlayerList" //更新玩家list
const SwitchRoomState 			string = "HD_SwitchRoomState" //切换房间状态
const Reenter 				string = "HD_Reenter" //GivePoker
const UpdatePlayerNum 		string = "HD_UpdatePlayerNum" //刷新人数
const Disconnect 			string = "HD_Disconnect" //掉线
const RefreshPrizePool 		string = "HD_RefreshPrizePool" //刷新奖池
const UpdatePlayerInfo		string = "HD_UpdatePlayerInfo" //刷新玩家状态
const UpdateTableInfo		string = "HD_UpdateTableInfo" //刷新桌子信息
const NotifyWaitingState 			string = "HD_NotifyWaitingState" //通知状态

//队列函数
const StartGame 			string = "HD_StartGame" //开始游戏
const RobotXiaZhu 			string = "HD_RobotXiaZhu" //下注
const ReadyGame 			string = "HD_ReadyGame" //准备游戏
const ClearTable 				string = "HD_ClearTable" //解散桌子
const RobotEnter 				string = "HD_RobotEnter" //机器人进入房间
const RobotQuitTable 				string = "HD_RobotQuitTable" //机器人退出房间
const RobotBetCalc 				string = "HD_RobotBetCalc" //机器人下注处理
const GetEnterData 				string = "HD_GetEnterData" //获取进入房间的数据

//捕鱼
const KillFish string = "HD_killFish"
const LeiTingKillFish string = "HD_leiTingKillFish"
const PlayerFire string = "HD_playerFire"
const ChangeCannon string = "HD_changeCannon"
const PlayerLeave string = "HD_playerLeave"
const FishGroup string = "HD_fishGroup"
const FishTide string = "HD_fishTide"
const FishTideCome string = "HD_fishTideCome"
const SpecialKillFish string = "HD_specialKillFish"
const ChangeSeat string = "HD_changeSeat"


func DealProtocolFormat(in interface{},gameType game.Type,action string,error *common.Err) interface{}{
	info := struct {
		Data interface{}
		GameType game.Type
		Action string
		ErrMsg string
		Code int
	}{
		Data: in,
		GameType: gameType,
		Action: action,
	}
	if error == nil{
		info.Code = 0
		info.ErrMsg = "操作成功"
	}else {
		info.Code = error.Code
		info.ErrMsg = error.SetKey().ErrMsg
	}
	return info
}
func SendPack(uid string,topic string,in interface{}) {
	b,_ := json.Marshal(in)
	sessionBean := gate2.QuerySessionBean(uid)
	if sessionBean !=nil{
		session,err := basegate.NewSession(common.App, sessionBean.Session)
		if err != nil{
			log.Error(err.Error())
		}else{
			if err := session.SendNR(topic, b);err != ""{
				log.Error(err)
			}
		}
	}
}
func SendPackToAll(topic string,in interface{}) {
	b,_ := json.Marshal(in)
	allSession := gate2.QueryAllSession()
	if allSession != nil{
		for _,v := range *allSession{
			session,err := basegate.NewSession(common.App, v.Session)
			if err != nil{
				log.Error(err.Error())
			}else{
				if err := session.SendNR(topic, b);err != ""{
					log.Error(err)
				}
			}
		}
	}
}





