package plugins

import "testing"

var sample = []DiscoveredAction{
	{PluginID: "ntindle.herdr-resurrect", ActionID: "save", Title: "Resurrect: save snapshot"},
	{PluginID: "ntindle.herdr-resurrect", ActionID: "restore", Title: "Resurrect: restore last snapshot"},
	{PluginID: "other.plugin", ActionID: "do.thing", Title: "Do thing"},
}

func TestQualifiedIDKeepsDots(t *testing.T) {
	if QualifiedID("other.plugin", "do.thing") != "other.plugin.do.thing" {
		t.Fatal("bad qid")
	}
}

func TestToPaletteItems(t *testing.T) {
	items := ToPaletteItems(sample, map[string]string{})
	if len(items) != 3 {
		t.Fatalf("len %d", len(items))
	}
	// sorted by plugin then title
	if items[0].Group != "ntindle.herdr-resurrect" || items[2].Group != "other.plugin" {
		t.Fatalf("groups %+v", items)
	}
	if items[0].Title != "Resurrect: restore last snapshot" {
		t.Fatalf("sort %+v", items)
	}
	if len(items[0].Invocation.Argv) != 4 || items[0].Invocation.Argv[3] != "ntindle.herdr-resurrect.restore" {
		t.Fatalf("argv %+v", items[0].Invocation)
	}
}

func TestParseActionListFailOpen(t *testing.T) {
	if got := ParseActionList(`{"result":{"actions":[]}}`); len(got) != 0 {
		t.Fatal("empty")
	}
	if got := ParseActionList("not json"); len(got) != 0 {
		t.Fatal("garbage should be empty")
	}
}

func TestParseShortcuts(t *testing.T) {
	src := "[[keys.command]]\nkey = \"prefix+ctrl+s\"\ntype = \"plugin_action\"\ncommand = \"ntindle.herdr-resurrect.save\"\n"
	m := ParsePluginActionShortcuts(src)
	if m["ntindle.herdr-resurrect.save"] != "prefix+ctrl+s" {
		t.Fatalf("%v", m)
	}
}
