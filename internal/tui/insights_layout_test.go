package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestInsightsColumns(t *testing.T) {
	tests := []struct {
		vw         int
		wantColW   int
		wantTwoCol bool
	}{
		{vw: 40, wantColW: 40, wantTwoCol: false},
		{vw: 95, wantColW: 95, wantTwoCol: false}, // width 99: single column
		{vw: 96, wantColW: 46, wantTwoCol: true},  // width 100: breakpoint
		{vw: 97, wantColW: 46, wantTwoCol: true},
		{vw: 116, wantColW: 56, wantTwoCol: true},
	}
	for _, tt := range tests {
		colW, twoCol := insightsColumns(tt.vw)
		if colW != tt.wantColW || twoCol != tt.wantTwoCol {
			t.Errorf("insightsColumns(%d) = (%d,%v), want (%d,%v)", tt.vw, colW, twoCol, tt.wantColW, tt.wantTwoCol)
		}
	}
}

func TestClampBarWidth(t *testing.T) {
	tests := []struct{ bw, want int }{
		{bw: -5, want: insightsMinBarWidth},
		{bw: 0, want: insightsMinBarWidth},
		{bw: insightsMinBarWidth, want: insightsMinBarWidth},
		{bw: 12, want: 12},
		{bw: insightsMaxBarWidth, want: insightsMaxBarWidth},
		{bw: 200, want: insightsMaxBarWidth},
	}
	for _, tt := range tests {
		if got := clampBarWidth(tt.bw); got != tt.want {
			t.Errorf("clampBarWidth(%d) = %d, want %d", tt.bw, got, tt.want)
		}
	}
}

func TestBarWidthForRows(t *testing.T) {
	row := func(prefixWidth int) func(int) string {
		return func(bw int) string {
			return strings.Repeat("x", prefixWidth) + strings.Repeat("#", bw)
		}
	}

	tests := []struct {
		name string
		w    int
		rows []func(int) string
		want int
	}{
		{"single row, room to spare", 50, []func(int) string{row(20)}, insightsMaxBarWidth},
		{"single row, tight", 26, []func(int) string{row(20)}, insightsMinBarWidth},
		{"widest row wins", 50, []func(int) string{row(10), row(30)}, 20},
		{"no rows uses full width, clamped to max", 50, nil, insightsMaxBarWidth},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := barWidthForRows(tt.w, tt.rows...); got != tt.want {
				t.Errorf("barWidthForRows(%d, ...) = %d, want %d", tt.w, got, tt.want)
			}
		})
	}
}

func TestTopFoodsLayout(t *testing.T) {
	tests := []struct {
		w              int
		wantNameW      int
		wantShowAvgPts bool
		wantShowKcal   bool
	}{
		{82, foodNameW, true, true},
		{81, foodNameW, true, false},
		{67, foodNameW, true, false},
		{66, foodNameW, false, false},
		{55, foodNameW, false, false},
		{54, 33, false, false},
		{40, 19, false, false},
		{20, foodNameMinW, false, false},
	}
	for _, tt := range tests {
		got := topFoodsLayout(tt.w)
		if got.nameW != tt.wantNameW || got.showAvgPts != tt.wantShowAvgPts || got.showKcal != tt.wantShowKcal {
			t.Errorf("topFoodsLayout(%d) = %+v, want {%d,%v,%v}", tt.w, got, tt.wantNameW, tt.wantShowAvgPts, tt.wantShowKcal)
		}
	}
}

func TestZeroPointLayout(t *testing.T) {
	tests := []struct {
		w            int
		wantNameW    int
		wantShowKcal bool
	}{
		{56, foodNameW, true},
		{55, foodNameW, false},
		{41, foodNameW, false},
		{40, 33, false},
		{20, foodNameMinW, false},
	}
	for _, tt := range tests {
		got := zeroPointLayout(tt.w)
		if got.nameW != tt.wantNameW || got.showKcal != tt.wantShowKcal {
			t.Errorf("zeroPointLayout(%d) = %+v, want {%d,_,%v}", tt.w, got, tt.wantNameW, tt.wantShowKcal)
		}
	}
}

func TestPadBlock(t *testing.T) {
	styled := styleDetailLabel.Render("Points") + makeBar(3, 10, 8, false)
	natural := lipgloss.Width(styled)

	for _, w := range []int{natural + 5, natural, natural - 3} {
		got := padBlock(styled, w)
		for _, line := range strings.Split(got, "\n") {
			if lw := lipgloss.Width(line); lw != w {
				t.Errorf("padBlock(_, %d): line width = %d, want %d", w, lw, w)
			}
		}
	}
}

func TestJoinColumns(t *testing.T) {
	tall := "a\nb\nc"
	short := "x"

	t.Run("both non-empty", func(t *testing.T) {
		got := joinColumns(short, 10, tall, 10)
		lines := strings.Split(got, "\n")
		if len(lines) != 3 {
			t.Fatalf("height = %d, want 3", len(lines))
		}
		wantWidth := 10 + insightsGutter + 10
		for _, l := range lines {
			if lipgloss.Width(l) != wantWidth {
				t.Errorf("line width = %d, want %d", lipgloss.Width(l), wantWidth)
			}
		}
	})

	t.Run("right empty keeps left's own width", func(t *testing.T) {
		got := joinColumns(short, 10, "", 20)
		if lipgloss.Width(got) != 10 {
			t.Errorf("width = %d, want 10 (left's width, not expanded)", lipgloss.Width(got))
		}
	})

	t.Run("left empty keeps right's own width", func(t *testing.T) {
		got := joinColumns("", 10, short, 20)
		if lipgloss.Width(got) != 20 {
			t.Errorf("width = %d, want 20 (right's width, not expanded)", lipgloss.Width(got))
		}
	})

	t.Run("both empty", func(t *testing.T) {
		if got := joinColumns("", 10, "  ", 20); got != "" {
			t.Errorf("got %q, want empty", got)
		}
	})
}

func TestSectionHeading(t *testing.T) {
	long, short := "A Very Long Section Title", "Short"
	tests := []struct {
		w         int
		wantTitle string
	}{
		{w: 100, wantTitle: long},
		{w: lipgloss.Width(long), wantTitle: long},
		{w: lipgloss.Width(long) - 1, wantTitle: short},
		{w: 2, wantTitle: short}, // narrower than even the short form
	}
	for _, tt := range tests {
		got := sectionHeading(long, short, tt.w)
		firstLine := strings.SplitN(got, "\n", 2)[0]
		if !strings.Contains(firstLine, tt.wantTitle) {
			t.Errorf("sectionHeading(_,_,%d) first line = %q, want to contain %q", tt.w, firstLine, tt.wantTitle)
		}
	}
}
