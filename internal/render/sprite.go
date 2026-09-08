package render

import "sort"

// spriteTier is presentation-only; it never changes a game coordinate.
type spriteTier uint8

const (
	compactTier spriteTier = iota
	mediumTier
	largeTier
)

type sprite struct {
	rows             []string
	anchorX, anchorY int
}

var sprites = map[spriteTier]map[byte]sprite{
	compactTier: {
		'A': {rows: []string{"A"}}, 'W': {rows: []string{"W"}}, '|': {rows: []string{"|"}}, 'v': {rows: []string{"v"}},
		'B': {rows: []string{"B"}}, '1': {rows: []string{"."}}, '2': {rows: []string{"#"}},
	},
	mediumTier: {
		'A': {rows: []string{" /^\\ ", "/###\\"}, anchorX: 2, anchorY: 1},
		'W': {rows: []string{"/W W\\", " WWW "}, anchorX: 2, anchorY: 1},
		'|': {rows: []string{"|", "|"}, anchorY: 1},
		'v': {rows: []string{"\\/", "/\\"}, anchorX: 1, anchorY: 1},
		'B': {rows: []string{"-B-", "---"}, anchorX: 1, anchorY: 1}, '1': {rows: []string{"+"}}, '2': {rows: []string{"#"}},
	},
	largeTier: {
		'A': {rows: []string{"  /A\\  ", " /###\\ ", "/#####\\"}, anchorX: 3, anchorY: 2},
		'W': {rows: []string{" /W W\\ ", "/WWWWW\\", " WWWWW "}, anchorX: 3, anchorY: 1},
		'|': {rows: []string{" | ", " | ", " | "}, anchorX: 1, anchorY: 1},
		'v': {rows: []string{" /\\  ", "  v  ", " \\/  "}, anchorX: 2, anchorY: 1},
		'B': {rows: []string{" /B\\ ", "<--->", " \\_/ "}, anchorX: 2, anchorY: 1}, '1': {rows: []string{"+"}}, '2': {rows: []string{"#"}},
	},
}

func selectSpriteTier(field projection) spriteTier {
	w, h := field.maxX-field.minX+1, field.maxY-field.minY+1
	if w >= 80 && h >= 20 {
		return largeTier
	}
	if w >= 50 && h >= 14 {
		return mediumTier
	}
	return compactTier
}

// formationSpriteTier prevents neighboring projected enemies from merging.
// It changes presentation only; the logical coordinates remain untouched.
func formationSpriteTier(field projection, tier spriteTier, centers []int) spriteTier {
	centers = append([]int(nil), centers...)
	sort.Ints(centers)
	// Each formation row reuses the same projected X centers. Compare unique
	// columns so vertically separated enemies do not force every sprite to
	// compact.
	uniqueCenters := centers[:0]
	for _, center := range centers {
		if len(uniqueCenters) == 0 || center != uniqueCenters[len(uniqueCenters)-1] {
			uniqueCenters = append(uniqueCenters, center)
		}
	}
	for tier > compactTier {
		width := len(sprites[tier]['W'].rows[0])
		separated := true
		for i := 1; i < len(uniqueCenters); i++ {
			if uniqueCenters[i]-uniqueCenters[i-1] < width {
				separated = false
				break
			}
		}
		if separated {
			return tier
		}
		tier--
	}
	return compactTier
}

func spriteFor(glyph byte, tier spriteTier, x, y int, field projection) (sprite, bool) {
	for current := tier; ; current-- {
		s, ok := sprites[current][glyph]
		if ok && x-s.anchorX >= field.minX && x-s.anchorX+len(s.rows[0])-1 <= field.maxX && y-s.anchorY >= field.minY && y-s.anchorY+len(s.rows)-1 <= field.maxY {
			return s, true
		}
		if current == compactTier {
			break
		}
	}
	return sprite{}, false
}
