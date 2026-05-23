package tui

import (
	"strings"

	xansi "github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/lipgloss"
)

// overlayCenter composites fg centered over bg.
// bgW and bgH are the terminal dimensions used for centering.
func overlayCenter(bg, fg string, bgW, bgH int) string {
	fgLines := strings.Split(fg, "\n")
	fgH := len(fgLines)
	fgW := 0
	for _, l := range fgLines {
		if w := lipgloss.Width(l); w > fgW {
			fgW = w
		}
	}
	if fgW <= 0 || fgH <= 0 || bgW <= 0 {
		return bg
	}

	startX := (bgW - fgW) / 2
	if startX < 0 {
		startX = 0
	}
	startY := (bgH - fgH) / 2
	if startY < 0 {
		startY = 0
	}

	bgLines := strings.Split(bg, "\n")
	for len(bgLines) < bgH {
		bgLines = append(bgLines, strings.Repeat(" ", bgW))
	}

	for i, fgLine := range fgLines {
		y := startY + i
		if y >= len(bgLines) {
			break
		}
		bgLine := bgLines[y]
		// ANSI-aware: extract prefix (columns 0..startX) and suffix (columns startX+fgW..bgW).
		prefix := xansi.Cut(bgLine, 0, startX)
		prefixW := lipgloss.Width(prefix)
		if prefixW < startX {
			prefix += strings.Repeat(" ", startX-prefixW)
		}
		suffix := xansi.Cut(bgLine, startX+fgW, bgW)
		bgLines[y] = prefix + fgLine + suffix
	}

	return strings.Join(bgLines, "\n")
}
