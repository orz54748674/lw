package lotteryStorage

import (
	"strings"
	"vn/common/utils"
)

type playFunc func(bet string, betAmount, odds int64, OpenCode map[PrizeLevel][]string) int64

var PlayMap map[string]playFunc

func InitPlayMap() {
	PlayMap = make(map[string]playFunc)
	PlayMap["North_BZ_C2"] = bzC2
	PlayMap["North_BZ_C3"] = bzC3
	PlayMap["North_BZ_C4"] = bzC4
	PlayMap["North_2DTW_TOU"] = north2DTWTOU
	PlayMap["North_2DTW_WEI"] = north2DTWWEI
	PlayMap["North_2DTW_TOUWEI"] = north2DTWTOUWEI
	PlayMap["North_3DTW_6J"] = north3DTW6J
	PlayMap["North_3DTW_TJ"] = north3DTWTJ
	PlayMap["North_3DTW_6T"] = north3DTW6T
	PlayMap["North_1D_TJ4"] = north1DTJ4
	PlayMap["North_1D_TJ5"] = north1DTJ5
	PlayMap["North_ZH_X2"] = combination
	PlayMap["North_ZH_X3"] = combination
	PlayMap["North_ZH_X4"] = combination
	PlayMap["North_BCZH_4BC"] = noCombination
	PlayMap["North_BCZH_8BC"] = noCombination
	PlayMap["North_BCZH_10BC"] = noCombination

	PlayMap["Central_BZ_C2"] = bzC2
	PlayMap["Central_BZ_C3"] = bzC3
	PlayMap["Central_BZ_C4"] = bzC4
	PlayMap["Central_2DTW_TOU"] = south2DTWTOU
	PlayMap["Central_2DTW_WEI"] = south2DTWWEI
	PlayMap["Central_2DTW_TOUWEI"] = south2DTWTOUWEI
	PlayMap["Central_3DTW_7J"] = south3DTW7J
	PlayMap["Central_3DTW_TJ"] = south3DTWTJ
	PlayMap["Central_3DTW_7T"] = south3DTW7T
	PlayMap["Central_1D_TJ4"] = north1DTJ4
	PlayMap["Central_1D_TJ5"] = north1DTJ5
	PlayMap["Central_ZH_X2"] = combination
	PlayMap["Central_ZH_X3"] = combination
	PlayMap["Central_ZH_X4"] = combination
	PlayMap["Central_BCZH_4BC"] = noCombination
	PlayMap["Central_BCZH_8BC"] = noCombination
	PlayMap["Central_BCZH_10BC"] = noCombination

	PlayMap["South_BZ_C2"] = bzC2
	PlayMap["South_BZ_C3"] = bzC3
	PlayMap["South_BZ_C4"] = bzC4
	PlayMap["South_2DTW_TOU"] = south2DTWTOU
	PlayMap["South_2DTW_WEI"] = south2DTWWEI
	PlayMap["South_2DTW_TOUWEI"] = south2DTWTOUWEI
	PlayMap["South_3DTW_7J"] = south3DTW7J
	PlayMap["South_3DTW_TJ"] = south3DTWTJ
	PlayMap["South_3DTW_7T"] = south3DTW7T
	PlayMap["South_1D_TJ4"] = north1DTJ4
	PlayMap["South_1D_TJ5"] = north1DTJ5
	PlayMap["South_ZH_X2"] = combination
	PlayMap["South_ZH_X3"] = combination
	PlayMap["South_ZH_X4"] = combination
	PlayMap["South_BCZH_4BC"] = noCombination
	PlayMap["South_BCZH_8BC"] = noCombination
	PlayMap["South_BCZH_10BC"] = noCombination
}

// ?????? ??????
// =========================================================================
func bzC2(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	return getBZWinCount(strBet, openCode, 2) * betAmount * odds
}

func bzC3(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	return getBZWinCount(strBet, openCode, 3) * betAmount * odds
}

func bzC4(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	return getBZWinCount(strBet, openCode, 4) * betAmount * odds
}

func getBZWinCount(strBet string, openCode map[PrizeLevel][]string, codeLen int) (count int64) {
	codeMap := make(map[string]int64)
	for _, codes := range openCode {
		for _, code := range codes {
			if len(code) >= codeLen {
				codeMap[code[len(code)-codeLen:]] += 1
			}
		}
	}
	bets := strings.Split(strBet, "-")
	for _, bet := range bets {
		count += codeMap[bet]
	}
	return
}

// =========================================================================
// 2d ??????
// =========================================================================
/**
 * @title north2DTWTOU
 * @description ??????2d?????? (???)
 */
func north2DTWTOU(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	return getBetWinCount(strBet, tailOpenCodes(openCode, 2, PrizeLevel7)) * betAmount * odds
}

/**
 * @title north2DTWWEI
 * @description ??????2d?????? (???)
 */
func north2DTWWEI(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	return getBetWinCount(strBet, tailOpenCodes(openCode, 2, PrizeLevel0)) * betAmount * odds
}

/**
 * @title north2DTWTOUWEI
 * @description ??????2d?????? (??????)
 */
func north2DTWTOUWEI(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	codeMap := make(map[string]int64)
	for _, priceLevel := range []PrizeLevel{PrizeLevel0, PrizeLevel7} {
		for _, code := range openCode[priceLevel] {
			if len(code) >= 2 {
				codeMap[code[len(code)-2:]] += 1
			}
		}
	}
	bets := strings.Split(strBet, "-")
	var count int64
	for _, bet := range bets {
		count += codeMap[bet]
	}
	return count * odds * betAmount
}

/**
 * @title south2DTWTOU
 * @description ??????2d?????? (???)
 */
func south2DTWTOU(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	return getBetWinCount(strBet, tailOpenCodes(openCode, 2, PrizeLevel8)) * betAmount * odds
}

/**
 * @title south2DTWWEI
 * @description ??????2d?????? (???)
 */
func south2DTWWEI(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	return north2DTWWEI(strBet, betAmount, odds, openCode)
}

/**
 * @title south2DTWTOUWEI
 * @description ??????2d?????? (??????)
 */
func south2DTWTOUWEI(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	codeMap := make(map[string]int64)
	for _, priceLevel := range []PrizeLevel{PrizeLevel0, PrizeLevel8} {
		for _, code := range openCode[priceLevel] {
			if len(code) >= 2 {
				codeMap[code[len(code)-2:]] += 1
			}
		}
	}
	bets := strings.Split(strBet, "-")
	var count int64
	for _, bet := range bets {
		count += codeMap[bet]
	}
	return count * odds * betAmount
}

// =========================================================================
// 3d ??????
// =========================================================================
/**
 * @title north3DTW6J
 * @description ??????3d?????? (3 C??ng ?????u)
 */
func north3DTW6J(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	return getBetWinCount(strBet, tailOpenCodes(openCode, 3, PrizeLevel6)) * odds * betAmount
}

/**
 * @title north3DTWTJ
 * @description ??????3d?????? (3 C??ng ?????c Bi???t)
 */
func north3DTWTJ(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	return getBetWinCount(strBet, tailOpenCodes(openCode, 3, PrizeLevel0)) * odds * betAmount
}

/**
 * @title north3DTW6T
 * @description ??????3d?????? (3 C??ng ?????u ??u??i)
 */
func north3DTW6T(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	codeMap := make(map[string]int64)
	for _, priceLevel := range []PrizeLevel{PrizeLevel0, PrizeLevel6} {
		for _, code := range openCode[priceLevel] {
			if len(code) >= 3 {
				codeMap[code[len(code)-3:]] += 1
			}
		}
	}
	bets := strings.Split(strBet, "-")
	var count int64
	for _, bet := range bets {
		count += codeMap[bet]
	}
	return count * odds * betAmount
}

/**
 * @title south3DTW7J
 * @description ??????3d?????? (3 C??ng ?????u)
 */
func south3DTW7J(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	return getBetWinCount(strBet, tailOpenCodes(openCode, 3, PrizeLevel7)) * odds * betAmount
}

/**
 * @title south3DTWTJ
 * @description ??????3d?????? (3 C??ng ?????c Bi???t)
 */
func south3DTWTJ(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	return getBetWinCount(strBet, tailOpenCodes(openCode, 3, PrizeLevel0)) * odds * betAmount
}

/**
 * @title south3DTW7T
 * @description ??????3d?????? (3 C??ng ?????u ??u??i)
 */
func south3DTW7T(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	codeMap := make(map[string]int64)
	for _, priceLevel := range []PrizeLevel{PrizeLevel0, PrizeLevel7} {
		for _, code := range openCode[priceLevel] {
			if len(code) >= 3 {
				codeMap[code[len(code)-3:]] += 1
			}
		}
	}
	bets := strings.Split(strBet, "-")
	var count int64
	for _, bet := range bets {
		count += codeMap[bet]
	}
	return count * odds * betAmount
}

// =========================================================================
// 1d ??????
// =========================================================================
/**
 * @title north1DTJ4
 * @description ??????1d???
 */
func north1DTJ4(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	code := openCode[PrizeLevel0][0]
	bets := strings.Split(strBet, "-")
	var count int64
	if utils.StrInArray(code[len(code)-2:len(code)-1], bets) {
		count++
	}
	return count * odds * betAmount
}

/**
 * @title north1DTJ5
 * @description ??????1d???
 */
func north1DTJ5(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	code := openCode[PrizeLevel0][0]
	bets := strings.Split(strBet, "-")
	var count int64
	if utils.StrInArray(code[len(code)-1:], bets) {
		count++
	}
	return count * odds * betAmount
}

// =========================================================================
// ???????????? ??????
// =========================================================================
/**
 * @title combination
 * @description ??????????????????
 */
func combination(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	codeNumbers := allTailOpenCodes(openCode, 2)
	bets := strings.Split(strBet, "-")
	for _, bet := range bets {
		if !utils.StrInArray(bet, codeNumbers) {
			return 0
		}
	}
	return betAmount * odds
}

// =========================================================================
// ?????????????????? ??????
// =========================================================================
/**
 * @title noCombination
 * @description ????????????????????????
 */
func noCombination(strBet string, betAmount, odds int64, openCode map[PrizeLevel][]string) int64 {
	codeNumbers := allTailOpenCodes(openCode, 2)
	bets := strings.Split(strBet, "-")
	for _, bet := range bets {
		if utils.StrInArray(bet, codeNumbers) {
			return 0
		}
	}
	return betAmount * odds
}

// =========================================================================
func getBetWinCount(strBet string, codeNumbers []string) (count int64) {
	bets := strings.Split(strBet, "-")
	for _, betCode := range bets {
		if utils.StrInArray(betCode, codeNumbers) {
			count++
		}
	}
	return
}

func allTailOpenCodes(openCode map[PrizeLevel][]string, codeLen int) (codeNumbers []string) {
	for _, codes := range openCode {
		for _, code := range codes {
			if len(code) >= codeLen {
				codeNumbers = append(codeNumbers, code[len(code)-codeLen:])
			}
		}
	}
	return
}

func tailOpenCodes(openCode map[PrizeLevel][]string, codeLen int, prizeLevels ...PrizeLevel) (codeNumbers []string) {
	for _, prizeLevel := range prizeLevels {
		for _, code := range openCode[prizeLevel] {
			if len(code) >= codeLen {
				codeNumbers = append(codeNumbers, code[len(code)-codeLen:])
			}
		}
	}
	return
}
