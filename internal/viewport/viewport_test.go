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

func TestGroupedGapsCountsGapLines(t *testing.T) {
	keys := []string{"Custom\x00a", "Custom\x00a", "Custom\x00b"}
	// 3 items + 2 headers + 1 gap = 6 rows; capacity 6 fits all
	if got := GroupedGaps(len(keys), 2, 6, func(i int) string { return keys[i] }); got != (Window{0, 3}) {
		t.Fatalf("got %+v", got)
	}
	// capacity 4 cannot fit the gap + second header + item, so it must window
	if got := GroupedGaps(len(keys), 2, 4, func(i int) string { return keys[i] }); got.End != 3 || got.Start > 2 {
		t.Fatalf("got %+v", got)
	}
}

func TestGroupedAdvSkipsGapAfterCategory(t *testing.T) {
	cats := []string{"Custom", "Custom"}
	grps := []string{"a", "a"}
	one := func(i int) int { return 1 }
	cat := func(i int) string { return cats[i] }
	grp := func(i int) string { return grps[i] }
	// header + subheader (no gap between) + 2 items = 4 rows fits exactly
	if got := GroupedAdv(2, 1, 4, cat, grp, one); got != (Window{0, 2}) {
		t.Fatalf("got %+v", got)
	}
}

func TestGroupedAdvVariableHeights(t *testing.T) {
	cats := []string{"A", "A", "B"}
	grps := []string{"", "", ""}
	hs := []int{1, 3, 1}
	cat := func(i int) string { return cats[i] }
	grp := func(i int) string { return grps[i] }
	h := func(i int) int { return hs[i] }
	// rows: hdr(1) + item0(1) + item1(3) = 5 <= 6, so the window keeps
	// the earlier rows: extending to item2 would add gap + hdr + item = 8 > 6.
	if got := GroupedAdv(3, 1, 6, cat, grp, h); got != (Window{0, 2}) {
		t.Fatalf("got %+v", got)
	}
}
