package cardCddN

import (
	"sort"
	"vn/common/protocol"
)

const RoundValue = 7//一手的价值
const BombExtraValue = 20 //炸弹连对，额外价值

//手牌权值结构
type HandCardValue struct {
	SumValue int//手牌总价值
	NeedRound int//需要打几手牌
}
//最佳出牌结构
type BestHandCard struct {
	SumValue int//手牌总价值
	NeedRound int//需要打几手牌
	Poker []int //最优出牌
	FirstPoker []int //当前出牌，用来记录出的第一手牌
}
//牌型组合数据结构
type CardGroupData struct {
	cgType PokerType //类型
	nValue int //该牌的价值
}
//用来计算出牌的数据结构
type CalcCardData struct {
	Round int
	Value int
	IsGetTotal bool //是否需要计算当前出的牌的值，用来获取当前牌的总价值
}
func (this *MyTable) GetGroupDataValue(cgType PokerType,maxCard int) int{
	maxCard = maxCard % 0x10
	if cgType < 0{//
		return 0
	}else if cgType == Single{//
		return maxCard - 10
	}else if cgType == Pair{
		return maxCard - 10
	}else if cgType == ThreeOfAKind{
		return maxCard - 10
	}else if cgType == Straight{
		return maxCard - 10 + 1
	}else if cgType == StraightPair3{
		return maxCard - 3 + 7
	}else if cgType == Bomb{
		return maxCard - 3 + 9
	}else if cgType == StraightPair4{
		return maxCard - 3 + 11
	}else{
		return 0
	}
}
func (this *MyTable) GetHandCardValue(pk []int,data CalcCardData) (uctHandCardValue HandCardValue){
	handPoker := make([]int,len(pk))
	copy(handPoker,pk)
	nHandCardCount := len(handPoker)
	if nHandCardCount == 0{
		uctHandCardValue.SumValue = data.Value
		uctHandCardValue.NeedRound = data.Round
		return uctHandCardValue
	}
	//判断是否可以一手出完牌
	uctCardGroupData := this.InsSurCardsType(handPoker)
	if uctCardGroupData.cgType != ERROR { //一手出完
		data.Round += 1
		uctHandCardValue.SumValue = uctCardGroupData.nValue + data.Value
		uctHandCardValue.NeedRound = data.Round
		return uctHandCardValue
	}
	data.Round += 1
	this.GetPutCardList(handPoker,data)
	uctHandCardValue.NeedRound = -1 //用来判断递归没结束
	return uctHandCardValue
}
//是否只有一手牌
func (this *MyTable) InsSurCardsType(handPoker []int) (uctCardGroupData CardGroupData){
	pkType,maxCard,_,_ := this.GetCardType(handPoker)
	if pkType == 0{
		uctCardGroupData.cgType = ERROR
		return uctCardGroupData
	}
	uctCardGroupData.cgType = pkType
	uctCardGroupData.nValue = this.GetGroupDataValue(pkType,maxCard)
	return uctCardGroupData
}
//获取最优出牌方案
var  start = int64(0)
var  end = int64(0)
func (this *MyTable) GetPutCardList(handPoker []int,data CalcCardData){
	//if data.Round == 0{
	//	start = time.Now().UnixNano()
	//}
	//炸弹
	compList := this.CompFourOfAKindNoCombine(handPoker)
	this.DealPutCardList(handPoker,compList,data,false)
	if data.Round >= 1 && len(compList) > 0{
		return
	}
	//if data.Round == 0{
	//	end = time.Now().UnixNano()
	//	log.Info("bomb cost time = %d",time.Duration(end -start) / time.Millisecond)
	//}
	//连对
	compList = this.CompStraightPairNoCombine(handPoker)
	this.DealPutCardList(handPoker,compList,data,false)
	if (data.Round >= 1 || this.BestHandCard.NeedRound == 0) && len(compList) > 0{//一手出完
		return
	}

	//顺子
	compList = this.CompStraightNoCombine(handPoker)
	this.DealPutCardList(handPoker,compList,data,true)
	if (data.Round >= 1 || this.BestHandCard.NeedRound == 0) && len(compList) > 0{
		return
	}

	//三条
	compList = this.CompThreeOfAKindNoCombine(handPoker)
	this.DealPutCardList(handPoker,compList,data,false)
	if (data.Round >= 1 || this.BestHandCard.NeedRound == 0) && len(compList) > 0{
		return
	}

	//对子
	compList = this.CompPairNoCombine(handPoker)
	this.DealPutCardList(handPoker,compList,data,false)
	if (data.Round >= 1 || this.BestHandCard.NeedRound == 0) && len(compList) > 0{
		return
	}

	//单张
	canPutSingle := true
	if data.Round == 0 && len(this.BestHandCard.Poker) != 0{//除单牌外，还有其他牌型
		for _,v := range this.PlayerList{
			if v.IsHavePeople && v.Ready && len(v.HandPoker) == 1{ //有人报单,优先出单牌之外的
				canPutSingle = false
			}
		}
	}
	putMaxSingle := false //是否出最大单牌
	if data.Round == 0 && len(this.BestHandCard.Poker) == 0 {
		for _,v := range this.PlayerList{
			if v.IsHavePeople && v.Ready && len(v.HandPoker) == 1{ //有人报单且自己只有单牌时，优先出最大的单牌
				putMaxSingle = true
			}
		}
	}
	if putMaxSingle {//出最大单牌
		this.BestHandCard.Poker = []int{handPoker[len(handPoker) - 1]}
		this.BestHandCard.SumValue += 100 //为了被动出牌取最大单牌
		return
	}
	if canPutSingle && !putMaxSingle{
		compList = this.CompSingleNoCombine(handPoker)
		this.DealPutCardList(handPoker,compList,data,false)
		if (data.Round >= 1 || this.BestHandCard.NeedRound == 0) && len(compList) > 0{
			return
		}

	}
}
func (this *MyTable) DealPutCardList(pk []int,compList []CompList,dataSrc CalcCardData,calcLen bool){//calcLen是否需要计算长度
	for _,v := range compList{
		remainPk := this.RemoveTblInList(pk,v.List)
		data := dataSrc
		if calcLen{
			data.Value += len(v.List)
		}
		if data.Round != 0 || data.IsGetTotal {//当前出的牌不算权值
			data.Value += this.InsSurCardsType(v.List).nValue
		}
		if data.Round == 0{ //记录第一手牌
			this.BestHandCard.FirstPoker = v.List
		}
		tmpHandValue := this.GetHandCardValue(remainPk,data)
		if tmpHandValue.NeedRound != -1{//遍历结束
			tmpHandValue.SumValue = tmpHandValue.SumValue - tmpHandValue.NeedRound * RoundValue
			if tmpHandValue.NeedRound == 1 && !dataSrc.IsGetTotal{//倒数第二手牌是最大牌时，直接出最大牌
				res := this.CheckOtherHaveBigger(this.GetCurIdx(),this.BestHandCard.FirstPoker)
				if !res{
					tmpHandValue.SumValue = 1000 - tmpHandValue.SumValue
				}
			}
			if tmpHandValue.SumValue  > this.BestHandCard.SumValue{
				this.BestHandCard.SumValue = tmpHandValue.SumValue
				this.BestHandCard.NeedRound = tmpHandValue.NeedRound
				this.BestHandCard.Poker = this.BestHandCard.FirstPoker
			}
		}
	}
}
//被动出牌
func (this *MyTable) GetPutCardListLimit(userID string){
	idx := this.GetPlayerIdx(userID)
	if idx < 0{
		return
	}
	handPoker := this.PlayerList[idx].HandPoker
	if this.CompBigger(this.LastData.LastPutCard,handPoker){ //可以一手出完,不需要计算价值
		this.AutoPutPoker(userID,handPoker)
		return
	}
	this.InitBestCardData()
	this.GetPutCardList(handPoker,CalcCardData{}) //计算不出牌的价值
	putPoker := make([]int,0)
	if this.BestHandCard.NeedRound == 1{
		this.BestHandCard.SumValue = -this.BestHandCard.SumValue + 1000 //为了倒数第二手牌是最大牌时，直接出最大牌
	}
	tmpValue := this.BestHandCard.SumValue - RoundValue//不出，权值要减去一轮权值
	bigPk := this.CheckBiggerPoker(this.LastData.LastPutCard,handPoker)
	putMaxSingle := false
	for _,v := range this.PlayerList{
		if v.IsHavePeople && v.Ready && len(v.HandPoker) == 1{ //有人报单时，优先出最大的单牌
			putMaxSingle = true
		}
	}
	tmpBigPk := make([][]int,0)
	for _,v := range bigPk{
		if len(this.CompFourOfAKind(v)) > 0 || len(this.CompStraightPair(v)) > 0 {//连对和炸弹，直接炸他丫的，管他三七二十一
			putPoker = v
			tmpBigPk = append(tmpBigPk,v)
		}
	}
	if len(tmpBigPk) > 0 {//连对也可能有多种组合
		tmpValue = -1000 //为了后面的判断
		bigPk = tmpBigPk
	}
	if putMaxSingle && len(tmpBigPk) == 0{
		if len(bigPk) > 0{
			putPoker = bigPk[len(bigPk) - 1]
		}
	}else{
		for _,v := range bigPk{
			remainPk := this.RemoveTblInList(handPoker,v)
			this.InitBestCardData()

			this.GetPutCardList(remainPk,CalcCardData{IsGetTotal: true}) //计算权值
			if this.BestHandCard.NeedRound == 0{//倒数第二手牌是最大牌时，直接出最大牌
				res := this.CheckOtherHaveBigger(idx,v)
				if !res{
					putPoker = v
					this.BestHandCard.SumValue = 1000 - this.BestHandCard.SumValue//
				}
			}
			if this.BestHandCard.SumValue > tmpValue { //有更优的
				putPoker = v
				tmpValue = this.BestHandCard.SumValue
			}
		}
	}
	if len(putPoker) > 0{
		this.AutoPutPoker(userID,putPoker)
	}else{
		this.PutQueue(protocol.CheckPoker,userID)
	}
}
func(this *MyTable) CompSingleNoCombine(pk []int) []CompList{ //单张
	list := make([]int,len(pk))
	copy(list,pk)
	tmp := this.CalCardNum(list)
	compList := make([]CompList,0)
	for _,v := range tmp{
		if v.size >= 1 && len(v.pk) >= 1{
			compList = append(compList,CompList{maxPk: v.pk[0],num: 0,List: []int{v.pk[0]}})
		}
	}
	return compList
}
func(this *MyTable) CompPairNoCombine(pk []int) []CompList{
	list := make([]int,len(pk))
	copy(list,pk)
	tmp := this.CalCardNum(list)
	compList := make([]CompList,0)
	for _,v := range tmp{
		if v.size >= 2 && len(v.pk) >= 2{
			compList = append(compList,CompList{maxPk: v.pk[1],num: 0,List: []int{v.pk[0],v.pk[1]}})
		}
	}

	return compList
}

func(this *MyTable) CompThreeOfAKindNoCombine(pk []int) []CompList{ //三条
	list := make([]int,len(pk))
	copy(list,pk)
	tmp := this.CalCardNum(list)
	compList := make([]CompList,0)
	for _,v := range tmp{
		if v.size >= 3 && len(v.pk) >= 3{
			compList = append(compList, CompList{maxPk: v.pk[2], num: 0, List: []int{v.pk[0],v.pk[1],v.pk[2]}})
		}
	}
	return compList
}
func(this *MyTable) CompFourOfAKindNoCombine(pk []int) []CompList{ //四条
	list := make([]int,len(pk))
	copy(list,pk)
	tmp := this.CalCardNum(list)
	compList := make([]CompList,0)
	for _,v := range tmp{
		if v.size >= 4 && len(v.pk) >= 4{
			combine := this.Combinations(v.pk,4)
			for _,v1 := range combine {
				compList = append(compList, CompList{maxPk: v1[3], num: 0, List: v1})
			}
		}
	}
	return compList
}
func(this *MyTable) CompStraightNoCombine(pk []int) []CompList{ //顺子
	list := make([]int,len(pk))
	copy(list,pk)
	tmp := this.CalCardNum(list)
	tlist := make([]struct{
		k int
		card []int
	},0)
	for _,v := range tmp{
		if v.mod % 0x10 != 0x0f && v.size >= 1 && len(v.pk) >= 1{
			tlist = append(tlist, struct {
				k    int
				card []int
			}{	k:v.mod , card:v.pk})
		}
	}

	sort.Slice(tlist, func(i, j int) bool { //升序排序
		return tlist[i].k < tlist[j].k
	})

	compList := make([]CompList,0)

	for i := 0;i < len(tlist);i++{
		s := tlist[i].k
		j := i + 1
		for ;j < len(tlist);{
			s1 := tlist[j].k
			if j - i == s1 - s{
				if j - i + 1 >= 3{ //三张以上才算顺子
					t := make([][]int,0)
					for n := i;n <= j;n++{
						t = append(t,tlist[n].card)
					}
					resList := make([]int,0)
					for k :=0;k < len(t);k++{
						resList = append(resList,t[k][0])
					}
					compList = append(compList, CompList{maxPk: t[len(t) - 1][0], num: j - i + 1, List: resList})
				}
				j += 1
			}else{
				break
			}
		}
	}

	return compList
}
func(this *MyTable) CompStraightPairNoCombine(pk []int) []CompList{ //连对
	list := make([]int,len(pk))
	copy(list,pk)
	tmp := this.CalCardNum(list)
	tlist := make([]struct{
		k int
		card []int
	},0)
	for _,v := range tmp{
		if v.mod % 0x10 != 0x0f && v.size >= 2 && len(v.pk) >= 2{
			tlist = append(tlist, struct {
				k    int
				card []int
			}{	k:v.mod , card:v.pk})
		}
	}

	sort.Slice(tlist, func(i, j int) bool { //升序排序
		return tlist[i].k < tlist[j].k
	})

	compList := make([]CompList,0)

	for i := 0;i < len(tlist);i++{
		s := tlist[i].k
		j := i + 1
		for ;j < len(tlist);{
			s1 := tlist[j].k
			if j - i == s1 - s{
				if j - i + 1 >= 3{ //至少三连对
					t := make([][]int,0)
					for n := i;n <= j;n++{
						t = append(t,tlist[n].card)
					}
					resList := make([]int,0)
					for k :=0;k < len(t);k++{
						resList = append(resList,t[k][0])
						resList = append(resList,t[k][1])
					}
					compList = append(compList, CompList{maxPk: t[len(t) - 1][1], num: j - i + 1, List: resList})
				}
				j += 1
			}else{
				break
			}
		}
	}

	return compList
}
func(this *MyTable) CheckOtherHaveBigger(idx int,pk []int) bool { //是否有大于该玩家出的牌
	for k,v := range this.PlayerList{
		if v.Ready && v.IsHavePeople && k != idx{
			list := this.CheckBiggerPoker(pk,v.HandPoker)
			if len(list) > 0{
				return true
			}
		}
	}
	return false
}
func (this *MyTable) GetCardValue(handPoker []int) int {
	//炸弹
	bomb := this.CompFourOfAKindNoCombine(handPoker)
	straightPair := this.CompStraightPairNoCombine(handPoker)
	value := 0
	if len(bomb) > 0 || len(straightPair) > 0{
		value += BombExtraValue
	}
	for _,v := range handPoker{
		value += v % 0x10
	}
	return value
}