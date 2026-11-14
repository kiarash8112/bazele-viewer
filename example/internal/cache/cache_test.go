package cache

import (
	"testing"
	"time"
)

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time { return c.now }

func TestEntriesExpire(t *testing.T) {
	clk := &testClock{now: time.Unix(0, 0)}
	c := New[string](time.Minute, clk)
	c.Put("k", "v")
	if v, ok := c.Get("k"); !ok || v != "v" {
		t.Fatalf("Get = %q, %v; want v, true", v, ok)
	}
	clk.now = clk.now.Add(time.Minute)
	if _, ok := c.Get("k"); ok {
		t.Error("Get after the TTL found the entry")
	}
}
