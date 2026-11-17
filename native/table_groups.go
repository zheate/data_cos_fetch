package main

type tableGroup struct {
	entryStart   bool
	stationStart bool
	shaded       bool
}

// Group only while entry order is intact. Sorting by a measurement must keep
// every row's identity visible, even if neighboring rows happen to match.
func tableGroups(rows []record, grouped bool) []tableGroup {
	groups := make([]tableGroup, len(rows))
	station := 0
	for i, row := range rows {
		g := &groups[i]
		g.entryStart = !grouped || i == 0 || row.Entry != rows[i-1].Entry
		g.stationStart = g.entryStart || row.Category != rows[i-1].Category
		if g.entryStart {
			station = 0
		} else if g.stationStart {
			station++
		}
		g.shaded = grouped && station%2 == 1
	}
	return groups
}
