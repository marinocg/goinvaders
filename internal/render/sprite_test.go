package render

import (
	"strings"
	"testing"
)

func TestSpriteMatricesHaveFixedDimensions(t *testing.T) {
	for tier, entities := range sprites {
		for glyph, art := range entities {
			if len(art.rows) == 0 || len(art.rows[0]) == 0 {
				t.Fatalf("tier %d glyph %q is empty", tier, glyph)
			}
			for _, row := range art.rows {
				if len(row) != len(art.rows[0]) {
					t.Fatalf("tier %d glyph %q has ragged rows", tier, glyph)
				}
			}
		}
	}
}

func TestSpriteMatricesMatchVisualLanguage(t *testing.T) {
	want := map[spriteTier]map[byte]sprite{
		compactTier: {
			'A': {rows: []string{"A"}}, 'W': {rows: []string{"W"}}, '|': {rows: []string{"|"}},
		},
		mediumTier: {
			'A': {rows: []string{" /^\\ ", "/###\\"}, anchorX: 2, anchorY: 1},
			'W': {rows: []string{"/W W\\", " WWW "}, anchorX: 2, anchorY: 1},
			'|': {rows: []string{"|", "|"}, anchorY: 1},
		},
		largeTier: {
			'A': {rows: []string{"  /A\\  ", " /###\\ ", "/#####\\"}, anchorX: 3, anchorY: 2},
			'W': {rows: []string{" /W W\\ ", "/WWWWW\\", " WWWWW "}, anchorX: 3, anchorY: 1},
			'|': {rows: []string{" | ", " | ", " | "}, anchorX: 1, anchorY: 1},
		},
	}
	for tier, entities := range want {
		for glyph, expected := range entities {
			if got := sprites[tier][glyph]; got.anchorX != expected.anchorX || got.anchorY != expected.anchorY || strings.Join(got.rows, "\n") != strings.Join(expected.rows, "\n") {
				t.Errorf("tier %d glyph %q = %#v, want %#v", tier, glyph, got, expected)
			}
		}
	}
}

func TestEntitySpritesHaveExplicitResponsiveAssets(t *testing.T) {
	for _, glyph := range []byte{'v', 'B', '1', '2'} {
		for _, tier := range []spriteTier{compactTier, mediumTier, largeTier} {
			art, ok := sprites[tier][glyph]
			if !ok || len(art.rows) == 0 {
				t.Fatalf("missing tier %d asset for %q", tier, glyph)
			}
		}
	}
	if got := sprites[mediumTier]['v']; strings.Join(got.rows, "\n") != "\\/\n/\\" {
		t.Fatalf("medium enemy shot = %#v", got)
	}
	if got := sprites[largeTier]['B']; strings.Join(got.rows, "\n") != " /B\\ \n<--->\n \\_/ " {
		t.Fatalf("large bonus target = %#v", got)
	}
	if got := sprites[largeTier]['v']; strings.Join(got.rows, "\n") != " /\\  \n  v  \n \\/  " {
		t.Fatalf("large enemy shot = %#v", got)
	}
}

func TestSelectSpriteTierUsesFieldDensity(t *testing.T) {
	if got := selectSpriteTier(projection{minX: 1, maxX: 38, minY: 1, maxY: 10}); got != compactTier {
		t.Fatalf("minimum tier = %d, want compact", got)
	}
	if got := selectSpriteTier(projection{minX: 1, maxX: 50, minY: 1, maxY: 14}); got != mediumTier {
		t.Fatalf("medium tier = %d, want medium", got)
	}
	if got := selectSpriteTier(projection{minX: 1, maxX: 80, minY: 1, maxY: 20}); got != largeTier {
		t.Fatalf("large tier = %d, want large", got)
	}
}

func TestSpriteForDowngradesAtPlayfieldEdges(t *testing.T) {
	field := projection{minX: 1, maxX: 20, minY: 1, maxY: 10}
	if got, ok := spriteFor('W', largeTier, 1, 1, field); !ok || len(got.rows) != 1 {
		t.Fatalf("corner sprite = %#v, %v; want compact fallback", got, ok)
	}
	if _, ok := spriteFor('W', largeTier, 10, 5, field); !ok {
		t.Fatal("center sprite was rejected")
	}
}

func TestDefaultFormationMediumSpritesRemainSeparated(t *testing.T) {
	field := projection{minX: 1, maxX: 74, minY: 1, maxY: 20}
	leftX, _ := projectPoint(viewport{originX: 0, originY: 0, width: 77, height: 27}, 40, 18, 9, 0)
	rightX, _ := projectPoint(viewport{originX: 0, originY: 0, width: 77, height: 27}, 40, 18, 12, 0)
	left, ok := spriteFor('W', mediumTier, leftX, 5, field)
	if !ok {
		t.Fatal("left sprite unavailable")
	}
	right, ok := spriteFor('W', mediumTier, rightX, 5, field)
	if !ok {
		t.Fatal("right sprite unavailable")
	}
	leftEnd := leftX - left.anchorX + len(left.rows[0])
	rightStart := rightX - right.anchorX
	if leftEnd > rightStart {
		t.Fatalf("formation sprites overlap: left ends %d, right starts %d", leftEnd, rightStart)
	}
}

func TestFormationTierDowngradesAtRequiredThresholds(t *testing.T) {
	tests := []struct {
		name    string
		field   projection
		centers []int
		want    spriteTier
	}{
		{"medium threshold", projection{minX: 1, maxX: 44, minY: 1, maxY: 6}, []int{8, 11, 15, 18, 22, 25, 29, 32}, compactTier},
		{"large threshold", projection{minX: 1, maxX: 79, minY: 1, maxY: 19}, []int{16, 22, 28, 34, 40, 46, 52, 58}, mediumTier},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tier := mediumTier
			if tt.name == "large threshold" {
				tier = largeTier
			}
			got := formationSpriteTier(tt.field, tier, tt.centers)
			if got != tt.want {
				t.Fatalf("tier = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFormationTierIgnoresRepeatedColumnsAcrossRows(t *testing.T) {
	field := projection{minX: 1, maxX: 74, minY: 1, maxY: 20}
	centers := []int{8, 14, 20, 26, 32, 38, 44, 50}
	centers = append(centers, centers...)
	if got := formationSpriteTier(field, mediumTier, centers); got != mediumTier {
		t.Fatalf("tier with repeated row columns = %d, want medium", got)
	}

	field = projection{minX: 1, maxX: 79, minY: 1, maxY: 19}
	centers = []int{16, 24, 32, 40, 48, 56, 64, 72}
	centers = append(centers, centers...)
	if got := formationSpriteTier(field, largeTier, centers); got != largeTier {
		t.Fatalf("large tier with repeated row columns = %d, want large", got)
	}
}
