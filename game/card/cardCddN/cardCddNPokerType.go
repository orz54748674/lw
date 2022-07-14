package cardCddN

import (
	"sort"
)

var card = []int{
	0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, //黑桃
	0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f, //梅花
	0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f, //方块
	0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f, //红桃
}
var switchBackendCard = map[int]int{
	0x1e: 40, 0x1f: 41, 0x13: 42, 0x14: 43, 0x15: 44, 0x16: 45, 0x17: 46, 0x18: 47, 0x19: 48, 0x1a: 49, 0x1b: 50, 0x1c: 51, 0x1d: 52,
	0x2e: 14, 0x2f: 15, 0x23: 16, 0x24: 17, 0x25: 18, 0x26: 19, 0x27: 20, 0x28: 21, 0x29: 22, 0x2a: 23, 0x2b: 24, 0x2c: 25, 0x2d: 26,
	0x3e: 1, 0x3f: 2, 0x33: 3, 0x34: 4, 0x35: 5, 0x36: 6, 0x37: 7, 0x38: 8, 0x39: 9, 0x3a: 10, 0x3b: 11, 0x3c: 12, 0x3d: 13,
	0x4e: 27, 0x4f: 28, 0x43: 29, 0x44: 30, 0x45: 31, 0x46: 32, 0x47: 33, 0x48: 34, 0x49: 35, 0x4a: 36, 0x4b: 37, 0x4c: 38, 0x4d: 39,
}

type StraightType int //直接得分
const (
	Four3             StraightType = 1  //四个3
	Four2             StraightType = 2  //四个2
	TwoBomb           StraightType = 3  //2个炸弹
	SixPair           StraightType = 4  //6对
	ThreeStraightPair StraightType = 5  //3连对
	FourStraightPair  StraightType = 6  //4连对
	FiveStraightPair  StraightType = 7  //5连对
	SixStraightPair   StraightType = 8  //6连对
	SameColor         StraightType = 9  //清一色
	YiTiaoLong        StraightType = 10 //一条龙
	QingLong          StraightType = 11 //青龙
)

type PokerType int //牌型
const (
	ERROR         PokerType = -1 //错误类型
	Single        PokerType = 1  //单张
	Pair          PokerType = 2  //对子
	ThreeOfAKind  PokerType = 3  //三条
	Straight      PokerType = 4  //顺子
	StraightPair3 PokerType = 5  //3连对
	Bomb          PokerType = 6  //炸弹
	StraightPair4 PokerType = 7  //4连对
)

type CardNum struct {
	size int
	pk   []int
	mod  int
}

func (this *MyTable) CalCardNum(pk []int) []CardNum {
	tmp := make([]CardNum, 0)
	for _, v := range pk {
		mod := v % 0x10
		find := false
		for k1, v1 := range tmp {
			if v1.mod == mod {
				tmp[k1].size++
				tmp[k1].pk = append(tmp[k1].pk, v)
				find = true
				break
			}
		}
		if !find {
			tmp = append(tmp, CardNum{
				size: 1,
				pk:   []int{v},
				mod:  mod,
			})
		}
	}
	return tmp
}
func (this *MyTable) CheckQingLong(pk []int) bool {
	if len(pk) < 13 {
		return false
	}
	for i := 1; i < 13; i++ {
		if pk[i] != pk[i-1]+1 {
			return false
		}
	}
	return true
}
func (this *MyTable) CheckYiTiaoLong(modList []int) bool {
	if len(modList) < 13 {
		return false
	}
	for i := 1; i < 13; i++ {
		if modList[i] != modList[i-1]+1 {
			return false
		}
	}
	return true
}
func (this *MyTable) CheckSameColor(pk []int) bool {
	red, black := 0, 0
	for _, v := range pk {
		if v/0x10 == 1 || v/0x10 == 3 {
			red++
		} else {
			black++
		}
	}
	if red == 13 || black == 13 {
		return true
	}
	return false
}

type CompList struct {
	maxPk int
	num   int
	List  []int
}

func (this *MyTable) CompSingle(pk []int) []CompList { //单张
	list := make([]int, len(pk))
	copy(list, pk)
	tmp := this.CalCardNum(list)
	compList := make([]CompList, 0)
	for _, v := range tmp {
		if v.size >= 1 && len(v.pk) >= 1 {
			combine := this.Combinations(v.pk, 1)
			for _, v1 := range combine {
				compList = append(compList, CompList{maxPk: v1[0], num: 0, List: v1})
			}
		}
	}
	//for _,v := range tmp{
	//	if v.size == 1 && len(v.pk) == 1{
	//		combine := this.Combinations(v.pk,1)
	//		for _,v1 := range combine{
	//			compList = append(compList,CompList{maxPk: v1[0],num: 0,List: v1})
	//		}
	//	}
	//}
	//for _,v := range tmp{
	//	if v.size == 2 && len(v.pk) == 2{
	//		combine := this.Combinations(v.pk,1)
	//		for _,v1 := range combine{
	//			compList = append(compList,CompList{maxPk: v1[0],num: 0,List: v1})
	//		}
	//	}
	//}
	//for _,v := range tmp{
	//	if v.size == 3 && len(v.pk) == 3{
	//		combine := this.Combinations(v.pk,1)
	//		for _,v1 := range combine{
	//			compList = append(compList,CompList{maxPk: v1[0],num: 0,List: v1})
	//		}
	//	}
	//}
	//for _,v := range tmp{
	//	if v.size == 4 && len(v.pk) == 4{
	//		combine := this.Combinations(v.pk,1)
	//		for _,v1 := range combine{
	//			compList = append(compList,CompList{maxPk: v1[0],num: 0,List: v1})
	//		}
	//	}
	//}
	return compList
}
func (this *MyTable) CompPair(pk []int) []CompList {
	list := make([]int, len(pk))
	copy(list, pk)
	tmp := this.CalCardNum(list)
	compList := make([]CompList, 0)
	for _, v := range tmp {
		if v.size >= 2 && len(v.pk) >= 2 {
			combine := this.Combinations(v.pk, 2)
			for _, v1 := range combine {
				compList = append(compList, CompList{maxPk: v1[1], num: 0, List: v1})
			}
		}
	}
	//for _,v := range tmp{
	//	if v.size == 2 && len(v.pk) == 2{
	//		combine := this.Combinations(v.pk,2)
	//		for _,v1 := range combine{
	//			compList = append(compList,CompList{maxPk: v1[1],num: 0,List: v1})
	//		}
	//	}
	//}
	//for _,v := range tmp{
	//	if v.size == 3 && len(v.pk) == 3{
	//		combine := this.Combinations(v.pk,2)
	//		for _,v1 := range combine{
	//			compList = append(compList,CompList{maxPk: v1[1],num: 0,List: v1})
	//		}
	//	}
	//}
	//for _,v := range tmp{
	//	if v.size == 4 && len(v.pk) == 4{
	//		combine := this.Combinations(v.pk,2)
	//		for _,v1 := range combine{
	//			compList = append(compList,CompList{maxPk: v1[1],num: 0,List: v1})
	//		}
	//	}
	//}
	return compList
}

func (this *MyTable) CompThreeOfAKind(pk []int) []CompList { //三条
	list := make([]int, len(pk))
	copy(list, pk)
	tmp := this.CalCardNum(list)
	compList := make([]CompList, 0)
	for _, v := range tmp {
		if v.size >= 3 && len(v.pk) >= 3 {
			combine := this.Combinations(v.pk, 3)
			for _, v1 := range combine {
				compList = append(compList, CompList{maxPk: v1[2], num: 0, List: v1})
			}
		}
	}
	//for _,v := range tmp{
	//	if v.size == 3 && len(v.pk) == 3{
	//		combine := this.Combinations(v.pk,3)
	//		for _,v1 := range combine {
	//			compList = append(compList, CompList{maxPk: v1[2], num: 0, List: v1})
	//		}
	//	}
	//}
	//for _,v := range tmp{
	//	if v.size == 4 && len(v.pk) == 4{
	//		combine := this.Combinations(v.pk,3)
	//		for _,v1 := range combine {
	//			compList = append(compList, CompList{maxPk: v1[2], num: 0, List: v1})
	//		}
	//	}
	//}
	return compList
}
func (this *MyTable) CompFourOfAKind(pk []int) []CompList { //四条
	list := make([]int, len(pk))
	copy(list, pk)
	tmp := this.CalCardNum(list)
	compList := make([]CompList, 0)
	for _, v := range tmp {
		if v.size >= 4 && len(v.pk) >= 4 {
			combine := this.Combinations(v.pk, 4)
			for _, v1 := range combine {
				compList = append(compList, CompList{maxPk: v1[3], num: 0, List: v1})
			}
		}
	}
	return compList
}
func (this *MyTable) CompStraight(pk []int) []CompList { //顺子
	list := make([]int, len(pk))
	copy(list, pk)
	tmp := this.CalCardNum(list)
	tlist := make([]struct {
		k    int
		card []int
	}, 0)
	for _, v := range tmp {
		if v.mod%0x10 != 0x0f && v.size >= 1 && len(v.pk) >= 1 {
			tlist = append(tlist, struct {
				k    int
				card []int
			}{k: v.mod, card: v.pk})
		}
	}

	sort.Slice(tlist, func(i, j int) bool { //升序排序
		return tlist[i].k < tlist[j].k
	})

	compList := make([]CompList, 0)

	for i := 0; i < len(tlist); i++ {
		s := tlist[i].k
		j := i + 1
		for j < len(tlist) {
			s1 := tlist[j].k
			if j-i == s1-s {
				if j-i+1 >= 3 { //三张以上才算顺子
					t := make([][]int, 0)
					for n := i; n <= j; n++ {
						t = append(t, tlist[n].card)
					}
					pk := this.Cartesian(t)
					for _, v := range pk {
						compList = append(compList, CompList{maxPk: v[len(v)-1], num: j - i + 1, List: v})
					}
				}
				j += 1
			} else {
				break
			}
		}
	}

	return compList
}
func (this *MyTable) CompStraightPair(pk []int) []CompList { //连对
	list := make([]int, len(pk))
	copy(list, pk)
	tmp := this.CalCardNum(list)
	tlist := make([]struct {
		k    int
		card []int
	}, 0)
	for _, v := range tmp {
		if v.mod%0x10 != 0x0f && v.size >= 2 && len(v.pk) >= 2 {
			tlist = append(tlist, struct {
				k    int
				card []int
			}{k: v.mod, card: v.pk})
		}
	}

	sort.Slice(tlist, func(i, j int) bool { //升序排序
		return tlist[i].k < tlist[j].k
	})

	compList := make([]CompList, 0)

	for i := 0; i < len(tlist); i++ {
		s := tlist[i].k
		j := i + 1
		for j < len(tlist) {
			s1 := tlist[j].k
			if j-i == s1-s {
				if j-i+1 >= 3 { //至少三连对
					t := make([][][]int, 0)
					for n := i; n <= j; n++ {
						comb := this.Combinations(tlist[n].card, 2)
						t = append(t, comb)
					}
					pk := this.Cartesian2(t)
					for _, v := range pk {
						maxPk := 0
						list := make([]int, 0)
						for _, v1 := range v {
							for _, v2 := range v1 {
								list = append(list, v2)
								if v2%0x10 > maxPk%0x10 {
									maxPk = v2
								}
							}
						}
						compList = append(compList, CompList{maxPk: maxPk, num: j - i + 1, List: list})
					}
				}
				j += 1
			} else {
				break
			}
		}
	}

	return compList
}
func (this *MyTable) CheckStraightPair(pk []int, num int, black3 bool) CompList {
	ldList := this.CompStraightPair(pk)
	for _, v := range ldList {
		if v.num >= num {
			if black3 {
				for _, v1 := range v.List {
					if v1 == 0x13 {
						return v
					}
				}
			} else {
				return v
			}
		}
	}

	return CompList{}
}
func (this *MyTable) CheckPair6(modList []int) bool {
	if len(modList) < 13 {
		return false
	}
	num1 := 0
	num2 := 0
	len := 1
	for i := 1; i < 14; i++ {
		if i == 13 || modList[i] != modList[i-1] {
			if len == 1 {
				num1 += 1
				len = 1
				if num1 > 1 {
					break
				}
			} else if len == 2 {
				num2 += 1
				len = 1
			} else if len == 3 {
				num1 += 1
				num2 += 1
				len = 1
				if num1 > 1 {
					break
				}
			} else if len == 4 {
				num2 += 2
				len = 1
			}
		} else {
			len++
		}
	}
	if num1 == 1 && num2 == 6 {
		return true
	}
	return false
}
func (this *MyTable) CheckTwoBomb(pk []int) int {
	list := make([]int, len(pk))
	copy(list, pk)
	tmp := this.CalCardNum(list)
	bombNum := 0
	maxPk := 0
	for _, v := range tmp {
		if v.size >= 4 {
			bombNum++
			maxPk = v.pk[len(v.pk)-1]
		}
	}
	if bombNum >= 2 {
		return maxPk
	}
	return 0
}
func (this *MyTable) CheckIsBomb(modList []int, mod int) bool {
	num := 0
	for _, v := range modList {
		if v == mod {
			num++
		}
	}
	if num == 4 {
		return true
	}
	return false
}
func (this *MyTable) CheckStraightScore(pk []int) (StraightType, int) {
	cardList := make([]int, len(pk))
	copy(cardList, pk)
	sort.Slice(cardList, func(i, j int) bool { //升序排序
		if cardList[i]%0x10 == cardList[j]%0x10 {
			return cardList[i] < cardList[j]
		}
		return cardList[i]%0x10 < cardList[j]%0x10
	})
	modList := make([]int, len(cardList))
	color := make([]int, len(cardList))
	mapList := map[int]int{}
	for i := len(cardList) - 1; i >= 0; i-- {
		modList[i] = cardList[i] % 0x10
		color[i] = cardList[i] / 0x10
		mapList[modList[i]] += 1
	}
	if this.CheckQingLong(cardList) {
		maxPk := cardList[len(cardList)-1]
		maxVal := int(QingLong)*0xFFFF + maxPk%0x10*0xFF + maxPk/0x10
		return QingLong, maxVal
	} else if this.CheckYiTiaoLong(modList) {
		maxPk := cardList[len(cardList)-1]
		maxVal := int(YiTiaoLong)*0xFFFF + maxPk%0x10*0xFF + maxPk/0x10
		return YiTiaoLong, maxVal
	} else if this.CheckSameColor(cardList) {
		maxPk := cardList[len(cardList)-1]
		maxVal := int(SameColor)*0xFFFF + maxPk%0x10*0xFF + maxPk/0x10
		return SameColor, maxVal
	}
	compList := this.CheckStraightPair(cardList, 6, false)
	if len(compList.List) > 0 {
		maxPk := compList.maxPk
		maxVal := int(SixStraightPair)*0xFFFF + maxPk%0x10*0xFF + maxPk/0x10
		return SixStraightPair, maxVal
	}
	compList = this.CheckStraightPair(cardList, 5, false)
	if len(compList.List) > 0 {
		maxPk := compList.maxPk
		maxVal := int(FiveStraightPair)*0xFFFF + maxPk%0x10*0xFF + maxPk/0x10
		return FiveStraightPair, maxVal
	}
	compList = this.CheckStraightPair(cardList, 4, true)
	if len(compList.List) > 0 {
		maxPk := compList.maxPk
		maxVal := int(FourStraightPair)*0xFFFF + maxPk%0x10*0xFF + maxPk/0x10
		return FourStraightPair, maxVal
	}
	compList = this.CheckStraightPair(cardList, 3, true)
	if len(compList.List) > 0 {
		maxPk := compList.maxPk
		maxVal := int(ThreeStraightPair)*0xFFFF + maxPk%0x10*0xFF + maxPk/0x10
		return ThreeStraightPair, maxVal
	}
	if this.CheckPair6(modList) {
		tmp := this.CalCardNum(cardList)
		lastPk := cardList[len(cardList)-1]
		maxPk := 0
		for _, v := range tmp {
			if v.mod == lastPk%0x10 && v.size == 1 {
				maxPk = cardList[len(cardList)-2]
			} else {
				maxPk = lastPk
			}
		}
		maxVal := int(SixPair)*0xFFFF + maxPk%0x10*0xFF + maxPk/0x10
		return SixPair, maxVal
	}

	maxPk := this.CheckTwoBomb(cardList)
	if maxPk > 0 {
		maxVal := int(TwoBomb)*0xFFFF + maxPk%0x10*0xFF + maxPk/0x10
		return TwoBomb, maxVal
	}

	if this.CheckIsBomb(modList, 0x0f) {
		maxPk := 0x4f
		maxVal := int(Four2)*0xFFFF + maxPk%0x10*0xFF + maxPk/0x10
		return Four2, maxVal
	} else if this.CheckIsBomb(modList, 0x03) {
		maxPk := 0x43
		maxVal := int(Four3)*0xFFFF + maxPk%0x10*0xFF + maxPk/0x10
		return Four3, maxVal
	}
	return StraightType(0), 0
}

func (this *MyTable) CheckFourStraightPair(pkList []int) (bool, int) {
	if len(pkList) != 8 {
		return false, 0
	}
	compList := this.CompStraightPair(pkList)
	for _, v := range compList {
		if v.num == 4 {
			return true, pkList[len(pkList)-1]
		}
	}
	return false, pkList[len(pkList)-1]
}
func (this *MyTable) CheckThreeStraightPair(pkList []int) (bool, int) {
	if len(pkList) != 6 {
		return false, 0
	}
	compList := this.CompStraightPair(pkList)
	for _, v := range compList {
		if v.num == 3 {
			return true, pkList[len(pkList)-1]
		}
	}
	return false, pkList[len(pkList)-1]
}
func (this *MyTable) CheckBomb(pkList []int, modList []int) (bool, int) {
	if len(pkList) != 4 {
		return false, 0
	}
	if modList[0] == modList[1] && modList[0] == modList[2] && modList[0] == modList[3] {
		return true, pkList[len(pkList)-1]
	}
	return false, 0
}
func (this *MyTable) CheckStraight(pkList []int, modList []int) (bool, int) {
	if len(pkList) < 3 {
		return false, 0
	}
	for i := 0; i < len(modList)-1; i++ {
		if modList[i] != modList[i+1]-1 || modList[i] == 0x0f || modList[i+1] == 0x0f {
			return false, 0
		}
	}
	return true, pkList[len(pkList)-1]
}
func (this *MyTable) CheckThreeOfAKind(pkList []int, modList []int) (bool, int) {
	if len(pkList) != 3 {
		return false, 0
	}
	if modList[0] == modList[1] && modList[0] == modList[2] {
		return true, pkList[len(pkList)-1]
	}
	return false, 0
}
func (this *MyTable) CheckPair(pkList []int, modList []int) (bool, int) {
	if len(pkList) != 2 {
		return false, 0
	}
	if modList[0] == modList[1] {
		return true, pkList[len(pkList)-1]
	}
	return false, 0
}
func (this *MyTable) CheckSingle(pkList []int, modList []int) (bool, int) {
	if len(pkList) != 1 {
		return false, 0
	}
	return true, pkList[0]
}
func (this *MyTable) GetCardType(pk []int) (pkType PokerType, max int, maxV int, num int) { //
	pkList := make([]int, len(pk))
	copy(pkList, pk)
	modList := make([]int, len(pk))
	for i := len(pkList) - 1; i >= 0; i-- {
		modList[i] = pk[i] % 0x10
	}
	sort.Slice(modList, func(i, j int) bool { //升序排序
		return modList[i] < modList[j]
	})
	sort.Slice(pkList, func(i, j int) bool { //升序排序
		_maxI := pkList[i]%0x10*0xFF + pkList[i]/0x10
		_maxJ := pkList[j]%0x10*0xFF + pkList[j]/0x10
		return _maxI < _maxJ
	})

	ok, _max := this.CheckFourStraightPair(pkList)
	if ok {
		_maxV := _max%0x10*0xFF + _max/0x10
		return StraightPair4, _max, _maxV, 4
	}
	ok, _max = this.CheckThreeStraightPair(pkList)
	if ok {
		_maxV := _max%0x10*0xFF + _max/0x10
		return StraightPair3, _max, _maxV, 3
	}
	ok, _max = this.CheckBomb(pkList, modList)
	if ok {
		_maxV := _max%0x10*0xFF + _max/0x10
		return Bomb, _max, _maxV, 0
	}
	ok, _max = this.CheckStraight(pkList, modList)
	if ok {
		_maxV := _max%0x10*0xFF + _max/0x10
		return Straight, _max, _maxV, len(pk)
	}
	ok, _max = this.CheckThreeOfAKind(pkList, modList)
	if ok {
		_maxV := _max%0x10*0xFF + _max/0x10
		return ThreeOfAKind, _max, _maxV, len(pk)
	}
	ok, _max = this.CheckPair(pkList, modList)
	if ok {
		_maxV := _max%0x10*0xFF + _max/0x10
		return Pair, _max, _maxV, len(pk)
	}
	ok, _max = this.CheckSingle(pkList, modList)
	if ok {
		_maxV := _max%0x10*0xFF + _max/0x10
		return Single, _max, _maxV, len(pk)
	}
	return PokerType(0), 0, 0, 0
}
func (this *MyTable) CompBigger(last []int, now []int) bool { //
	lastPk := make([]int, len(last))
	copy(lastPk, last)
	nowPk := make([]int, len(now))
	copy(nowPk, now)

	lastPkType, lastMax, lastMaxV, lastNum := this.GetCardType(lastPk)
	nowPkType, _, nowMaxV, nowNum := this.GetCardType(nowPk)

	if lastPkType == Single { //单牌
		if nowPkType == Single {
			if nowMaxV > lastMaxV {
				return true
			} else {
				return false
			}
		} else if lastMax%0x10 == 0x0F && (nowPkType == Bomb || nowPkType == StraightPair3 || nowPkType == StraightPair4) {
			return true
		} else {
			return false
		}
	} else if lastPkType == Pair { //对子
		if nowPkType == Pair {
			if nowMaxV > lastMaxV {
				return true
			} else {
				return false
			}
		} else if lastMax%0x10 == 0x0F && (nowPkType == Bomb || nowPkType == StraightPair4) {
			return true
		} else {
			return false
		}
	} else if lastPkType == ThreeOfAKind { //三条
		if nowPkType == ThreeOfAKind {
			if nowMaxV > lastMaxV {
				return true
			} else {
				return false
			}
		} else {
			return false
		}
	} else if lastPkType == Straight { //顺子
		if nowPkType == Straight && lastNum == nowNum {
			if nowMaxV > lastMaxV {
				return true
			} else {
				return false
			}
		} else {
			return false
		}
	} else if lastPkType == StraightPair3 { //三连对
		if nowPkType == StraightPair3 {
			if nowMaxV > lastMaxV {
				return true
			} else {
				return false
			}
		} else if nowPkType == Bomb || nowPkType == StraightPair4 {
			return true
		} else {
			return false
		}
	} else if lastPkType == Bomb { //炸弹
		if nowPkType == Bomb {
			if nowMaxV > lastMaxV {
				return true
			} else {
				return false
			}
		} else if nowPkType == StraightPair4 {
			return true
		} else {
			return false
		}
	} else if lastPkType == StraightPair4 { //四连对
		if nowPkType == StraightPair4 {
			if nowMaxV > lastMaxV {
				return true
			} else {
				return false
			}
		} else {
			return false
		}
	}
	return false
}
func (this *MyTable) CheckBiggerPoker(last []int, now []int) [][]int { //
	lastPk := make([]int, len(last))
	copy(lastPk, last)
	nowPk := make([]int, len(now))
	copy(nowPk, now)
	bigPk := make([][]int, 0)
	var compList []CompList

	lastPkType, lastMax, lastMaxV, lastNum := this.GetCardType(lastPk)
	if lastPkType == Single { //单牌
		compList = this.CompSingle(nowPk)
		for _, v := range compList {
			maxPkV := v.maxPk%0x10*0xFF + v.maxPk/0x10
			if maxPkV > lastMaxV {
				bigPk = append(bigPk, v.List)
			}
		}
		if lastMax%0x10 == 0x0f { //2
			compList = this.CompFourOfAKind(nowPk)
			for _, v := range compList {
				bigPk = append(bigPk, v.List)
			}

			compList = this.CompStraightPair(nowPk)
			for _, v := range compList {
				bigPk = append(bigPk, v.List)
			}
		}
	}

	if lastPkType == Pair { //对子
		compList = this.CompPair(nowPk)
		for _, v := range compList {
			maxPkV := v.maxPk%0x10*0xFF + v.maxPk/0x10
			if maxPkV > lastMaxV {
				bigPk = append(bigPk, v.List)
			}
		}
		if lastMax%0x10 == 0x0f { //2
			compList = this.CompFourOfAKind(nowPk)
			for _, v := range compList {
				bigPk = append(bigPk, v.List)
			}

			compList = this.CompStraightPair(nowPk)
			for _, v := range compList {
				if v.num > 3 { //四连队才能压
					bigPk = append(bigPk, v.List)
				}
			}
		}
	}

	if lastPkType == Straight { //顺子
		compList = this.CompStraight(nowPk)
		for _, v := range compList {
			if v.num == lastNum {
				maxPkV := v.maxPk%0x10*0xFF + v.maxPk/0x10
				if maxPkV > lastMaxV {
					bigPk = append(bigPk, v.List)
				}
			}
		}
	}

	if lastPkType == ThreeOfAKind { //三条
		compList = this.CompThreeOfAKind(nowPk)
		for _, v := range compList {
			maxPkV := v.maxPk%0x10*0xFF + v.maxPk/0x10
			if maxPkV > lastMaxV {
				bigPk = append(bigPk, v.List)
			}
		}
	}

	if lastPkType == StraightPair3 { //三连对
		compList = this.CompStraightPair(nowPk)
		for _, v := range compList {
			maxPkV := v.maxPk%0x10*0xFF + v.maxPk/0x10
			if v.num > lastNum || maxPkV > lastMaxV {
				bigPk = append(bigPk, v.List)
			}
		}
		compList = this.CompFourOfAKind(nowPk)
		for _, v := range compList {
			bigPk = append(bigPk, v.List)
		}

	}
	if lastPkType == Bomb { //炸弹
		compList = this.CompFourOfAKind(nowPk)
		for _, v := range compList {
			maxPkV := v.maxPk%0x10*0xFF + v.maxPk/0x10
			if maxPkV > lastMaxV {
				bigPk = append(bigPk, v.List)
			}
		}

		compList = this.CompStraightPair(nowPk)
		for _, v := range compList {
			if v.num > 3 { //四连队才能压
				bigPk = append(bigPk, v.List)
			}
		}
	}

	if lastPkType == StraightPair4 { //四连对
		compList = this.CompStraightPair(nowPk)
		for _, v := range compList {
			maxPkV := v.maxPk%0x10*0xFF + v.maxPk/0x10
			if v.num > 3 && maxPkV > lastMaxV {
				bigPk = append(bigPk, v.List)
			}
		}
	}

	return bigPk
}
