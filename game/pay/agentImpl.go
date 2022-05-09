package pay

import (
	"vn/framework/mongo-driver/bson/primitive"
	"vn/game"
)

/**
uid: 下注人的Uid
betId: 下注动作的事件ID，同一局游戏可重复
 */
func ParseBet(uid primitive.ObjectID,amount int64,betId string,game game.Type)  {

}

func CheckoutAgentIncome(uid primitive.ObjectID,amount int64,betId string,game game.Type)  {

}