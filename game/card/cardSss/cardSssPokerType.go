package cardSss

import (
	"sort"
)
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
const TypeVal = 0xFFFFFF
const (
	QingLong	StraightType = 1 //青龙
	YiTiaoLong  StraightType = 2 //一条龙
	Pair5Three1  StraightType = 3 //五对加三条
	Flush3   StraightType = 4 //三同花
	Straight3 StraightType = 5  //三顺子
	StraightFlush3 StraightType = 6  //三同花顺
	Pair6 StraightType = 7  //6对
	SameColor   StraightType = 8 //清一色
)

type PokerType int //牌型
const (
	WuLong 		PokerType = 0 //乌龙
	Single 		PokerType = 1 //单张
	Pair   		PokerType = 2 //对子
	TwoPair   	PokerType = 3 //两对
	ThreeOfAKind   	PokerType = 4 //三条
	Straight   	PokerType = 5 //顺子
	Flush		PokerType = 6 //同花
	FullHouse	PokerType = 7 //葫芦
	FourOfAKind	PokerType = 8 //四条
	StraightFlush	PokerType = 9 //同花顺
	BigStraightFlush	PokerType = 10 //大同花顺
)
func(this *MyTable) CheckQingLong(pk []int) bool{
	for i := 1;i < 13;i++{
		if pk[i] != pk[i - 1] + 1{
			return false
		}
	}
	return true
}
func(this *MyTable) CheckYiTiaoLong(modList []int) bool{
	for i := 1;i < 13;i++{
		if modList[i] != modList[i - 1] + 1{
			return false
		}
	}
	return true
}
func(this *MyTable) CheckSameColor(pk []int) bool{
	red,black := 0,0
	for _,v := range pk {
		if v / 0x10 == 3 || v / 0x10 == 4{
			red++
		}else{
			black++
		}
	}
	if red == 13 || black == 13{
		return true
	}
	return false
}
func(this *MyTable) CheckStraightFlush3(list []int,sanDao bool,pos int) bool{
	if pos >= 13{
		return true
	}

	if sanDao{
		maxPos := pos + 5
		for i := pos + 1;i < maxPos;i++{
			if list[i] != list[i - 1] + 1{
				if !(list[i - 1] % 0x10 == 5 && list[i] % 0x10 == 14) || (list[i] / 0x10 != list[i -1] / 0x10){
					return false
				}
			}
		}
		return this.CheckStraightFlush3(list,sanDao,pos + 5)
	}else{
		flag := true
		maxPos := pos + 5
		for i := pos + 1;i < maxPos;i++{
			if list[i] != list[i - 1] + 1{
				if !(list[i - 1] % 0x10 == 5 && list[i] % 0x10 == 14) || (list[i] / 0x10 != list[i -1] / 0x10){
					flag = false
					break
				}
			}
		}

		if flag{
			return this.CheckStraightFlush3(list,sanDao,pos + 5)
		}

		maxPos = pos + 2
		for i := pos + 1;i < maxPos;i++{
			if list[i] != list[i - 1] + 1{
				if !(list[i - 1] % 0x10 == 5 && list[i] % 0x10 == 14) || (list[i] / 0x10 != list[i -1] / 0x10){
					return false
				}
			}
		}
		return this.CheckStraightFlush3(list,true,pos + 3)
	}
}
func(this *MyTable) CheckPair5Three1(modList []int) bool{
	num1 := 0
	num3 := 0
	num4 := 0
	len := 0
	for i := 1;i < 14;i++{
		if i == 13 || modList[i] != modList[i -1]{
			if len == 1{
				num1 += 1
				len = 0
			}else if len == 2{
				num3 += 1
				len = 0
			}else if len == 3{
				num4 += 1
				len = 0
			}else if len == 4{
				num1 += 1
				num3 += 1
				len = 0
			}else{
				break
			}
		}else{
			len += 1
		}
	}
	if num4 > 0{
		num1 += num4 * 2
	}

	if num1 == 5 && num3 == 1{
		return true
	}
	return false
}
func(this *MyTable) CheckFlush3(color []int) bool{
	colorNum := map[int]int{
		1:0,
		2:0,
		3:0,
		4:0,
	}
	for i := 0;i < 13;i++{
		colorNum[color[i]]++
	}
	for _,v := range colorNum{
		if v != 0 && v != 3 && v != 5 && v != 8 && v != 10{
			return false
		}
	}
	return true
}
func(this *MyTable) CheckStraight3(mapList map[int]int,sanDao bool,cnt int) bool{
	if cnt <= 0{
		return true
	}
	if !sanDao{
		aFlag := false
		start := 0
		len := 0
		for i := 2;i < 15;i++{
			if mapList[i] > 0{
				if start == 0{
					if i == 2 && mapList[14] > 0{
						mapList[14] -= 1
						len += 1
						aFlag = true
					}
					start = i
				}
				len += 1
				mapList[i] -= 1
			}else if start > 0{
				break
			}
			if len == 3{
				ret := this.CheckStraight3(mapList,true,cnt -3)
				if ret{
					return true
				}
			}

			if len == 4 && aFlag{
				aFlag = false
				len -= 1
				mapList[14] += 1
				ret := this.CheckStraight3(mapList,true,cnt -3)
				if ret{
					return true
				}
			}
		}
		if aFlag{
			mapList[14] += 1
			len -= 1
		}
		if start > 0{
			pos := start + len
			for i := start;i < pos;i++{
				mapList[i] += 1
			}
		}
	}
	aFlag := false
	start := 0
	len := 0
	for i := 2;i < 15;i++{
		if mapList[i] > 0{
			if start == 0{
				if i == 2 && mapList[14] > 0{
					mapList[14] -= 1
					len += 1
					aFlag = true
				}
				start = i
			}
			len += 1
			mapList[i] -= 1
		}else if start > 0{
			break
		}

		if len == 5{
			ret := this.CheckStraight3(mapList,sanDao,cnt -5)
			if ret{
				return true
			}
		}

		if len == 6 && aFlag{
			aFlag = false
			len -= 1
			mapList[14] += 1
			ret := this.CheckStraight3(mapList,sanDao,cnt - 5)
			if ret{
				return true
			}
		}
	}
	if aFlag{
		mapList[14] += 1
		len -= 1
	}
	if start > 0{
		pos := start + len
		for i := start;i < pos;i++ {
			mapList[i] += 1
		}
	}
	return false
}

func(this *MyTable) CheckPair6(modList []int) bool{
	num1 := 0
	num2 := 0
	len := 1
	for i := 1;i < 14;i++{
		if i == 13 || modList[i] != modList[i - 1] {
			if len == 1{
				num1 += 1
				len = 1
				if num1 > 1{
					break
				}
			}else if len == 2{
				num2 += 1
				len =1
			}else if len == 3{
				num1 += 1
				num2 += 1
				len = 1
				if num1 > 1{
					break
				}
			}else if len == 4{
				num2 += 2
				len = 1
			}
		} else{
			len += 1
		}
	}
	if num1 == 1 && num2 == 6{
		return true
	}
	return false
}
func(this *MyTable) CheckStraightScore(pk []int) StraightType{
	cardList := make([]int,len(pk))
	copy(cardList,pk)
	for i := 0;i < len(cardList);i++{
		if cardList[i] % 0x10 == 1{
			cardList[i] += 13
		}
	}

	sort.Slice(cardList, func(i, j int) bool { //升序排序
		if cardList[i] % 0x10 == cardList[j] % 0x10{
			return cardList[i] < cardList[j]
		}
		return cardList[i] % 0x10 < cardList[j] % 0x10
	})
	modList := make([]int,len(cardList))
	color := make([]int,len(cardList))
	mapList := map[int]int{}
	for i := len(cardList) -1;i >= 0;i--{
		modList[i] = cardList[i] % 0x10
		color[i] = cardList[i] / 0x10
		mapList[modList[i]] += 1
	}

	cardList2 := make([]int,len(pk))
	copy(cardList2,pk)
	for i := 0;i < len(cardList2);i++{
		if cardList2[i] % 0x10 == 1{
			cardList2[i] += 13
		}
	}

	sort.Slice(cardList2, func(i, j int) bool { //升序排序
		return cardList2[i] < cardList2[j]
	})
	if this.CheckQingLong(cardList){
		return QingLong
	}else if this.CheckYiTiaoLong(modList){
		return YiTiaoLong
	}else if this.CheckSameColor(cardList){
		return SameColor
	}else if this.CheckPair5Three1(modList){
		return Pair5Three1
	}else if this.CheckStraightFlush3(cardList2,false,0){
		return StraightFlush3
	}else if this.CheckFlush3(color){
		return Flush3
	}else if this.CheckStraight3(mapList,false,13){
		return Straight3
	}else if this.CheckPair6(modList){
		return Pair6
	}
	return StraightType(0)
}

func(this *MyTable) CheckPokerType(pk []int) (pkType PokerType,val int){
	if len(pk) != 5 && len(pk) != 3{
		return PokerType(-1),-1
	}
	cardList := make([]int,len(pk))
	copy(cardList,pk)
	for i := 0;i < len(cardList);i++{
		if cardList[i] % 0x10 == 1{
			cardList[i] += 13
		}
	}

	sort.Slice(cardList, func(i, j int) bool { //升序排序
		if cardList[i] % 0x10 == cardList[j] % 0x10{
			return cardList[i] < cardList[j]
		}
		return cardList[i] % 0x10 < cardList[j] % 0x10
	})
	modList := make([]int,len(cardList))
	color := make([]int,len(cardList))
	mapList := map[int]int{}
	allVal := 0
	for i := len(cardList) -1;i >= 0;i--{
		modList[i] = cardList[i] % 0x10
		color[i] = cardList[i] / 0x10
		mapList[modList[i]] += 1
		allVal += modList[i]
	}
	if len(cardList) == 3{
		if modList[0] == modList[2]{
			return ThreeOfAKind,int(ThreeOfAKind) * TypeVal + modList[1] * 0xFFFFF
		}
		if modList[0] == modList[1] {
			return Pair,int(Pair) * TypeVal + modList[1] * 0xFFFFF + modList[2] * 0xFFFF + color[1]
		}
		if modList[1] == modList[2]{
			return Pair,int(Pair) * TypeVal + modList[2] * 0xFFFFF + modList[0] * 0xFFFF + color[2]
		}
		return Single,int(Single) * TypeVal + modList[2] * 0xFFFFF + modList[1] * 0xFFFF + modList[0] * 0xFFF + color[2]
	}

	tp := StraightFlush
	for i := 1;i < len(cardList);i++{
		if cardList[i] != cardList[i - 1] + 1{
			if (modList[i - 1] != 5 || modList[i] != modList[i - 1] + 9) || (cardList[i] / 0x10  != cardList[i -1] / 0x10 ){
				tp = PokerType(-1)
				break
			}
		}
	}
	if tp == StraightFlush{
		for _,v := range modList{
			if v == 0x0E{
				return BigStraightFlush,int(BigStraightFlush) * TypeVal + modList[4] * 0xFFFFF + allVal * 0xF + color[4]
			}
		}
		return StraightFlush,int(StraightFlush) * TypeVal + modList[4] * 0xFFFFF + allVal * 0xF + color[4]
	}
	if modList[0] == modList[3] || modList[1] == modList[4]{
		return FourOfAKind,int(FourOfAKind) * TypeVal + modList[3] * 0xFFFFF
	}
	if modList[0] == modList[2] && modList[3] == modList[4]{
		return FullHouse,int(FullHouse) * TypeVal + modList[0] * 0xFFFFF
	}else if modList[2] == modList[4] && modList[0] == modList[1]{
		return FullHouse,int(FullHouse) * TypeVal + modList[2] * 0xFFFFF
	}

	tp = Flush
	for i := 1;i < len(color);i++ {
		if color[i] != color[i -1]{
			tp = PokerType(-1)
			break
		}
	}
	if tp == Flush{
		return Flush,int(Flush) * TypeVal + modList[4] * 0xFFFFF + modList[3] * 0xFFFF + modList[2] * 0xFFF + modList[1] * 0xFF + modList[0] * 0xF + color[4]
	}

	tp = Straight
	for i := 1;i < len(modList);i++ {
		if (modList[i] != modList[i -1] + 1) && (modList[i - 1] != 5 || modList[i] != modList[i - 1] + 9){
			tp = PokerType(-1)
			break
		}
	}
	if tp == Straight{
		return Straight,int(Straight) * TypeVal + modList[4] * 0xFFFFF + allVal * 0x0F + color[4]
	}

	if modList[0] == modList[2] || modList[1] == modList[3] || modList[2] == modList[4]{
		return ThreeOfAKind,int(ThreeOfAKind) * TypeVal + modList[2] * 0xFFFFF
	}

	if modList[0] == modList[1] && modList[3] == modList[4]{
		return TwoPair,int(TwoPair) * TypeVal + modList[4] * 0xFFFFF + modList[1] * 0xFFFF + modList[2] * 0xFFF + color[4]
	}else if modList[1] == modList[2] && modList[3] == modList[4]{
		return TwoPair,int(TwoPair) * TypeVal + modList[4] * 0xFFFFF + modList[2] * 0xFFFF + modList[0] * 0xFFF + color[4]
	}else if modList[0] == modList[1] && modList[2] == modList[3]{
		return TwoPair,int(TwoPair) * TypeVal + modList[3] * 0xFFFFF + modList[1] * 0xFFFF +modList[4] * 0xFFF + color[3]
	}

	if modList[0] == modList[1]{
		return Pair,int(Pair) * TypeVal + modList[1] * 0xFFFFF + modList[4] * 0xFFFF +modList[3] * 0xFFF + modList[2] * 0xFF + color[1]
	}else if modList[1] == modList[2]{
		return Pair,int(Pair) * TypeVal + modList[2] * 0xFFFFF + modList[4] * 0xFFFF +modList[3] * 0xFFF + modList[0] * 0xFF + color[2]
	}else if modList[2] == modList[3]{
		return Pair,int(Pair) * TypeVal + modList[3] * 0xFFFFF + modList[4] * 0xFFFF +modList[1] * 0xFFF + modList[0] * 0xFF + color[3]
	}else if modList[3] == modList[4]{
		return Pair,int(Pair) * TypeVal + modList[4] * 0xFFFFF + modList[2] * 0xFFFF +modList[1] * 0xFFF + modList[0] * 0xFF + color[4]
	}

	return Single,int(Single) * TypeVal + modList[4] * 0xFFFFF + modList[3] * 0xFFFF +modList[2] * 0xFFF + modList[1] * 0xFF + modList[0] * 0xF + color[4]
}



//-------------------------------------------------------自动摆牌的牌型判断---------------------------------------
func (this *MyTable) GetAllFlushStraight(pk []int,) [][]int{ //获取所有的同花顺
	flush := this.GetAllFlush(pk)
	flushStraight := make([][]int,0)
	for _,v := range flush{
		flushMap := map[int]int{}
		for _,v1 := range v{
			flushMap[v1 % 0x10] += 1
		}
		straight := this.GetAllStraight(flushMap,v)
		if len(straight) > 0{
			flushStraight = append(flushStraight,v)
		}
	}
	return flushStraight
}
func(this *MyTable) GetAllFlush(pk []int) [][]int{ //获取所有的同花
	color := map[int][]int{}
	for _,v := range pk{
		color[v / 0x10] = append(color[v / 0x10],v)
	}
	flush := make([][]int,0)
	for _,v := range color{
		if len(v) >= 5{
			comb := this.Combinations(v,5) //获取所有的组合
			for _,v1 := range comb{
				flush = append(flush,v1)
			}
		}
	}
	return flush
}
func(this *MyTable) GetAllStraight(mapList map[int]int,pk []int) [][]int{ //获取所有的顺子
	straight := make([][]int,0)
	aFlag := false
	lenPk := 0
	broken := false
	start := 0
	mapCopy := map[int]int{}
	mapCopy = this.CopyMap(mapList)
	for i := 2;i < 15;i++{
		if mapCopy[i] > 0{
			if start == 0{
				start = i
			}
			if i == 2 && mapCopy[14] > 0{
				mapCopy[14] -= 1
				lenPk += 1
				aFlag = true
			}
			lenPk += 1
			mapCopy[i] -= 1
		}else{
			broken = true
		}
		if lenPk >= 5 && (broken || i == 14){ //顺子
			tp := make([]int,0)
			if aFlag{
				for _,v := range pk{
					if v % 0x10 == 1{
						tp = append(tp,v)
						break
					}
				}
			}
			for k := start;k < start + lenPk;k++{
				for _,v := range pk{
					if v % 0x10 == k{
						tp = append(tp,v)
						break
					}
				}
			}

			for j := 0;j <= len(tp) - 5;j++{
				comb := []int{tp[j],tp[j + 1],tp[j + 2],tp[j + 3],tp[j + 4]}
				straight = append(straight,comb)
			}
		}

		if broken{
			broken = false
			start = 0
			lenPk = 0

			if aFlag{
				mapCopy[14] += 1
			}
		}


	}

	return straight
}
func(this *MyTable) GetAllFourOfAKind(mapList map[int]int,pk []int) [][]int { //获取所有的四条
	fourOfAKind := make([][]int,0)
	four := make([][]int,0)
	for k,v := range mapList{
		if v == 4{
			tp := make([]int,0)
			for _,v1 := range pk{
				if v1 % 0x10 == k{
					tp = append(tp,v1)
				}
			}
			four = append(four,tp)
		}
	}
	for _,v := range four{
		for _,v1 := range pk {
			fh := make([]int, len(v))
			copy(fh, v)
			find := false
			for _, v2 := range v {
				if v1 == v2{
					find = true
					break
				}
			}

			if !find{
				fh = append(fh,v1)
				fourOfAKind = append(fourOfAKind,fh)
			}
		}
	}
	return fourOfAKind
}
func(this *MyTable) GetAllFullHouse(mapList map[int]int,pk []int) [][]int { //获取所有的葫芦
	fullHouse := make([][]int,0)
	three := make([][]int,0)
	two := make([][]int,0)
	for k,v := range mapList{
		if v == 3{
			tp := make([]int,0)
			for _,v1 := range pk{
				if v1 % 0x10 == k{
					tp = append(tp,v1)
				}
			}
			three = append(three,tp)
		}else if v == 2{
			tp := make([]int,0)
			for _,v1 := range pk{
				if v1 % 0x10 == k{
					tp = append(tp,v1)
				}
			}
			two = append(two,tp)
		}
	}
	for i := 0;i < len(three);i++{
		for j := 0;j < len(three);j++{
			if j != i{
				comb := this.Combinations(three[j],2)
				for _,v := range comb{
					fh := make([]int,len(three[i]))
					copy(fh,three[i])
					for _,v1 := range v{
						fh = append(fh,v1)
					}
					fullHouse = append(fullHouse,fh)
				}
			}
		}

		for _,v := range two{
			fh := make([]int,len(three[i]))
			copy(fh,three[i])
			for _,v1 := range v{
				fh = append(fh,v1)
			}
			fullHouse = append(fullHouse,fh)
		}
	}

	return fullHouse
}
func(this *MyTable) GetAllThreeOfAKind(mapList map[int]int,pk []int) [][]int { //获取所有的三条
	threeOfAKind := make([][]int,0)
	three := make([]int,0)  //三条只会有一组，有两组的话，优先葫芦的
	for k,v := range mapList{
		if v == 3{
			for _,v1 := range pk{
				if v1 % 0x10 == k{
					three = append(three,v1)
				}
			}
		}
	}
	if len(three) > 0{
		single := make([]int,0)
		for _,v := range pk{
			if v % 0x10 != three[0] % 0x10{
				single = append(single,v)
			}
		}
		comb := this.Combinations(single,2) //获取所有的组合
		for _,v := range comb{
			fh := make([]int, len(three))
			copy(fh, three)
			for _,v1 := range v{
				fh = append(fh,v1)
			}

			threeOfAKind = append(threeOfAKind,fh)
		}
	}

	return threeOfAKind
}
func(this *MyTable) GetAllPair2(mapList map[int]int,pk []int) [][]int { //获取所有的两对
	pair2 := make([][]int,0)
	two := make([][]int,0)  //所有的对子
	twoMod := make([]int,0)
	for k,v := range mapList{
		if v == 2{
			tp := make([]int,0)
			for _,v1 := range pk{
				if v1 % 0x10 == k{
					tp = append(tp,v1)
				}
			}
			two = append(two,tp)
			twoMod = append(twoMod,tp[0] % 0x10)
		}
	}
	if len(two) >= 2{
		comb := this.Combinations(twoMod,2) //获取所有的组合
		for _,v := range comb{
			fh := make([]int,0)
			for _,v1 := range v{ //对子的模的组合
				for _,v2 := range pk{
					if v2 % 0x10 == v1{
						fh = append(fh,v2)
					}
				}
			}

			for _,v1 := range pk{
				find := false
				for _,v2 := range fh{
					if v2 == v1 {
						find = true
						break
					}
				}

				if !find{
					tp := make([]int,len(fh))
					copy(tp,fh)
					tp = append(tp,v1)
					pair2 = append(pair2,tp)
				}
			}

		}
	}

	return pair2
}
func(this *MyTable) GetAllPair(mapList map[int]int,pk []int) [][]int { //获取所有的对子
	pair := make([][]int,0)
	two := make([]int,0)  //只会有一对
	for k,v := range mapList{
		if v == 2{
			for _,v1 := range pk{
				if v1 % 0x10 == k{
					two = append(two,v1)
				}
			}
			break
		}
	}
	if len(two) > 0{
		single := make([]int,0)
		for _,v := range pk{
			if v % 0x10 != two[0] % 0x10{
				single = append(single,v)
			}
		}

		comb := this.Combinations(single,3) //获取所有的组合
		for _,v := range comb{
			fh := make([]int,len(two))
			copy(fh,two)
			for _,v1 := range v{
				fh = append(fh,v1)
			}
			pair = append(pair,fh)
		}
	}

	return pair
}
func(this *MyTable) GetAllSingle(pk []int) [][]int { //获取最大的单牌组合
	single := make([][]int,0)
	single = append(single,[]int{pk[len(pk) - 5],pk[len(pk) - 4],pk[len(pk) - 3],pk[len(pk) - 2],pk[len(pk) - 1]})
	return single
}
func(this *MyTable) DealSortPoker(in [][]int,firstPk []int,secondPk []int,maxVal int)([][]int,int){
	poker := make([][]int,3)
	for _,v1 := range in{
		thirdPk := this.FindNotInArrayList(v1,secondPk)
		_,val := this.CheckPokerType(thirdPk)
		if val >= maxVal{ //取这三墩
			maxVal = val
			poker[0] = thirdPk
			poker[1] = v1
			poker[2] = firstPk
		}
	}
	return poker,maxVal
}
func(this *MyTable) DealSortPoker2(in [][]int,pk []int)[][]int{
	maxVal := 0
	maxType := PokerType(-1)
	maxSecondVal := 0
	poker := make([][]int,3)
	for _,v := range in{
		secondPk := this.FindNotInArrayList(v,pk)
		sort.Slice(secondPk, func(i, j int) bool { //升序排序
			if secondPk[i] % 0x10 == secondPk[j] % 0x10{
				return secondPk[i] < secondPk[j]
			}
			return secondPk[i] % 0x10 < secondPk[j] % 0x10
		})
		secondModList := make([]int,len(secondPk))
		secondMapList := map[int]int{}
		for i := len(secondPk) -1;i >= 0;i--{
			secondModList[i] = secondPk[i] % 0x10
			secondMapList[secondModList[i]] += 1
		}

		secondFlushStraight := this.GetAllFlushStraight(secondPk)
		if len(secondFlushStraight) > 0{
			find := false
			if StraightFlush > maxType{
				find = true
				maxType = StraightFlush
				maxVal = 0
			} else if StraightFlush < maxType{
				continue
			}
			pk,val := this.DealSortPoker(secondFlushStraight,v,secondPk,maxVal)
			if val > maxVal || find{
				maxVal = val
				poker = pk
				_,maxSecondVal = this.CheckPokerType(pk[1])
			}else if val == maxVal{
				_,secondVal := this.CheckPokerType(pk[1])
				if secondVal > maxSecondVal{
					maxVal = val
					poker = pk
					maxSecondVal = secondVal
				}
			}
		}else{
			secondFourOfAKind := this.GetAllFourOfAKind(secondMapList,secondPk)
			if len(secondFourOfAKind) > 0{
				find := false
				if FourOfAKind > maxType{
					find = true
					maxType = FourOfAKind
					maxVal = 0
				} else if FourOfAKind < maxType{
					continue
				}
				pk,val := this.DealSortPoker(secondFourOfAKind,v,secondPk,maxVal)
				if val > maxVal || find{
					maxVal = val
					poker = pk
					_,maxSecondVal = this.CheckPokerType(pk[1])
				}else if val == maxVal{
					_,secondVal := this.CheckPokerType(pk[1])
					if secondVal > maxSecondVal{
						maxVal = val
						poker = pk
						maxSecondVal = secondVal
					}
				}
			}else{
				secondFullHouse := this.GetAllFullHouse(secondMapList,secondPk)
				if len(secondFullHouse) > 0{
					find := false
					if FullHouse > maxType{
						find = true
						maxType = FullHouse
						maxVal = 0
					} else if FullHouse < maxType{
						continue
					}
					pk,val := this.DealSortPoker(secondFullHouse,v,secondPk,maxVal)
					if val > maxVal || find{
						maxVal = val
						poker = pk
						_,maxSecondVal = this.CheckPokerType(pk[1])
					}else if val == maxVal{
						_,secondVal := this.CheckPokerType(pk[1])
						if secondVal > maxSecondVal{
							maxVal = val
							poker = pk
							maxSecondVal = secondVal
						}
					}
				}else{
					secondFlush := this.GetAllFlush(secondPk)
					if len(secondFlush) > 0{
						find := false
						if Flush > maxType{
							find = true
							maxType = Flush
							maxVal = 0
						} else if Flush < maxType{
							continue
						}
						pk,val := this.DealSortPoker(secondFlush,v,secondPk,maxVal)
						if val > maxVal || find{
							maxVal = val
							poker = pk
							_,maxSecondVal = this.CheckPokerType(pk[1])
						}else if val == maxVal{
							_,secondVal := this.CheckPokerType(pk[1])
							if secondVal > maxSecondVal{
								maxVal = val
								poker = pk
								maxSecondVal = secondVal
							}
						}
					}else{
						secondStraight := this.GetAllStraight(secondMapList,secondPk)
						if len(secondStraight) > 0{
							find := false
							if Straight > maxType{
								find = true
								maxType = Straight
								maxVal = 0
							} else if Straight < maxType{
								continue
							}
							pk,val := this.DealSortPoker(secondStraight,v,secondPk,maxVal)
							if val > maxVal || find{
								maxVal = val
								poker = pk
								_,maxSecondVal = this.CheckPokerType(pk[1])
							}else if val == maxVal{
								_,secondVal := this.CheckPokerType(pk[1])
								if secondVal > maxSecondVal{
									maxVal = val
									poker = pk
									maxSecondVal = secondVal
								}
							}
						}else{
							secondThreeOfAKind := this.GetAllThreeOfAKind(secondMapList,secondPk)
							if len(secondThreeOfAKind) > 0{
								find := false
								if ThreeOfAKind > maxType{
									find = true
									maxType = ThreeOfAKind
									maxVal = 0
								} else if ThreeOfAKind < maxType{
									continue
								}
								pk,val := this.DealSortPoker(secondThreeOfAKind,v,secondPk,maxVal)
								if val > maxVal || find{
									maxVal = val
									poker = pk
									_,maxSecondVal = this.CheckPokerType(pk[1])
								}else if val == maxVal{
									_,secondVal := this.CheckPokerType(pk[1])
									if secondVal > maxSecondVal{
										maxVal = val
										poker = pk
										maxSecondVal = secondVal
									}
								}
							}else {
								secondPair2 := this.GetAllPair2(secondMapList,secondPk)
								if len(secondPair2) > 0{
									find := false
									if TwoPair > maxType{
										find = true
										maxType = TwoPair
										maxVal = 0
									} else if TwoPair < maxType{
										continue
									}
									pk,val := this.DealSortPoker(secondPair2,v,secondPk,maxVal)
									if val > maxVal || find{
										maxVal = val
										poker = pk
										_,maxSecondVal = this.CheckPokerType(pk[1])
									}else if val == maxVal{
										_,secondVal := this.CheckPokerType(pk[1])
										if secondVal > maxSecondVal{
											maxVal = val
											poker = pk
											maxSecondVal = secondVal
										}
									}
								}else{
									secondPair := this.GetAllPair(secondMapList,secondPk)
									if len(secondPair) > 0{
										find := false
										if Pair > maxType{
											find = true
											maxType = Pair
											maxVal = 0
										} else if Pair < maxType{
											continue
										}
										pk,val := this.DealSortPoker(secondPair,v,secondPk,maxVal)
										if val > maxVal || find{
											maxVal = val
											poker = pk
											_,maxSecondVal = this.CheckPokerType(pk[1])
										}else if val == maxVal{
											_,secondVal := this.CheckPokerType(pk[1])
											if secondVal > maxSecondVal{
												maxVal = val
												poker = pk
												maxSecondVal = secondVal
											}
										}
									}else{
										secondSingle := this.GetAllSingle(secondPk)
										if len(secondSingle) > 0{
											find := false
											if Single > maxType{
												find = true
												maxType = Single
												maxVal = 0
											} else if Single < maxType{
												continue
											}
											pk,val := this.DealSortPoker(secondSingle,v,secondPk,maxVal)
											if val > maxVal || find{
												maxVal = val
												poker = pk
												_,maxSecondVal = this.CheckPokerType(pk[1])
											}else if val == maxVal{
												_,secondVal := this.CheckPokerType(pk[1])
												if secondVal > maxSecondVal{
													maxVal = val
													poker = pk
													maxSecondVal = secondVal
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return poker
}
func(this *MyTable) SortPokerFunc(userID string,pk []int)[]int{
	if len(pk) != 13{
		return nil
	}
	cardList := make([]int,len(pk))
	copy(cardList,pk)
	for i := 0;i < len(cardList);i++{
		if cardList[i] % 0x10 == 1{
			cardList[i] += 13
		}
	}

	sort.Slice(cardList, func(i, j int) bool { //升序排序
		if cardList[i] % 0x10 == cardList[j] % 0x10{
			return cardList[i] < cardList[j]
		}
		return cardList[i] % 0x10 < cardList[j] % 0x10
	})
	modList := make([]int,len(cardList))
	mapList := map[int]int{}
	for i := len(cardList) -1;i >= 0;i--{
		modList[i] = cardList[i] % 0x10
		mapList[modList[i]] += 1
	}

	poker := make([][]int,3) //返回的三墩
	//是否有同花顺
	flushStraight := this.GetAllFlushStraight(cardList)
	if len(flushStraight) > 0{
		 poker = this.DealSortPoker2(flushStraight,cardList)
	}else{
		fourOfAKind := this.GetAllFourOfAKind(mapList,cardList)
		if len(fourOfAKind) > 0{
			poker = this.DealSortPoker2(fourOfAKind,cardList)
		}else{
			fullHouse := this.GetAllFullHouse(mapList,cardList)
			if len(fullHouse) > 0{
				poker = this.DealSortPoker2(fullHouse,cardList)
			}else{
				flush := this.GetAllFlush(cardList)
				if len(flush) > 0{
					poker = this.DealSortPoker2(flush,cardList)
				}else{
					straight := this.GetAllStraight(mapList,cardList)
					if len(straight) > 0{
						poker = this.DealSortPoker2(straight,cardList)
					}else{
						threeOfAKind := this.GetAllThreeOfAKind(mapList,cardList)
						if len(threeOfAKind) > 0{
							poker = this.DealSortPoker2(threeOfAKind,cardList)
						}else{
							pair2 := this.GetAllPair2(mapList,cardList)
							if len(pair2) > 0{
								poker = this.DealSortPoker2(pair2,cardList)
							}else{
								pair := this.GetAllPair(mapList,cardList)
								if len(pair) > 0{
									poker = this.DealSortPoker2(pair,cardList)
								}else{
									single := this.GetAllSingle(cardList)
									if len(single) > 0{
										poker = this.DealSortPoker2(single,cardList)
									}
								}
							}
						}
					}
				}
			}
		}
	}
	for k,v := range poker{
		for k1,v1 := range v{
			if v1 % 0x10 == 14{
				poker[k][k1] -= 13
			}
		}
	}
	_,midVal := this.CheckPokerType(poker[1])
	_,tailVal := this.CheckPokerType(poker[2])
	if midVal > tailVal{
		mid := make([]int,len(poker[1]))
		copy(mid,poker[1])
		tail := make([]int,len(poker[2]))
		copy(tail,poker[2])

		poker[1] = tail
		poker[2] = mid
	}

	showPoker := make([]int,0)
	for _,v := range poker{
		for _,v1 := range v{
			showPoker = append(showPoker,v1)
		}
	}
	if userID != ""{
		pokeType := this.GetPokerTypeInterface(userID,showPoker)
		showPoker = this.SortShowPoker(poker,pokeType.PokerType)
		if pokeType.PokerType[0] == Single && pokeType.PokerType[1] == Single{
			tmp := showPoker[2]
			showPoker[2] = showPoker[6]
			showPoker[6] = tmp

			tmp = showPoker[1]
			showPoker[1] = showPoker[5]
			showPoker[5] = tmp
		}
	}
	return showPoker
}



