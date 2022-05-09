package cardPhom

import "sort"

var card = []int{
	0x11,0x12,0x13,0x14,0x15,0x16,0x17,0x18,0x19,0x1a,0x1b,0x1c,0x1d, //黑桃
	0x21,0x22,0x23,0x24,0x25,0x26,0x27,0x28,0x29,0x2a,0x2b,0x2c,0x2d, //梅花
	0x31,0x32,0x33,0x34,0x35,0x36,0x37,0x38,0x39,0x3a,0x3b,0x3c,0x3d,  //方块
	0x41,0x42,0x43,0x44,0x45,0x46,0x47,0x48,0x49,0x4a,0x4b,0x4c,0x4d,  //红桃
}
var switchBackendCard = map[int]int{
	0x11:40,0x12:41,0x13:42,0x14:43,0x15:44,0x16:45,0x17:46,0x18:47,0x19:48,0x1a:49,0x1b:50,0x1c:51,0x1d:52,
	0x21:14,0x22:15,0x23:16,0x24:17,0x25:18,0x26:19,0x27:20,0x28:21,0x29:22,0x2a:23,0x2b:24,0x2c:25,0x2d:26,
	0x31:1,0x32:2,0x33:3,0x34:4,0x35:5,0x36:6,0x37:7,0x38:8,0x39:9,0x3a:10,0x3b:11,0x3c:12,0x3d:13,
	0x41:27,0x42:28,0x43:29,0x44:30,0x45:31,0x46:32,0x47:33,0x48:34,0x49:35,0x4a:36,0x4b:37,0x4c:38,0x4d:39,
}
type StraightType int //直接得分
const (
	NotStraight	StraightType = 1 //不靠张
	ThreePhom   StraightType = 2 //三道
)

type PokerType int //牌型
const (
	Single 		PokerType = 1 //单张
	Pair   		PokerType = 2 //对子
	ThreeOfAKind   	PokerType = 3 //三条
	Straight 	PokerType = 4 //顺子
	StraightPair3  PokerType = 5 //3连对
	Bomb	PokerType = 6 //炸弹
	StraightPair4  PokerType = 7 //4连对
)
type CardNum struct {
	size int
	pk []int
	mod int
}
func(this *MyTable) CalCardNum(pk []int) []CardNum{
	tmp := make([]CardNum,0)
	for _,v := range pk{
		mod := v % 0x10
		find := false
		for k1,v1 := range tmp{
			if v1.mod == mod{
				tmp[k1].size++
				tmp[k1].pk = append(tmp[k1].pk,v)
				find = true
				break
			}
		}
		if !find{
			tmp = append(tmp,CardNum{
				size: 1,
				pk: []int{v},
				mod: mod,
			})
		}
	}
	return tmp
}
func(this *MyTable) CheckEatPoker(handPk []int,forbidPk []int,mid int) [][]int{
	poker := this.FindNotInArrayList(forbidPk,handPk)
	tmp := this.CalCardNum(poker)
	mod := mid % 0x10
	eatPk := make([][]int,0)
	for _,v := range tmp{
		if v.mod == mod && v.size >= 2{
			eatPk = append(eatPk,v.pk)
			break
		}
	}

	poker = append(poker,mid)
	sort.Slice(poker, func(i, j int) bool { //升序排序
		return poker[i] < poker[j]
	})
	pkIdx := 0
	for k,v := range poker{
		if v == mid{
			pkIdx = k
			break
		}
	}
	straightPk := make([]int,0)
	for i := pkIdx - 1;i >= 0;i--{
		if mid - poker[i] == pkIdx - i && mid / 0x10 == poker[i] / 0x10{
			straightPk = append(straightPk,poker[i])
		}else{
			break
		}
	}
	for i := pkIdx + 1;i < len(poker);i++{
		if  poker[i] - mid == i - pkIdx && mid / 0x10 == poker[i] / 0x10{
			straightPk = append(straightPk,poker[i])
		}else{
			break
		}
	}
	if len(straightPk) >= 2{
		sort.Slice(straightPk, func(i, j int) bool { //升序排序
			return straightPk[i] < straightPk[j]
		})
		eatPk = append(eatPk,straightPk)
	}
	return eatPk
}
func(this *MyTable) CheckGivePokerFour(handPk []int,forbidPk []int,mid int) [][]int{
	poker := this.FindNotInArrayList(forbidPk,handPk)
	tmp := this.CalCardNum(poker)
	mod := mid % 0x10
	eatPk := make([][]int,0)
	for _,v := range tmp{
		if v.mod == mod && v.size >= 2{
			eatPk = append(eatPk,v.pk)
			break
		}
	}
	return eatPk
}
func(this *MyTable) CheckGivePokerStraight(handPk []int,forbidPk []int,mid int) [][]int{
	poker := this.FindNotInArrayList(forbidPk,handPk)
	eatPk := make([][]int,0)
	poker = append(poker,mid)
	sort.Slice(poker, func(i, j int) bool { //升序排序
		return poker[i] < poker[j]
	})
	pkIdx := 0
	for k,v := range poker{
		if v == mid{
			pkIdx = k
			break
		}
	}
	straightPk := make([]int,0)
	for i := pkIdx - 1;i >= 0;i--{
		if mid - poker[i] == pkIdx - i && mid / 0x10 == poker[i] / 0x10{
			straightPk = append(straightPk,poker[i])
		}else{
			break
		}
	}
	for i := pkIdx + 1;i < len(poker);i++{
		if  poker[i] - mid == i - pkIdx && mid / 0x10 == poker[i] / 0x10{
			straightPk = append(straightPk,poker[i])
		}else{
			break
		}
	}
	if len(straightPk) >= 2{
		sort.Slice(straightPk, func(i, j int) bool { //升序排序
			return straightPk[i] < straightPk[j]
		})
		eatPk = append(eatPk,straightPk)
	}
	return eatPk
}
func(this *MyTable) CheckNearGivePoker(handPk []int,forbidPk []int,mid int) []int{
	poker := this.FindNotInArrayList(forbidPk,handPk)
	sort.Slice(poker, func(i, j int) bool { //升序排序
		return poker[i] < poker[j]
	})
	idx := -1
	res := make([]int,0)
	for k,v := range poker{
		if v == mid{
			idx = k
			break
		}
	}
	for i := idx - 1;i >= 0;i--{
		if poker[idx] - poker[i] == idx - i{
			res = append(res,poker[i])
		}else{
			break
		}
	}
	for i := idx + 1;i < len(poker);i++{
		if poker[idx] - poker[i] == idx - i{
			res = append(res,poker[i])
		}else{
			break
		}
	}
	return res
}
func(this *MyTable) CompStraight(pk []int) [][]int{ //顺子
	list := make([]int,len(pk))
	copy(list,pk)
	sort.Slice(list, func(i, j int) bool { //升序排序
		return list[i] < list[j]
	})

	compList := make([][]int,0)

	for i := 0;i < len(list);i++{
		s := list[i]
		j := i + 1
		for ;j < len(list);{
			s1 := list[j]
			if j - i == s1 - s{
				if j - i + 1 >= 3{ //三张以上才算顺子
					t := make([]int,0)
					for n := i;n <= j;n++{
						t = append(t,list[n])
					}
					compList = append(compList, t)
				}
				j += 1
			}else{
				break
			}
		}
	}

	return compList
}
func(this *MyTable) CompThreeOfAKind(pk []int) [][]int{ //三条
	list := make([]int,len(pk))
	copy(list,pk)
	tmp := this.CalCardNum(list)
	compList := make([][]int,0)
	for _,v := range tmp{
		if v.size >= 3 && len(v.pk) >= 3{
			combine := this.Combinations(v.pk,3)
			for _,v1 := range combine {
				compList = append(compList,v1)
			}
		}
	}
	return compList
}
func(this *MyTable) CompFourOfAKind(pk []int) [][]int{ //四条
	list := make([]int,len(pk))
	copy(list,pk)
	tmp := this.CalCardNum(list)
	compList := make([][]int,0)
	for _,v := range tmp{
		if v.size >= 4 && len(v.pk) >= 4{
			combine := this.Combinations(v.pk,4)
			for _,v1 := range combine {
				compList = append(compList, v1)
			}
		}
	}
	return compList
}
func (this *MyTable) CalcRankList(){
	for k,v := range this.PlayerList{
		if v.Ready && v.CalcPhomData.State != MOM {
			sort.Slice(v.HandPoker, func(i, j int) bool { //升序排序
				if v.HandPoker[i] % 0x10 == v.HandPoker[j] % 0x10{
					return v.HandPoker[i] < v.HandPoker[j]
				}
				return v.HandPoker[i] % 0x10 < v.HandPoker[j] % 0x10
			})
			point := 0
			for _,v1 := range v.HandPoker{
				point += v1 % 0x10
			}

			pointV := point * 0xFFF + len(v.HandPoker) * 0xFF + v.HandPoker[len(v.HandPoker) - 1] % 0x10 * 0xF + v.HandPoker[len(v.HandPoker) - 1] / 0x10
			this.RankList = append(this.RankList,RankList{
				Idx: k,
				PointV: pointV,
				State: v.CalcPhomData.State,
			})
		}
	}
	for k,v := range this.PlayerList{
		if v.Ready && v.CalcPhomData.State == MOM {
			sort.Slice(v.HandPoker, func(i, j int) bool { //升序排序
				if v.HandPoker[i] % 0x10 == v.HandPoker[j] % 0x10{
					return v.HandPoker[i] < v.HandPoker[j]
				}
				return v.HandPoker[i] % 0x10 < v.HandPoker[j] % 0x10
			})
			point := 0
			for _,v1 := range v.HandPoker{
				point += v1 % 0x10
			}

			pointV := 0xFFFFFF + point * 0xFFF + len(v.HandPoker) * 0xFF + v.HandPoker[len(v.HandPoker) - 1] % 0x10 * 0xF + v.HandPoker[len(v.HandPoker) - 1] / 0x10
			this.RankList = append(this.RankList,RankList{
				Idx: k,
				PointV: pointV,
				State: v.CalcPhomData.State,
			})
		}
	}

	sort.Slice(this.RankList, func(i, j int) bool { //升序排序
		return this.RankList[i].PointV < this.RankList[j].PointV
	})

}