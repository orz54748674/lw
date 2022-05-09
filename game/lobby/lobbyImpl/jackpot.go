package lobbyImpl

import (
	"sort"
	"vn/common/utils"
	"vn/game"
	pk "vn/game/mini/poker"
	"vn/storage/gbsStorage"
	"vn/storage/slotStorage/slotCsStorage"
	"vn/storage/slotStorage/slotLsStorage"
	"vn/storage/slotStorage/slotSexStorage"
	"vn/storage/yxxStorage"
)

type MaxJackpot struct {
	GameType game.Type
	Jackpot  int64
}
func GetMaxJackpotAll() interface{} {
	res := make([]MaxJackpot,0)
	//大小
	jackpot,_ := utils.ConvertInt(getDxInfo().(map[string]interface{})["Jackpot"])
	res = append(res,MaxJackpot{
		GameType: game.BiDaXiao,
		Jackpot: jackpot,
	})
	//鱼虾蟹
	yxx := yxxStorage.GetTableInfo("000000")
	jackpot = yxx.PrizePool
	res = append(res,MaxJackpot{
		GameType: game.YuXiaXie,
		Jackpot: jackpot,
	})
	//龙神
	goldJackpot,_:= slotLsStorage.GetJackpot()
	jackpot = goldJackpot[len(goldJackpot) - 1]
	res = append(res,MaxJackpot{
		GameType: game.SlotLs,
		Jackpot: jackpot,
	})
	//财神
	cs := slotCsStorage.GetJackpot()
	jackpot = cs[len(cs) - 1]
	res = append(res,MaxJackpot{
		GameType: game.SlotCs,
		Jackpot: jackpot,
	})
	//性感
	sex := slotSexStorage.GetJackpot()
	jackpot = sex[len(sex) - 1]
	res = append(res,MaxJackpot{
		GameType: game.SlotSex,
		Jackpot: jackpot,
	})
	//Mini Poker
	jackpot = pk.GetPrizePool()[500000]
	res = append(res,MaxJackpot{
		GameType: game.MiniPoker,
		Jackpot: jackpot,
	})
	//猜大小
	gbs := gbsStorage.GetGameConf()
	jackpot = gbs[len(gbs) - 1].PoolVal
	res = append(res,MaxJackpot{
		GameType: game.GuessBigSmall,
		Jackpot: jackpot,
	})
	sort.Slice(res, func(i, j int) bool {
		return res[i].Jackpot > res[j].Jackpot
	})
	return res
}