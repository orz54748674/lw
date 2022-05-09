package lotteryStorage

import (
	"vn/common"
	"vn/framework/mongo-driver/bson"
	"vn/framework/mongo-driver/mongo"
)

/**
 *  @title	SetName
 *	@description	获取集合名
 *	@return	 setName	string	集合名
 */
func (m *BetStats) SetName() string {
	return cLotteryBetRecord
}

func (m *BetStats) C() *common.Collect {
	return common.GetMongoDB().C(m.SetName())
}

func (m *BetStats) NumberTotalPatAmount(number, lotteryCode string) (totalPatAmount int64, err error) {
	pipe := mongo.Pipeline{
		{{"$group",
			bson.M{
				"_id":            "$PlayCode",
				"TotalPatAmount": bson.M{"$sum": bson.M{"$add": "$TotalPatAmount"}},
			},
		}},
		{{"$match", bson.M{"Number": number, "LotteryCode": lotteryCode}}},
	}
	res := map[string]interface{}{}
	err = m.C().Pipe(pipe).One(&res)
	if err == mongo.ErrNoDocuments {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	return int64(res["TotalPatAmount"].(float64)), nil
}

func (m *BetStats) GetTotalPatAmounts(number, lotteryCode string) (res []map[string]interface{}, err error) {
	pipe := mongo.Pipeline{
		{{"$group",
			bson.M{
				"_id":            "$PlayCode",
				"TotalPatAmount": bson.M{"$sum": bson.M{"$add": "$TotalPatAmount"}},
			},
		}},
		{{"$match", bson.M{"Number": number, "LotteryCode": lotteryCode}}},
		{{"$sort", bson.M{"TotalPatAmount": 1}}},
	}
	res = []map[string]interface{}{}
	err = m.C().Pipe(pipe).One(&res)
	if err != nil && err != mongo.ErrNoDocuments {
		return res, err
	}
	return res, nil
}

func (m *BetStats) GetMinAmountCodes(number, lotteryCode, subPlayCode string) (betStats *BetStats, err error) {
	find := bson.M{"Number": number, "LotteryCode": lotteryCode, "SubPlayCode": subPlayCode}
	err = m.C().Find(find).Sort("+TotalPatAmount").Limit(1).One(betStats)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	return
}
