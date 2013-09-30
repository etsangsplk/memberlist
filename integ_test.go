package memberlist

import (
	"os"
	"testing"
	"time"
)

// CheckInteg will skip a test if integration testing is not enabled.
func CheckInteg(t *testing.T) {
	if !IsInteg() {
		t.SkipNow()
	}
}

// IsInteg returns a boolean telling you if we're in integ testing mode.
func IsInteg() bool {
	return os.Getenv("INTEG_TESTS") != ""
}

// Tests the memberlist by creating a cluster of 100 nodes
// and checking that we get strong convergence of changes.
func TestMemberlist_Integ(t *testing.T) {
	CheckInteg(t)

	num := 16
	var members []*Memberlist

	ch := make(chan *Node, num)
	leaveCh := make(chan *Node, num)

	for i := 0; i < num; i++ {
		addr, _ := GetBindAddr()
		c := DefaultConfig()
		c.Name = addr
		c.BindAddr = addr
		c.RTT = 100 * time.Microsecond
		c.ProbeInterval = 5 * time.Millisecond
		c.GossipInterval = 3 * time.Millisecond
		c.PushPullInterval = 20 * time.Millisecond

		if i == 0 {
			m, err := Create(c)
			if err != nil {
				t.Fatalf("unexpected err: %s", err)
			}
			members = append(members, m)
			defer m.Shutdown()
			m.config.JoinCh = ch
			m.config.LeaveCh = leaveCh
		} else {
			last := members[i-1]
			m, err := Join(c, []string{last.config.Name})
			if err != nil {
				t.Fatalf("unexpected err: %s", err)
			}
			members = append(members, m)
			defer m.Shutdown()
		}
	}

	// Wait for node 1 to see all the others
	for i := 1; i < num; i++ {
		select {
		case <-ch:
		case l := <-leaveCh:
			t.Fatalf("unexpected leave %s", l)
		case <-time.After(10 * time.Millisecond):
			t.Fatalf("timeout")
		}
	}
	time.Sleep(10 * time.Millisecond)

	for idx, m := range members {
		if len(m.Members()) != num {
			t.Fatalf("bad num %d at idx %d", len(m.Members()), idx)
		}
	}
}
