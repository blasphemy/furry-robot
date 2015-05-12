package utils

import (
	"math/rand"
)

var (
	base62Chars = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	base62Base  = uint64(len(base62Chars))
)

func Base62Encode(num uint64) string {
	if num == 0 {
		return "0"
	}
	arr := []uint8{}
	for num > 0 {
		rem := num % base62Base
		num = num / base62Base
		arr = append(arr, base62Chars[rem])
	}
	for i, j := 0, len(arr)-1; i < j; i, j = i+1, j-1 {
		arr[i], arr[j] = arr[j], arr[i]
	}
	return string(arr)
}

func Base62Rand(n int) string {
	var s string
	for i := 0; i < n; i++ {
		s = s + string(base62Chars[rand.Intn(len(base62Chars))])
	}
	return s
}
