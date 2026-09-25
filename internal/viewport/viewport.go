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
