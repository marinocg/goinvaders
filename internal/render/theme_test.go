package render

import "testing"

func TestArenaThemeBorderIsCompleteASCIIFrame(t *testing.T) {
	grid := make([][]byte, 6)
	for y := range grid {
		grid[y] = blankLine(12)
	}
	drawArenaBorder(func(x, y int, value byte) { grid[y][x] = value }, 12, 5, defaultArenaTheme)
	if string(grid[0]) != "+==========+" || string(grid[5]) != "+==========+" {
		t.Fatalf("border rows = %q, %q", grid[0], grid[5])
	}
	for y := 1; y < 5; y++ {
		if grid[y][0] != '|' || grid[y][11] != '|' {
			t.Fatalf("border sides missing at row %d: %q", y, grid[y])
		}
	}
}

func TestThemeDecorationIsStableAndSkipsReservedCells(t *testing.T) {
	field := projection{minX: 0, maxX: 79, minY: 0, maxY: 30}
	makeGrid := func() [][]byte {
		grid := make([][]byte, 31)
		for y := range grid {
			grid[y] = blankLine(80)
		}
		return grid
	}
	a, b := makeGrid(), makeGrid()
	reserved := func(x, y int) bool { return x == 10 && y == 10 }
	decorateArena(a, field, reserved)
	decorateArena(b, field, reserved)
	for y := range a {
		for x := range a[y] {
			if a[y][x] != b[y][x] {
				t.Fatalf("decoration changed at %d,%d", x, y)
			}
		}
	}
	if a[10][10] != ' ' {
		t.Fatal("decoration occupied a reserved cell")
	}
	if countThemeStars(a) == 0 {
		t.Fatal("large arena received no decoration")
	}
}

func TestThemeDecorationDisabledForCompactField(t *testing.T) {
	if themeDecorationEnabled(projection{minX: 0, maxX: 36, minY: 0, maxY: 5}) {
		t.Fatal("compact field enabled decoration")
	}
}

func TestThemeDecorationSkipsCompleteProjectilePath(t *testing.T) {
	field := projection{minX: 0, maxX: 79, minY: 0, maxY: 30}
	grid := make([][]byte, 31)
	for y := range grid {
		grid[y] = blankLine(80)
	}
	projectileX, projectileY := 40, 25
	decorateArena(grid, field, func(x, y int) bool {
		return x == projectileX && y <= projectileY
	})
	for y := field.minY + 1; y <= projectileY; y++ {
		if grid[y][projectileX] == '.' {
			t.Fatalf("decoration occupied projectile path at %d,%d", projectileX, y)
		}
	}
	if grid[projectileY+1][projectileX] != '.' && starAt(projectileX, projectileY+1, field) {
		t.Fatalf("test setup expected a decoration immediately beyond projectile path")
	}
}

func TestThemeDecorationIsPresentationOnlyForEntityAssets(t *testing.T) {
	field := projection{minX: 0, maxX: 79, minY: 0, maxY: 30}
	grid := make([][]byte, 31)
	for y := range grid {
		grid[y] = blankLine(80)
	}
	decorateArena(grid, field, func(x, y int) bool { return x == 40 && y <= 20 })
	if grid[20][40] != ' ' {
		t.Fatalf("reserved projectile cell was decorated: %q", grid[20][40])
	}
	if sprites[compactTier]['v'].rows[0] == sprites[compactTier]['|'].rows[0] {
		t.Fatal("compact enemy and player projectiles share an ambiguous glyph")
	}
}

func countThemeStars(grid [][]byte) int {
	count := 0
	for _, row := range grid {
		for _, cell := range row {
			if cell == '.' {
				count++
			}
		}
	}
	return count
}

func TestStyleForRolesAreResettableAndDistinct(t *testing.T) {
	seen := map[string]bool{}
	for _, role := range []emphasisRole{roleTitle, roleHUD, roleControls, roleStatus, roleWin, roleGameOver} {
		style := styleFor(role)
		if style == "" || seen[style] {
			t.Fatalf("role %d has missing or duplicate style %q", role, style)
		}
		seen[style] = true
	}
	if ansiReset != "\x1b[0m" {
		t.Fatalf("reset = %q", ansiReset)
	}
}
