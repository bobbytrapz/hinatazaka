package members

import (
	"strings"
)

// todo: update
// they have not moved to hinatazaka46.com yet
var lookup = map[string]string{
	"井口眞緒": "https://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=23",
	"齊藤京子": "http://www.keyakizaka46.com/s/k46o/diary/member/list?ima=0000&ct=28",
}

func init() {
	lookup["iguchi"] = lookup["井口眞緒"]
	lookup["mao"] = lookup["井口眞緒"]

	lookup["きょんこ"] = lookup["齊藤京子"]
	lookup["キョンこ"] = lookup["齊藤京子"]
	lookup["さいきょー"] = lookup["齊藤京子"]
	lookup["saito kyoko"] = lookup["齊藤京子"]
	lookup["kyoko"] = lookup["齊藤京子"]
	lookup["kyonko"] = lookup["齊藤京子"]
	lookup["saikyo"] = lookup["齊藤京子"]
}

// GetBlogURL of a member by name or nickname
func GetBlogURL(name string) string {
	return lookup[strings.ToLower(name)]
}
