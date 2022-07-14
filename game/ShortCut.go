package game

type ShortCutMode struct {
	Type string
	Text string
}

var ShortCutText = map[Type][]string{
	YuXiaXie: {
		"Thanh xuân chờ hũ nổ .",
		"Dễ ăn vcl.",
		"Im d9iii !!!",
		"Anh em cho xin cái lộc nào.",
		"Gấp mấy tay ra đảo luôn .",
		"Bỏ lại ra . Má nó !!!",
		"1 phát ăn luôn hũ , đm.",
		"Gà ăn cám nãy giờ .",
		"Nuôi con nào đây ae ?",
		"Hũ sắp nổ anh em vô tiền mạnh nào .",
		"Trúng không trượt phát nào .",
		"Hôm nay đen quá !",
		"Đặt vui để ăn cái hũ thôi .",
		"Nuôi nãy giờ toàn tịt .",
		"Tôm , cua hết ngon cmnr.",
		"Nhắm mắt đưa tay lại ăn .",
		"Nai ra đúng lúc kao bỏ .",
		"Cá dạo này phất nè .",
	},
	SeDie: {
		"Tất tay nào anh em !",
		"Bạc đỏ muốn thua cũng khó",
		"Má ! bẻ cầu lại thua đâu thật.",
		"Ăn thông nhé bà con .Toàn chuẩn vị .",
		"Cầu này khó quá . Đánh chậm lại nhé ae .",
		"Đặt chẳn ra lẻ , đặt lẻ ra chẳn . Đời quá đen .",
		"Tới cầu của tôi rồi nhé . Các ông theo tôi kiếm tiền .",
		"ĐM cầu ảo vãi .",
		"Xin 1 tay chuẩn vị nào anh em .",
		"Chẳn đi ae .",
		"Gãy phát này đau quá .",
		"Thôi rồi lẻ ơi !",
		"Chẵn ợi , e ở đâu !",
		"1 ván vô thành vô sản luôn , đậu xanh !!!",
		"Móa lại sập cầu .",
		"Thời tới khó cản .",
		"Đỏ rồi ae ơi .",
	},
	CardLhd: {
		"Thanh xuân chờ hũ nổ .",
		"Dễ ăn vcl.",
		"Im d9iii !!!",
		"Anh em cho xin cái lộc nào.",
		"Gấp mấy tay ra đảo luôn .",
		"Bỏ lại ra . Má nó !!!",
		"1 phát ăn luôn hũ , đm.",
		"Gà ăn cám nãy giờ .",
		"Nuôi con nào đây ae ?",
		"Hũ sắp nổ anh em vô tiền mạnh nào .",
		"Trúng không trượt phát nào .",
		"Hôm nay đen quá !",
		"Đặt vui để ăn cái hũ thôi .",
		"Nuôi nãy giờ toàn tịt .",
		"Tôm , cua hết ngon cmnr.",
		"Nhắm mắt đưa tay lại ăn .",
		"Nai ra đúng lúc kao bỏ .",
		"Cá dạo này phất nè .",
	},
}
var ShortCut = map[Type][]ShortCutMode{}

func InitShortCut() {
	for k, v := range ShortCutText {
		for _, v1 := range v {
			st := ShortCutMode{
				Type: "System",
				Text: v1,
			}
			ShortCut[k] = append(ShortCut[k][0:], st)
		}
	}
}
