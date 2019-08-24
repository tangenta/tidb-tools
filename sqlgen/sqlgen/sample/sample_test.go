package sample

import (
	"fmt"
	"testing"
)

func TestA(t *testing.T) {
	for i := 0; i < 10; i++ {
		counter = map[string]int{
			"A": 2,
		}
		fmt.Println(Generate())
	}
}
