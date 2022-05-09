package pk

import (
	"fmt"
	"sort"
)

var (
	Prize0 int8 = 0 // PrizeType
	Prize1 int8 = 1 // PrizeType
	Prize2 int8 = 2 // PrizeType
	Prize3 int8 = 3 // PrizeType
	Prize4 int8 = 4 // PrizeType
	Prize5 int8 = 5 // PrizeType
	Prize6 int8 = 6 // PrizeType
	Prize7 int8 = 7 // PrizeType
	Prize8 int8 = 8 // PrizeType
	Prize9 int8 = 9 // PrizeType
)

type pok struct {
	color  int8
	number int8
}

var pkMap map[int8]*pok

func initPkMap() {
	pkMap = make(map[int8]*pok)
	var count int8 = 1
	for i := 1; i <= 4; i++ {
		for j := 1; j <= 13; j++ {
			pkMap[count] = &pok{
				color:  int8(i),
				number: int8(j),
			}
			count++
		}
	}
}

// poks 长度必须大于等于3
func win(poks []int8) int8 {
	cMap := make(map[int8]int8)
	nMap := make(map[int8]int8)
	J := false
	var poksObj []*pok
	for _, k := range poks {

		if pkMap[k].number == 11 {
			J = true
		}
		poksObj = append(poksObj, pkMap[k])
		cMap[pkMap[k].color]++
		nMap[pkMap[k].number]++
	}
	if len(nMap) == 4 {
		for k, v := range nMap {
			if v > 1 && (k >= 11 || k == 1) {
				return Prize1 // >=11 (2-1-1-1) 2.7
			}
		}
		return Prize0 // <11 (2-1-1-1)  0
	} else if len(nMap) == 3 {
		for _, v := range nMap {
			if v == 3 {
				return Prize3 // 3-1-1 8
			}
		}
		return Prize2 // 2-2-1  5
	} else if len(nMap) == 2 {
		for _, v := range nMap {
			if v == 4 {
				return Prize7 // 4-1
			}
		}
		return Prize6 // 3-2
	} else {
		sort.SliceStable(poksObj, func(i, j int) bool {
			if poksObj[i].number > poksObj[j].number {
				return true
			}
			return false
		})
		pkCount := int8(len(poksObj))
		// for k, v := range poksObj {
		// 	fmt.Println("pokObj", k, v)
		// }
		// fmt.Println((poksObj[pkCount-1].number + pkCount - 1), poksObj[0].number, poksObj[pkCount-1].number, poksObj[pkCount-2].number)
		// // || ((poksObj[pkCount-2].number+pkCount-2) == poksObj[0].number && poksObj[pkCount-1].number == 5)
		if (poksObj[pkCount-1].number+pkCount-1) == poksObj[0].number || (poksObj[pkCount-1].number == 1 && poksObj[pkCount-2].number == 10) {
			fmt.Println("大奖", J, len(cMap))
			if len(cMap) == 1 {
				if J {
					return Prize9
				} else {
					return Prize8
				}
			}
			return Prize4
		}

	}
	if len(cMap) == 1 {
		return Prize5
	}
	return Prize0
}
