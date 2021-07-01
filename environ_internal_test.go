package environ

import (
	"testing"
)

// TestLocks checks for locking functions missing the second ()
// in their defer call.
func TestLocks(t *testing.T) {
	a := New([]string{"A=A"})
	fl := fakeLocker{}
	a.l = &fl

	_ = a.Len()
	_ = a.Keys()
	_ = a.AsSlice()
	_ = a.AsMap()
	_, _ = a.Keep("A")
	_, _ = a.Drop("A")
	_, _ = a.MarshalJSON()
	_ = a.Get("A")
	a.Set("B", "B")
	a.Unset("B")

	if fl.locks != 0 {
		t.Errorf("locks was non-zero %d, search for 'writeLocker\\(\\)$' and add additional parens", fl.locks)
	}

	if fl.rlocks != 0 {
		t.Errorf("rlocks was non-zero %d, search for 'readLocker\\(\\)$' and add additional parens", fl.rlocks)
	}
}

type fakeLocker struct {
	rlocks, locks int
}

func (f *fakeLocker) RLock() {
	f.rlocks++
}

func (f *fakeLocker) RUnlock() {
	f.rlocks--
}

func (f *fakeLocker) Lock() {
	f.locks++
}

func (f *fakeLocker) Unlock() {
	f.locks--
}
