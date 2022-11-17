package main

import (
	"testing"
)

func Test(t *testing.T) {
	var first []string
	second := make([]string, len(first))
	actual := len(second)
	if actual != 0 {
		t.Errorf("expected %v but got %v",
			0, actual)
	}
}
