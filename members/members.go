package members

import (
	"strings"
)

// todo: update
// they have not moved to hinatazaka46.com yet
var lookup = map[string]string{
	"井口眞緒":  "https://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=23",
	"潮紗理菜":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=24",
	"柿崎芽実":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=25",
	"影山優佳":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=26",
	"加藤史帆":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=27",
	"齊藤京子":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=28",
	"佐々木久美": "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=29",
	"佐々木美玲": "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=30",
	"高瀬愛奈":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=31",
	"高本彩花":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=32",
	"東村芽依":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=33",
	"金村美玖":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=34",
	"河田陽菜":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=35",
	"小坂菜緒":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=36",
	"富田鈴花":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=37",
	"丹生明里":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=38",
	"濱岸ひより": "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=39",
	"松田好花":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=40",
	"宮田愛萌":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=41",
	"渡邉美穂":  "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=42",
	"上村ひなの": "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=52",
}

func init() {
	lookup["iguchi"] = lookup["井口眞緒"]
	lookup["mao"] = lookup["井口眞緒"]

	lookup["ushio"] = lookup["潮紗理菜"]
	lookup["sarina"] = lookup["潮紗理菜"]

	lookup["kakizaki"] = lookup["柿崎芽実"]
	lookup["memi"] = lookup["柿崎芽実"]

	lookup["kageyama"] = lookup["影山優佳"]
	lookup["yuuka"] = lookup["影山優佳"]
	lookup["kage"] = lookup["影山優佳"]
	lookup["kagechan"] = lookup["影山優佳"]
	lookup["影ちゃん"] = lookup["影山優佳"]

	lookup["kato"] = lookup["加藤史帆"]
	lookup["shiho"] = lookup["加藤史帆"]
	lookup["katoshi"] = lookup["加藤史帆"]

	lookup["きょんこ"] = lookup["齊藤京子"]
	lookup["キョンこ"] = lookup["齊藤京子"]
	lookup["さいきょー"] = lookup["齊藤京子"]
	lookup["ラーメン"] = lookup["齊藤京子"]
	lookup["saito kyoko"] = lookup["齊藤京子"]
	lookup["kyoko"] = lookup["齊藤京子"]
	lookup["kyonko"] = lookup["齊藤京子"]
	lookup["saikyo"] = lookup["齊藤京子"]
	lookup["ramen"] = lookup["齊藤京子"]

	lookup["sasaki"] = lookup["佐々木久美"]
	lookup["kumi"] = lookup["佐々木久美"]
	lookup["captain"] = lookup["佐々木久美"]

	lookup["sasaki"] = lookup["佐々木美玲"]
	lookup["mirei"] = lookup["佐々木美玲"]
	lookup["miipan"] = lookup["佐々木美玲"]

	lookup["takase"] = lookup["高瀬愛奈"]
	lookup["mana"] = lookup["高瀬愛奈"]
	lookup["manafi"] = lookup["高瀬愛奈"]

	lookup["takamoto"] = lookup["高本彩花"]
	lookup["ayaka"] = lookup["高本彩花"]
	lookup["otake"] = lookup["高本彩花"]
	lookup["おたけ"] = lookup["高本彩花"]

	lookup["higashimura"] = lookup["東村芽依"]
	lookup["mei"] = lookup["東村芽依"]
	lookup["meimei"] = lookup["東村芽依"]

	lookup["kanemura"] = lookup["金村美玖"]
	lookup["miku"] = lookup["金村美玖"]
	lookup["sushi"] = lookup["金村美玖"]

	lookup["kawata"] = lookup["河田陽菜"]
	lookup["hina"] = lookup["河田陽菜"]

	lookup["kosaka"] = lookup["小坂菜緒"]
	lookup["nao"] = lookup["小坂菜緒"]

	lookup["tomita"] = lookup["富田鈴花"]
	lookup["suzuka"] = lookup["富田鈴花"]
	lookup["paripi"] = lookup["富田鈴花"]
	lookup["パリピ"] = lookup["富田鈴花"]

	lookup["nibu"] = lookup["丹生明里"]
	lookup["akari"] = lookup["丹生明里"]

	lookup["hamagishi"] = lookup["濱岸ひより"]
	lookup["hiyori"] = lookup["濱岸ひより"]
	lookup["hiyotan"] = lookup["濱岸ひより"]
	lookup["ひよたん"] = lookup["濱岸ひより"]

	lookup["matsuda"] = lookup["松田好花"]
	lookup["konoka"] = lookup["松田好花"]

	lookup["miyata"] = lookup["宮田愛萌"]
	lookup["manamo"] = lookup["宮田愛萌"]

	lookup["watanabe"] = lookup["渡邉美穂"]
	lookup["miho"] = lookup["渡邉美穂"]

	lookup["kamimura"] = lookup["上村ひなの"]
	lookup["hinano"] = lookup["上村ひなの"]
	lookup["hinanonano"] = lookup["上村ひなの"]
	lookup["nano"] = lookup["上村ひなの"]
}

// GetBlogURL of a member by name or nickname
func GetBlogURL(name string) string {
	return lookup[strings.ToLower(name)]
}
