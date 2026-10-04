package main

import "testing"

func TestParseArgs(t *testing.T) {
	c, err := parseArgs([]string{"fresh", "--seed", "--force", "--redis"})
	if err != nil || c.name != "fresh" || !c.seed || !c.force || !c.redis {
		t.Fatalf("got %+v %v", c, err)
	}
	if c, err := parseArgs([]string{"migrate"}); err != nil || c.name != "migrate" || c.seed {
		t.Fatalf("got %+v %v", c, err)
	}
	for _, bad := range [][]string{nil, {"drop"}, {"migrate", "--seed"}, {"fresh", "--nope"}, {"status", "x"}} {
		if _, err := parseArgs(bad); err == nil {
			t.Errorf("%v: want an error", bad)
		}
	}
}
