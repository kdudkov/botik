package main

import (
	"fmt"
	"testing"
)

func TestInf(t *testing.T) {
	i := NewInflux()
	fmt.Println(i.getPressure("kott", 50))
}
