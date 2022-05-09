package game

type Type string

var (
	All      Type  = "all"
	YuXiaXie Type  = "yxx"
	BiDaXiao Type  = "dx"
	SeDie    Type  = "sd"
	Fish     Type  = "fish"
	Bjl      Type  = "bjl"
	Chat     Type  = "chat"
	Lobby    Type  = "lobby"
	Activity Type  = "activity"
	TouBao   Type  = "touBao"
	Mail     Type  = "mail"
	Win      int64 = 1
	Lost     int64 = -1

	Push    = "game/push"
	Nothing = "nothing"

	SlotLs        Type = "slotLs"  //龙神
	CardSss       Type = "cardSss" //十三水
	Lottery       Type = "lottery"
	MiniPoker     Type = "mini_poker"
	ChatGroups         = []string{"all", "yxx", "dx", "dxCurBet", "sd", "chat", "lobby", "touBao", "mail", "lottery", "mini_poker"}
	SlotCs        Type = "slotCs"    //财神
	SlotSex       Type = "slotSex"   //AV老虎机
	SlotDance     Type = "slotDance" //舞娘老虎机
	CardCddS      Type = "cardCddS"  //南方锄大地
	CardCddN      Type = "cardCddN"  //北方锄大地
	CardPhom      Type = "cardPhom"  //九张牌
	CardCatte     Type = "cardCatte" //六张牌
	CardLhd       Type = "cardLhd"   //龙虎斗
	CardQzsg      Type = "cardQzsg"  //抢庄三公
	Roshambo      Type = "roshambo"  //石头剪刀布
	GuessBigSmall Type = "guessBigSmall"
	Xg            Type = "apiXg"    // apiXg
	ApiLive       Type = "apiLive"  //视讯分类
	ApiSport      Type = "apiSport" //体育分类
	ApiCq         Type = "apiCq"
	ApiCmd        Type = "apiCmd"
	ApiAwc        Type = "apiAwc"
	ApiWm         Type = "apiWm"
	ApiSbo        Type = "apiSbo"
	ApiSpadeGame  Type = "apiSpadeGame"
	ApiDg         Type = "apiDg"
	ApiSaBa       Type = "apiSaBa"
	SuoHa         Type = "suoha"
)
var GameList = []Type{
	BiDaXiao,
	YuXiaXie,
	SeDie,
	SlotLs,
	SlotCs,
	SlotSex,
	SlotDance,
	CardSss,
	CardCddS,
	CardCddN,
	CardPhom,
	CardCatte,
	CardLhd,
	CardQzsg,
	Lottery,
	MiniPoker,
	Fish,
	Bjl,
	Roshambo,
	Xg,
	//ApiCq,
	ApiAwc,
	ApiWm,
	ApiCmd,
}
var ChatBotGame = []Type{
	BiDaXiao,
}
var BetGame = []Type{
	BiDaXiao,
	YuXiaXie,
	SeDie,
	CardLhd,
	Bjl,
}
