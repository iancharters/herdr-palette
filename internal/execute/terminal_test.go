package execute

import "testing"

func TestDisplayArgvPlugin(t *testing.T) {
	got := displayArgv([]string{"plugin", "action", "invoke", "a.b.c"})
	if len(got) != 3 || got[1] != "invoke" || got[2] != "a.b.c" {
		t.Fatalf("%v", got)
	}
}

func TestDisplayArgvHerdr(t *testing.T) {
	got := displayArgv([]string{"tab", "focus", "w1:t2"})
	if len(got) != 4 || got[0] != "herdr" || got[1] != "tab" {
		t.Fatalf("%v", got)
	}
}

func TestInvokeRejectsBareID(t *testing.T) {
	_, _, code := Invoke("bare")
	if code == 0 {
		t.Fatal("bare action id must fail")
	}
}
