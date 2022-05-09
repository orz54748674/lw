package suoha

import (
	"encoding/json"
	"sort"
	"vn/common"
	"vn/game"
)

var (
	CARD_TYPE_PUTONG      = 0
	CARD_TYPE_DUIZI       = 1
	CARD_TYPE_LIANGDUI    = 2
	CARD_TYPE_SANTIAO     = 3
	CARD_TYPE_SHUNZI      = 4
	CARD_TYPE_TONGHUA     = 5
	CARD_TYPE_HULU        = 6
	CARD_TYPE_SITIAO      = 7
	CARD_TYPE_TONGHUASHUN = 8
)

func (s *Table) DealProtocolFormat(in interface{}, action string, error *common.Err) []byte {
	info := struct {
		Data     interface{}
		GameType game.Type
		Action   string
		ErrMsg   string
		Code     int
	}{
		Data:     in,
		GameType: game.Bjl,
		Action:   action,
	}
	if error == nil {
		info.Code = 0
		info.ErrMsg = "操作成功"
	} else {
		info.Code = error.Code
		info.ErrMsg = error.SetKey().ErrMsg
	}

	ret, _ := json.Marshal(info)
	return ret
}

func (s *Table) sendPackToAll(topic string, in interface{}, action string, err *common.Err) {
	body := s.DealProtocolFormat(in, action, err)
	s.NotifyCallBackMsgNR(topic, body)
}

func (s *Table) sendPack(session string, topic string, in interface{}, action string, err *common.Err) {
	body := s.DealProtocolFormat(in, action, err)
	s.SendCallBackMsgNR([]string{session}, topic, body)
}

func (s *Table) getCardValue(card int) int {
	if card%13 == 1 {
		return 14
	}
	if card%13 == 0 {
		return 13
	}
	return card % 13
}

func (s *Table) getCardColor(card int) int {
	card = card - 1
	color := card / 13
	if color == 0 {
		return 2
	} else if color == 2 {
		return 3
	} else if color == 3 {
		return 0
	} else {
		return color
	}
}

func (s *Table) Card2Bigger(card1, card2 int) bool {
	if s.getCardValue(card1) != s.getCardValue(card2) {
		if s.getCardValue(card2) > s.getCardValue(card1) {
			return true
		}
		return false
	}
	if s.getCardColor(card2) > s.getCardColor(card1) {
		return true
	}
	return false
}

func (s *Table) GetMaxCard(cards []int) int {
	maxCard := 0
	for _, v := range cards {
		if maxCard == 0 {
			maxCard = v
		} else {
			if s.Card2Bigger(maxCard, v) {
				maxCard = v
			}
		}
	}
	return maxCard
}

func (s *Table) CompareTwoPair(cards1, cards2 []int) bool {
	if len(cards1) != 5 || len(cards2) != 5 {
		return false
	}
	var bigPair1, bigPair2, smallPair1, smallPair2 []int
	var danCard1, danCard2 int
	colorMap1 := make(map[int][]int)
	valueMap1 := make(map[int][]int)
	for _, v := range cards1 {
		colorMap1[s.getCardColor(v)] = append(colorMap1[s.getCardColor(v)], v)
		valueMap1[s.getCardValue(v)] = append(valueMap1[s.getCardValue(v)], v)
	}

	colorMap2 := make(map[int][]int)
	valueMap2 := make(map[int][]int)
	for _, v := range cards2 {
		colorMap2[s.getCardColor(v)] = append(colorMap2[s.getCardColor(v)], v)
		valueMap2[s.getCardValue(v)] = append(valueMap2[s.getCardValue(v)], v)
	}

	for val, arr := range valueMap1 {
		if len(arr) == 2 {
			if len(bigPair1) == 0 {
				bigPair1 = arr
				smallPair1 = arr
			} else {
				if val > s.getCardValue(bigPair1[0]) {
					bigPair1 = arr
				} else {
					smallPair1 = arr
				}
			}
		} else {
			danCard1 = arr[0]
		}
	}

	for val, arr := range valueMap2 {
		if len(arr) == 2 {
			if len(bigPair2) == 0 {
				bigPair2 = arr
				smallPair2 = arr
			} else {
				if val > s.getCardValue(bigPair2[0]) {
					bigPair2 = arr
				} else {
					smallPair2 = arr
				}
			}
		} else {
			danCard2 = arr[0]
		}
	}

	if len(bigPair1) != 2 || len(bigPair2) != 2 || len(smallPair1) != 2 || len(smallPair2) != 2 {
		return false
	}

	if s.getCardValue(bigPair2[0]) > s.getCardValue(bigPair1[0]) {
		return true
	} else if s.getCardValue(bigPair2[0]) < s.getCardValue(bigPair1[0]) {
		return false
	}
	if s.getCardValue(smallPair2[0]) > s.getCardValue(smallPair1[0]) {
		return true
	} else if s.getCardValue(smallPair2[0]) < s.getCardValue(smallPair1[0]) {
		return false
	}
	if s.getCardValue(danCard2) > s.getCardValue(danCard1) {
		return true
	} else if s.getCardValue(danCard2) < s.getCardValue(danCard1) {
		return false
	}

	bigPairColor1 := s.getCardColor(bigPair1[0])
	if s.getCardColor(bigPair1[1]) > bigPairColor1 {
		bigPairColor1 = s.getCardColor(bigPair1[1])
	}

	bigPairColor2 := s.getCardColor(bigPair2[0])
	if s.getCardColor(bigPair2[1]) > bigPairColor2 {
		bigPairColor2 = s.getCardColor(bigPair2[1])
	}

	return bigPairColor2 > bigPairColor1
}

func (s *Table) ComparePair(cards1, cards2 []int) bool {
	if len(cards1) != 5 || len(cards2) != 5 {
		return false
	}
	var pair1, pair2 []int
	colorMap1 := make(map[int][]int)
	valueMap1 := make(map[int][]int)
	for _, v := range cards1 {
		colorMap1[s.getCardColor(v)] = append(colorMap1[s.getCardColor(v)], v)
		valueMap1[s.getCardValue(v)] = append(valueMap1[s.getCardValue(v)], v)
	}

	colorMap2 := make(map[int][]int)
	valueMap2 := make(map[int][]int)
	for _, v := range cards2 {
		colorMap2[s.getCardColor(v)] = append(colorMap2[s.getCardColor(v)], v)
		valueMap2[s.getCardValue(v)] = append(valueMap2[s.getCardValue(v)], v)
	}

	var danArr1, danArr2 []int
	for _, arr := range valueMap1 {
		if len(arr) == 2 {
			pair1 = arr
		} else {
			danArr1 = append(danArr1, arr[0])
		}
	}
	for _, arr := range valueMap2 {
		if len(arr) == 2 {
			pair2 = arr
		} else {
			danArr2 = append(danArr2, arr[0])
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(danArr1)))
	sort.Sort(sort.Reverse(sort.IntSlice(danArr2)))

	if s.getCardValue(pair2[0]) > s.getCardValue(pair1[0]) {
		return true
	} else if s.getCardValue(pair2[0]) < s.getCardValue(pair1[0]) {
		return false
	}

	for i := 0; i <= len(danArr1); i++ {
		if s.getCardValue(danArr2[i]) > s.getCardValue(danArr1[i]) {
			return true
		} else if s.getCardValue(danArr2[i]) < s.getCardValue(danArr1[i]) {
			return false
		}
	}

	pairColor1 := s.getCardColor(pair1[0])
	if s.getCardColor(pair1[1]) > pairColor1 {
		pairColor1 = s.getCardColor(pair1[1])
	}

	pairColor2 := s.getCardColor(pair2[0])
	if s.getCardColor(pair2[1]) > pairColor2 {
		pairColor2 = s.getCardColor(pair2[1])
	}

	return pairColor2 > pairColor1
}

/*
普通牌：0，
对子：1，
两队：2，
三条：3，
顺子：4，
同花：5，
葫芦：6，
四条：7，
同花顺：8，
*/
func (s *Table) GetCardsTypeAndMaxCard(cards []int) (int, int) {
	colorMap := make(map[int][]int)
	valueMap := make(map[int][]int)
	for _, v := range cards {
		colorMap[s.getCardColor(v)] = append(colorMap[s.getCardColor(v)], v)
		valueMap[s.getCardValue(v)] = append(valueMap[s.getCardValue(v)], v)
	}

	//判断是否同花顺
	if len(colorMap) == 1 && len(valueMap) == 5 {
		maxVal := 0
		minVal := 0
		for val, _ := range valueMap {
			if val > maxVal {
				maxVal = val
			}
			if val < minVal {
				minVal = val
			}
		}
		if maxVal-minVal == 4 {
			return CARD_TYPE_TONGHUASHUN, s.GetMaxCard(cards)
		}
	}

	//判断是否四条,葫芦
	if len(valueMap) == 2 {
		for _, arr := range valueMap {
			if len(arr) == 4 {
				return CARD_TYPE_SITIAO, s.GetMaxCard(arr)
			}
			if len(arr) == 3 {
				return CARD_TYPE_HULU, s.GetMaxCard(arr)
			}
		}
	}

	//判断是否同花
	if len(valueMap) == 5 && len(colorMap) == 1 {
		return CARD_TYPE_TONGHUA, s.GetMaxCard(cards)
	}

	//判断是否顺子
	if len(valueMap) == 5 {
		maxVal := 0
		minVal := 0
		for val, _ := range valueMap {
			if val > maxVal {
				maxVal = val
			}
			if val < minVal {
				minVal = val
			}
		}
		if maxVal-minVal == 4 {
			return CARD_TYPE_SHUNZI, s.GetMaxCard(cards)
		}
	}

	//判断是否三条/两队
	if len(valueMap) == 3 {
		for _, arr := range valueMap {
			if len(arr) == 3 {
				return CARD_TYPE_SANTIAO, s.GetMaxCard(arr)
			}
		}

		return CARD_TYPE_LIANGDUI, s.GetMaxCard(cards)
	}

	//判断是否对子
	if len(valueMap) == 4 {
		for _, arr := range valueMap {
			if len(arr) == 2 {
				return CARD_TYPE_DUIZI, s.GetMaxCard(arr)
			}
		}
	}

	return CARD_TYPE_PUTONG, s.GetMaxCard(cards)
}

func (s *Table) CompareTwoCards(cards1, cards2 []int) bool {
	cardsType1, maxCard1 := s.GetCardsTypeAndMaxCard(cards1)
	cardsType2, maxCard2 := s.GetCardsTypeAndMaxCard(cards2)
	if cardsType2 > cardsType1 {
		return true
	} else if cardsType1 > cardsType2 {
		return false
	}

	if cardsType1 == CARD_TYPE_DUIZI || cardsType1 == CARD_TYPE_LIANGDUI {
		if cardsType1 == CARD_TYPE_DUIZI {
			return s.ComparePair(cards1, cards2)
		} else {
			return s.CompareTwoPair(cards1, cards2)
		}
	} else {
		return s.Card2Bigger(maxCard1, maxCard2)
	}
}
