package cardCatte

import (
	"sort"
)

var card = []int{
	0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, //黑桃
	0x22, 0x23, 0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, //梅花
	0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, //方块
	0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, //红桃
}
var switchBackendCard = map[int]int{
	0x1e: 40, 0x12: 41, 0x13: 42, 0x14: 43, 0x15: 44, 0x16: 45, 0x17: 46, 0x18: 47, 0x19: 48, 0x1a: 49, 0x1b: 50, 0x1c: 51, 0x1d: 52,
	0x2e: 14, 0x22: 15, 0x23: 16, 0x24: 17, 0x25: 18, 0x26: 19, 0x27: 20, 0x28: 21, 0x29: 22, 0x2a: 23, 0x2b: 24, 0x2c: 25, 0x2d: 26,
	0x3e: 1, 0x32: 2, 0x33: 3, 0x34: 4, 0x35: 5, 0x36: 6, 0x37: 7, 0x38: 8, 0x39: 9, 0x3a: 10, 0x3b: 11, 0x3c: 12, 0x3d: 13,
	0x4e: 27, 0x42: 28, 0x43: 29, 0x44: 30, 0x45: 31, 0x46: 32, 0x47: 33, 0x48: 34, 0x49: 35, 0x4a: 36, 0x4b: 37, 0x4c: 38, 0x4d: 39,
}

type StraightType int //直接得分
const (
	AllLessSix StraightType = 1 //两张牌小于6
	Flush      StraightType = 2 //同花
	Four       StraightType = 3 //四个
)

type AnimationType int //特效类型
const (
	WinAllLessSix AnimationType = 1 //直赢 全部小于6
	WinFlush      AnimationType = 2 //直赢 同花
	WinFour       AnimationType = 3 //直赢 炸弹
	WinNormal     AnimationType = 4 //正常赢
	LoserHaveA    AnimationType = 5 //输有A
	LoserAllCheck AnimationType = 6 //输全盖
	LoserNormal   AnimationType = 7 //普通输
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

type CompList struct {
	maxPk int
	num   int
	List  []int
}

func (this *MyTable) CompFourOfAKind(pk []int) CompList { //四条
	list := make([]int, len(pk))
	copy(list, pk)
	tmp := this.CalCardNum(list)
	for _, v := range tmp {
		if v.size >= 4 && len(v.pk) >= 4 {
			combine := this.Combinations(v.pk, 4)
			for _, v1 := range combine {
				return CompList{maxPk: v1[3], num: 0, List: v1}
			}
		}
	}
	return CompList{}
}

func (this *MyTable) CheckStraightScore(pk []int) (StraightType, int) {
	if len(pk) != 6 {
		return StraightType(0), 0
	}
	cardList := make([]int, len(pk))
	copy(cardList, pk)
	sort.Slice(cardList, func(i, j int) bool { //升序排序
		return cardList[i] < cardList[j]
	})
	modList := make([]int, len(cardList))
	color := make([]int, len(cardList))
	mapList := map[int]int{}
	for i := len(cardList) - 1; i >= 0; i-- {
		modList[i] = cardList[i] % 0x10
		color[i] = cardList[i] / 0x10
		mapList[modList[i]] += 1
	}

	bomb := this.CompFourOfAKind(cardList)
	if len(bomb.List) > 0 {
		maxPk := cardList[len(cardList)-1]
		maxVal := int(Four)*0xFFFF + maxPk%0x10*0xFF
		return Four, maxVal
	}

	if cardList[0]/0x10 == cardList[5]/0x10 {
		maxPk := cardList[5]
		maxVal := int(Flush)*0xFFFF + maxPk/0x10*0xFF
		return Flush, maxVal
	}

	find := false
	for _, v := range cardList {
		if v%0x10 >= 6 {
			find = true
			break
		}
	}
	if !find {
		maxPk := cardList[5]
		maxVal := int(AllLessSix)*0xFFFF - maxPk%0x10*0xFF - maxPk/0x10
		return AllLessSix, maxVal
	}
	return StraightType(0), 0
}

func (this *MyTable) CompBigger(last []int, now []int) bool { //
	lastPk := make([]int, len(last))
	copy(lastPk, last)
	nowPk := make([]int, len(now))
	copy(nowPk, now)

	return false
}
func (this *MyTable) CheckBiggerPoker(last int, now []int) []int { //
	nowPk := make([]int, len(now))
	copy(nowPk, now)
	bigPk := make([]int, 0)

	for _, v := range nowPk {
		if v/0x10 == last/0x10 && v > last {
			bigPk = append(bigPk, v)
		}
	}
	return bigPk
}
