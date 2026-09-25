package viewport

// Window is a visible slice [Start, End) of items.
type Window struct{ Start, End int }

// Of keeps selected visible in a flat list of capacity rows.
func Of(length, selected, capacity int) Window {
	start := max(0, min(selected-capacity+1, length-capacity))
	end := min(length, start+capacity)
	return Window{start, end}
}

// Grouped keeps selected visible when headers cost a row per group change.
// key returns the header key (category + group) for an item index.
func Grouped(length, selected, capacity int, key func(i int) string) Window {
	cost := func(start, end int) int {
		rows, prev := 0, ""
		for i := start; i < end; i++ {
			cur := key(i)
			if i == start || cur != prev {
				rows++
			}
			rows++
			prev = cur
		}
		return rows
	}
	start := selected
	for start > 0 && cost(start-1, selected+1) <= capacity {
		start--
	}
	end := selected + 1
	for end < length && cost(start, end+1) <= capacity {
		end++
	}
	return Window{start, end}
}

// GroupedGaps is Grouped plus a blank gap line before every header except
// the first rendered line (category + subgroup headings breathe).
// Prefer GroupedAdv for new code: it models the no-gap-after-category rule
// and variable item heights exactly.
func GroupedGaps(length, selected, capacity int, key func(i int) string) Window {
	cost := func(start, end int) int {
		rows, prev, first := 0, "", true
		for i := start; i < end; i++ {
			cur := key(i)
			if i == start || cur != prev {
				if !first {
					rows++ // gap before header
				}
				rows++ // header
				first = false
			}
			rows++ // item
			prev = cur
		}
		return rows
	}
	start := selected
	for start > 0 && cost(start-1, selected+1) <= capacity {
		start--
	}
	end := selected + 1
	for end < length && cost(start, end+1) <= capacity {
		end++
	}
	return Window{start, end}
}

// GroupedAdv is the exact model of the renderer: variable item heights,
// one header row per category/subgroup change, and one gap row before a
// header unless it is the first emitted line or a subgroup directly under
// its category header. cat/grp return the two header levels for an item
// (grp "" = no subgroup); h returns the item's own row count.
func GroupedAdv(length, selected, capacity int, cat, grp func(i int) string, h func(i int) int) Window {
	cost := func(start, end int) int {
		rows := 0
		pcat, pgrp, first, justH := "", "", true, false
		for i := start; i < end; i++ {
			c, g := cat(i), grp(i)
			if i == start || c != pcat {
				if !first {
					rows++ // gap
				}
				rows++ // category header
				first, justH = false, true
			}
			if g != "" && (i == start || g != pgrp || c != pcat) {
				if !justH {
					rows++ // gap
				}
				rows++ // subgroup header
				first, justH = false, true
			}
			rows += h(i)
			justH = false
			pcat, pgrp = c, g
		}
		return rows
	}
	start := selected
	for start > 0 && cost(start-1, selected+1) <= capacity {
		start--
	}
	end := selected + 1
	for end < length && cost(start, end+1) <= capacity {
		end++
	}
	return Window{start, end}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
