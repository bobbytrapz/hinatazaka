package members

import (
	"strings"
)

// Blogs maps names to blog list links
// todo: update
// they have not moved to hinatazaka46.com yet
var Blogs = map[string]string{
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

// nicknames of members
var nicknames = map[string]string{
	"iguchi": "井口眞緒",
	"mao":    "井口眞緒",
	"mama":   "井口眞緒",
	"bau":    "井口眞緒",
	"ママ":     "井口眞緒",
	"ばう":     "井口眞緒",

	"ushio":  "潮紗理菜",
	"sarina": "潮紗理菜",

	"kakizaki": "柿崎芽実",
	"memi":     "柿崎芽実",

	"kageyama": "影山優佳",
	"yuuka":    "影山優佳",
	"kage":     "影山優佳",
	"kagechan": "影山優佳",
	"影ちゃん":     "影山優佳",

	"kato":    "加藤史帆",
	"shiho":   "加藤史帆",
	"katoshi": "加藤史帆",

	"きょんこ":        "齊藤京子",
	"キョンこ":        "齊藤京子",
	"さいきょー":       "齊藤京子",
	"ラーメン":        "齊藤京子",
	"saito kyoko": "齊藤京子",
	"kyoko":       "齊藤京子",
	"kyonko":      "齊藤京子",
	"saikyo":      "齊藤京子",
	"ramen":       "齊藤京子",

	"kumi":    "佐々木久美",
	"captain": "佐々木久美",

	"mirei":  "佐々木美玲",
	"miipan": "佐々木美玲",

	"takase": "高瀬愛奈",
	"mana":   "高瀬愛奈",
	"manafi": "高瀬愛奈",

	"takamoto": "高本彩花",
	"ayaka":    "高本彩花",
	"otake":    "高本彩花",
	"おたけ":      "高本彩花",

	"higashimura": "東村芽依",
	"mei":         "東村芽依",
	"meimei":      "東村芽依",

	"kanemura": "金村美玖",
	"miku":     "金村美玖",
	"sushi":    "金村美玖",

	"kawata": "河田陽菜",
	"hina":   "河田陽菜",

	"kosaka": "小坂菜緒",
	"nao":    "小坂菜緒",

	"tomita": "富田鈴花",
	"suzuka": "富田鈴花",
	"paripi": "富田鈴花",
	"パリピ":    "富田鈴花",

	"nibu":  "丹生明里",
	"akari": "丹生明里",

	"hamagishi": "濱岸ひより",
	"hiyori":    "濱岸ひより",
	"hiyotan":   "濱岸ひより",
	"ひよたん":      "濱岸ひより",

	"matsuda": "松田好花",
	"konoka":  "松田好花",

	"miyata": "宮田愛萌",
	"manamo": "宮田愛萌",

	"watanabe": "渡邉美穂",
	"miho":     "渡邉美穂",
	"bemiho":   "渡邉美穂",

	"kamimura":   "上村ひなの",
	"hinano":     "上村ひなの",
	"hinanonano": "上村ひなの",
	"nano":       "上村ひなの",
	"ひなの":        "上村ひなの",
}

// RealName provides the real name of a member
func RealName(name string) string {
	name = strings.ToLower(name)
	if realname, ok := nicknames[name]; ok {
		// name is a nickname so use real name
		return realname
	}
	return name
}

// BlogURL of a member by name or nickname
func BlogURL(name string) string {
	return Blogs[RealName(name)]
}
