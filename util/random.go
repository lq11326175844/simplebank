package util

import (
	"math/rand"
	"strings"
	"time"
)

const alphabet = "abcdefghijklmnopqrstuvwxvz"

func init() {
	rand.Seed(time.Now().UnixNano())
}

// 随机生成一个再min和max之间的数
func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1)
}

//随机生成n个字符组成的字符串

func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)
	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}
	return sb.String()
}

//随机生成owner名字

func RandomOwner() string {
	return RandomString(6)
}

//随机生成monny

func RandomMonny() int64 {
	return RandomInt(0, 1000)
}

// 随机生成货币类型
func RandomCurrency() string {
	currency := []string{"EUR", "USD", "CAD", "RMB"}
	n := len(currency)
	return currency[rand.Intn(n)]
}
