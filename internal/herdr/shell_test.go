package herdr

import "testing"

func TestIsShell(t *testing.T) {
	if !IsShell([]ForegroundProc{{Name: "bash"}}) {
		t.Fatal("bash is a shell")
	}
	if !IsShell([]ForegroundProc{}) {
		t.Fatal("empty means fresh idle shell")
	}
	if IsShell(nil) {
		t.Fatal("nil means unknown -> not shell (split, don't type)")
	}
	if IsShell([]ForegroundProc{{Name: "opencode"}}) {
		t.Fatal("opencode is not a shell")
	}
	if IsShell([]ForegroundProc{{Name: "bash"}, {Name: "nvim"}}) {
		t.Fatal("mixed means busy")
	}
}

func TestParseProcessInfo(t *testing.T) {
	procs := ParseProcessInfo(`{"result":{"process_info":{"foreground_processes":[{"name":"opencode","pid":1}],"pane_id":"w9:p2"}}}`)
	if len(procs) != 1 || procs[0].Name != "opencode" {
		t.Fatalf("%+v", procs)
	}
	if ParseProcessInfo("garbage") != nil {
		t.Fatal("garbage should be nil")
	}
}
