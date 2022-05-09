package cardQzsg

import (
	"sort"
)
var card = []int{
	0x11,0x12,0x13,0x14,0x15,0x16,0x17,0x18,0x19, //黑桃
	0x21,0x22,0x23,0x24,0x25,0x26,0x27,0x28,0x29, //梅花
	0x31,0x32,0x33,0x34,0x35,0x36,0x37,0x38,0x39,  //方块
	0x41,0x42,0x43,0x44,0x45,0x46,0x47,0x48,0x49,  //红桃
}
var switchBackendCard = map[int]int{
	0x11:40,0x12:41,0x13:42,0x14:43,0x15:44,0x16:45,0x17:46,0x18:47,0x19:48,0x1a:49,0x1b:50,0x1c:51,0x1d:52,
	0x21:14,0x22:15,0x23:16,0x24:17,0x25:18,0x26:19,0x27:20,0x28:21,0x29:22,0x2a:23,0x2b:24,0x2c:25,0x2d:26,
	0x31:1,0x32:2,0x33:3,0x34:4,0x35:5,0x36:6,0x37:7,0x38:8,0x39:9,0x3a:10,0x3b:11,0x3c:12,0x3d:13,
	0x41:27,0x42:28,0x43:29,0x44:30,0x45:31,0x46:32,0x47:33,0x48:34,0x49:35,0x4a:36,0x4b:37,0x4c:38,0x4d:39,
}
type PokerType int //牌型
const (
	One 		PokerType = 1 //
	Two   		PokerType = 2 //
	Three   	PokerType = 3 //
	Four 		PokerType = 4 //
	Five  		PokerType = 5 //
	Six			PokerType = 6 //
	Seven  		PokerType = 7 //
	Eight  		PokerType = 8 //
	Nine  		PokerType = 9 //
	Ten  		PokerType = 10 //
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

type CompList struct {
	maxPk int
	num int
	List []int
}

func(this *MyTable) GetCardType(pk []int)(pkType PokerType,max int,maxV int,num int){ //
	pkList := make([]int,len(pk))
	copy(pkList,pk)
	modList := make([]int,len(pk))
	for i := len(pkList) -1;i >= 0;i--{
		modList[i] = pk[i] % 0x10
	}
	sort.Slice(modList, func(i, j int) bool { //升序排序
		return modList[i] < modList[j]
	})
	sort.Slice(pkList, func(i, j int) bool { //降序排序
		_maxI := pkList[i] % 0x10 * 0xFF + pkList[i] / 0x10
		_maxJ := pkList[j] % 0x10 * 0xFF + pkList[j] / 0x10
		return _maxI > _maxJ
	})

	return PokerType(0),0,0,0
}