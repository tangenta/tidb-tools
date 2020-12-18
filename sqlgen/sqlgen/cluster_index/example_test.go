package cluster_index

import (
	"fmt"
	"testing"
)

func TestA(t *testing.T) {
	gen := NewGenerator(NewState())
	for i := 0; i < 20; i++ {
		fmt.Println(gen())
	}
}
