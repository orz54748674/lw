package cardCddNStorage

import "vn/framework/mongo-driver/bson/primitive"

type RoomData struct {
	ID         int64                `bson:"-" json:"-"`
	Oid        primitive.ObjectID   `bson:"_id,omitempty" json:"Oid"`
	TablesInfo map[string]TableInfo `bson:"TablesInfo" json:"TablesInfo"` //桌子信息
}
type Conf struct {
	ReadyTime            int //准备时长
	PutPokerTime         int //出牌时间
	JieSuanTime          int //结算时间
	MinEnterTableOdds    int //进入房间最低底分倍数
	ProfitPerThousand    int `bson:"ProfitPerThousand" json:"ProfitPerThousand"` //系统抽水 2%
	BotProfitPerThousand int //机器人抽水
}
type TableInfo struct {
	TableID        string `bson:"TableID" json:"TableID"`
	ServerID       string
	BaseScore      int64
	TotalPlayerNum int    //总人数
	Master         string //房主
	RobotNum       int    //机器人数量
}
type RobotConf struct {
	HallType     string //大厅类型 4人场 solo房
	BaseScore    int64  //底分
	BaseRobotNum int    //需要额外增加的机器人数量
	MaxOffset    int    //机器人最大偏移量
}
