package commands

import "testing"

func TestHashBench(t *testing.T) {
	hb := NewHashBench()
	result := hb.DoTest()
	if result == 0 || result > 5000 {
		t.Errorf("Error. Invalid result %d", result)
	}
}
