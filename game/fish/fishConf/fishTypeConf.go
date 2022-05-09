package fishConf

type FishType struct {
	ID         int
	Name       string
	RewardMin  int
	RewardMax  int
	KillProMin int
	KillProMax int
	Time       int
	FType      int
	Interval   int
}

var FishTypeConf = map[int]FishType{
	101: {
		ID:         101,   //鱼的类型
		Name:       "小小鱼", //鱼的名字
		RewardMin:  2,     //最小奖励倍数
		RewardMax:  2,     //最大奖励倍数
		KillProMin: 5000,  //最小击杀概率
		KillProMax: 5000,  //最大击杀概率
		Time:       100000, //生存时间周期
		FType:      0,     // 0普通鱼, 1特殊鱼
		Interval:   800,   //两条同路径最小间隔时间
	},
	102: {
		ID:         102,
		Name:       "小金鱼",
		RewardMin:  3,
		RewardMax:  3,
		KillProMin: 3333,
		KillProMax: 3333,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	103: {
		ID:         103,
		Name:       "小黄鱼",
		RewardMin:  4,
		RewardMax:  4,
		KillProMin: 2500,
		KillProMax: 2500,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	104: {
		ID:         104,
		Name:       "小河豚",
		RewardMin:  5,
		RewardMax:  5,
		KillProMin: 2000,
		KillProMax: 2000,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	105: {
		ID:         105,
		Name:       "扇子鱼",
		RewardMin:  6,
		RewardMax:  6,
		KillProMin: 1667,
		KillProMax: 1667,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	106: {
		ID:         106,
		Name:       "扁褐鱼",
		RewardMin:  7,
		RewardMax:  7,
		KillProMin: 1429,
		KillProMax: 1429,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	107: {
		ID:         107,
		Name:       "小龙虾",
		RewardMin:  8,
		RewardMax:  8,
		KillProMin: 1250,
		KillProMax: 1250,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	108: {
		ID:         108,
		Name:       "小鲨鱼",
		RewardMin:  9,
		RewardMax:  9,
		KillProMin: 1111,
		KillProMax: 1111,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	109: {
		ID:         109,
		Name:       "章鱼",
		RewardMin:  10,
		RewardMax:  10,
		KillProMin: 1000,
		KillProMax: 1000,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	110: {
		ID:         110,
		Name:       "水母",
		RewardMin:  12,
		RewardMax:  12,
		KillProMin: 833,
		KillProMax: 833,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	111: {
		ID:         111,
		Name:       "灯笼鱼",
		RewardMin:  15,
		RewardMax:  15,
		KillProMin: 667,
		KillProMax: 667,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	112: {
		ID:         112,
		Name:       "乌龟",
		RewardMin:  18,
		RewardMax:  18,
		KillProMin: 556,
		KillProMax: 556,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	113: {
		ID:         113,
		Name:       "长嘴鱼",
		RewardMin:  20,
		RewardMax:  20,
		KillProMin: 500,
		KillProMax: 500,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	114: {
		ID:         114,
		Name:       "蓝鱼",
		RewardMin:  25,
		RewardMax:  25,
		KillProMin: 400,
		KillProMax: 400,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	115: {
		ID:         115,
		Name:       "大金鱼",
		RewardMin:  30,
		RewardMax:  30,
		KillProMin: 333,
		KillProMax: 333,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	116: {
		ID:         116,
		Name:       "大黄鱼",
		RewardMin:  40,
		RewardMax:  40,
		KillProMin: 250,
		KillProMax: 250,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	117: {
		ID:         117,
		Name:       "大河豚",
		RewardMin:  50,
		RewardMax:  50,
		KillProMin: 200,
		KillProMax: 200,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	118: {
		ID:         118,
		Name:       "大鲨鱼",
		RewardMin:  30,
		RewardMax:  70,
		KillProMin: 143,
		KillProMax: 333,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	119: {
		ID:         119,
		Name:       "超级金鱼",
		RewardMin:  30,
		RewardMax:  70,
		KillProMin: 143,
		KillProMax: 333,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	120: {
		ID:         120,
		Name:       "超级黄鱼",
		RewardMin:  40,
		RewardMax:  80,
		KillProMin: 125,
		KillProMax: 250,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	121: {
		ID:         121,
		Name:       "超级河豚",
		RewardMin:  40,
		RewardMax:  90,
		KillProMin: 111,
		KillProMax: 250,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	122: {
		ID:         122,
		Name:       "金鲨鱼",
		RewardMin:  50,
		RewardMax:  100,
		KillProMin: 100,
		KillProMax: 200,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	123: {
		ID:         123,
		Name:       "蓝鲨鱼",
		RewardMin:  50,
		RewardMax:  100,
		KillProMin: 100,
		KillProMax: 200,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	124: {
		ID:         124,
		Name:       "超级金鱼",
		RewardMin:  60,
		RewardMax:  200,
		KillProMin: 50,
		KillProMax: 167,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	125: {
		ID:         125,
		Name:       "龙",
		RewardMin:  60,
		RewardMax:  888,
		KillProMin: 11,
		KillProMax: 167,
		Time:       100000,
		FType:      0,
		Interval:   1200,
	},
	201: {
		ID:         201,
		Name:       "镭射蟹",
		RewardMin:  60,
		RewardMax:  888,
		KillProMin: 11,
		KillProMax: 167,
		Time:       100000,
		FType:      1,
		Interval:   1200,
	},
	202: {
		ID:         202,
		Name:       "电钻蟹",
		RewardMin:  60,
		RewardMax:  888,
		KillProMin: 11,
		KillProMax: 167,
		Time:       100000,
		FType:      1,
		Interval:   1200,
	},
	203: {
		ID:         203,
		Name:       "炸弹蟹",
		RewardMin:  60,
		RewardMax:  888,
		KillProMin: 11,
		KillProMax: 167,
		Time:       100000,
		FType:      1,
		Interval:   1200,
	},
	204: {
		ID:         204,
		Name:       "轮轴蟹",
		RewardMin:  60,
		RewardMax:  888,
		KillProMin: 11,
		KillProMax: 167,
		Time:       100000,
		FType:      1,
		Interval:   1200,
	},
	205: {
		ID:         205,
		Name:       "闪电水母",
		RewardMin:  60,
		RewardMax:  888,
		KillProMin: 11,
		KillProMax: 167,
		Time:       100000,
		FType:      1,
		Interval:   1200,
	},
	206: {
		ID:         206,
		Name:       "雷霆火炮",
		RewardMin:  60,
		RewardMax:  888,
		KillProMin: 11,
		KillProMax: 167,
		Time:       100000,
		FType:      1,
		Interval:   1200,
	},
	207: {
		ID:         207,
		Name:       "龙王献宝",
		RewardMin:  10,
		RewardMax:  300,
		KillProMin: 33,
		KillProMax: 1000,
		Time:       100000,
		FType:      1,
		Interval:   1200,
	},
	208: {
		ID:         208,
		Name:       "黄金福袋",
		RewardMin:  20,
		RewardMax:  200,
		KillProMin: 50,
		KillProMax: 500,
		Time:       100000,
		FType:      1,
		Interval:   1200,
	},
}