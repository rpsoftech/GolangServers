package main

import (
	"fmt"
	"strconv"
)

func main() {
	num64 := int64(12345)
	str := strconv.FormatInt(num64, 10)    // Decimal string
	hexStr := strconv.FormatInt(num64, 16) // Hexadecimal string

	fmt.Println("Decimal string:", str)
	fmt.Println("Hexadecimal string:", hexStr)
}
