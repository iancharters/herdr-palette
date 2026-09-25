package execute

import "testing"

func TestParseInvokeResponse(t *testing.T) {
	id, pid := ParseInvokeResponse(`{"result":{"log":{"log_id":"plugin-log-20","plugin_id":"ntindle.herdr-resurrect"}}}`)
	if id != "plugin-log-20" || pid != "ntindle.herdr-resurrect" {
		t.Fatalf("got %q %q", id, pid)
	}
	if a, b := ParseInvokeResponse("garbage"); a != "" || b != "" {
		t.Fatalf("garbage should be empty, got %q %q", a, b)
	}
}

func TestFindLogEntry(t *testing.T) {
	logs := `{"result":{"logs":[{"log_id":"plugin-log-19","status":"succeeded","exit_code":0,"stdout":"hello","stderr":""}]}}`
	e := FindLogEntry(logs, "plugin-log-19")
	if e == nil || e.Status != "succeeded" || e.Stdout != "hello" {
		t.Fatalf("got %+v", e)
	}
	if FindLogEntry(logs, "missing") != nil {
		t.Fatal("missing id should be nil")
	}
}
