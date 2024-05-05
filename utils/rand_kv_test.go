package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"fmt"
)

func TestGetTestKey(t *testing.T) {
	for i := 0; i < 10; i++ {
		// fmt.Println(string(GetTestKey(i)))
		assert.NotNil(t, string(GetTestKey(i)))
	}
}

func TestRandomValue(t *testing.T) {
	for i := 0; i < 10; i++ {
		fmt.Println(string(RandomValue(i)))
		assert.NotNil(t, string(RandomValue(10)))
	}
}
