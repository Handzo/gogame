package service

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	_, err := Parse("::::0:2::0:3:0:0:62a2c2c07371a3a05172508160b09093b2829183928061b3:c3635253c1b1a170:2:0")
	fmt.Println(err)
}
