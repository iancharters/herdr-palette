package viewport

import "testing"

func TestFlatWindow(t *testing.T) {
	if got := Of(24, 14, 8); got != (Window{7, 15}) {
		t.Fatalf("got %+v", got)
	}
}

func TestGroupedCountsHeaders(t *testing.T) {
	items := []string{"Panes", "Panes", "Tabs", "Tabs", "Tabs"}
	got := Grouped(len(items), 4, 4, func(i int) string { return items[i] })
	if got != (Window{2, 5}) {
		t.Fatalf("got %+v", got)
	}
}

func TestGroupedSubgroups(t *testing.T) {
	keys := []string{"Custom\x00a", "Custom\x00a", "Custom\x00b"}
	got := Grouped(len(keys), 2, 4, func(i int) string { return keys[i] })
	// selecting first row of subgroup b must keep its subheader visible
	if got.Start > 2 || got.End < 3 {
		t.Fatalf("got %+v", got)
	}
}
