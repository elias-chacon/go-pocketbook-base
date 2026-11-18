package inkview

import "testing"

func TestBackKeyIsLeftAndMenuBecauseThePB741HasNoBack(t *testing.T) {
	for _, code := range []int{KeyBack, KeyPrev, KeyMenu} {
		if !IsBackKey(code) {
			t.Errorf("key %#x must act as Back", code)
		}
	}
	for _, code := range []int{KeyNext, KeyOK, 0x41} {
		if IsBackKey(code) {
			t.Errorf("key %#x must not act as Back", code)
		}
	}
}
