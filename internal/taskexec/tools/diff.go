package tools

import (
	"strings"
)

type DiffPreviewer interface {
	PreviewDiff(workingDir string, params map[string]interface{}) (string, error)
}

type CommandContextProvider interface {
	GetCommandContext(workingDir string, params map[string]interface{}) map[string]any
}

func generateUnifiedDiff(oldContent, newContent string) string {
	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	hunks := computeHunks(oldLines, newLines, 3)

	if len(hunks) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, h := range hunks {
		sb.WriteString(formatHunk(h))
	}

	return sb.String()
}

type hunk struct {
	oldStart int
	oldCount int
	newStart int
	newCount int
	lines    []diffLine
}

type diffLine struct {
	kind    byte
	content string
}

func computeHunks(oldLines, newLines []string, context int) []hunk {
	edits := computeEditScript(oldLines, newLines)

	var hunks []hunk
	i := 0
	for i < len(edits) {
		if edits[i] == 0 {
			i++
			continue
		}

		start := i
		for start > 0 && edits[start-1] == 0 {
			start--
		}

		ctxBefore := i - start
		ctxBefore = min(ctxBefore, context)

		end := i + 1
		for end < len(edits) && edits[end] != 0 {
			end++
		}

		for end < len(edits) && edits[end] == 0 {
			end++
			if end >= len(edits) || edits[end] != 0 {
				break
			}
		}

		ctxAfter := 0
		ctxEnd := i
		for ctxEnd < len(edits) && ctxEnd < end {
			if edits[ctxEnd] == 0 {
				ctxAfter++
			}
			ctxEnd++
		}
		ctxAfter = min(ctxAfter, context)

		hunkStart := start - context + ctxBefore
		if hunkStart < 0 {
			hunkStart = 0
		}
		hunkEnd := i
		for hunkEnd < len(edits) && hunkEnd < end {
			hunkEnd++
		}
		hunkEnd = min(hunkEnd+context, len(edits))

		oldIdx := hunkStart
		newIdx := hunkStart
		oldStart := 0
		newStart := 0
		for j := 0; j < hunkStart; j++ {
			if edits[j] == 0 || edits[j] == -1 {
				oldStart++
			}
			if edits[j] == 0 || edits[j] == 1 {
				newStart++
			}
		}

		var lines []diffLine
		oldCount := 0
		newCount := 0
		oldIdx = oldStart
		newIdx = newStart

		for j := hunkStart; j < hunkEnd && j < len(edits); j++ {
			switch edits[j] {
			case 0:
				lines = append(lines, diffLine{kind: ' ', content: oldLines[oldIdx]})
				oldIdx++
				newIdx++
				oldCount++
				newCount++
			case -1:
				lines = append(lines, diffLine{kind: '-', content: oldLines[oldIdx]})
				oldIdx++
				oldCount++
			case 1:
				lines = append(lines, diffLine{kind: '+', content: newLines[newIdx]})
				newIdx++
				newCount++
			}
		}

		hunks = append(hunks, hunk{
			oldStart: oldStart + 1,
			oldCount: oldCount,
			newStart: newStart + 1,
			newCount: newCount,
			lines:    lines,
		})

		i = hunkEnd
	}

	return hunks
}

func computeEditScript(oldLines, newLines []string) []int {
	oldLen := len(oldLines)
	newLen := len(newLines)

	dp := make([][]int, oldLen+1)
	for i := range dp {
		dp[i] = make([]int, newLen+1)
	}

	for i := oldLen - 1; i >= 0; i-- {
		for j := newLen - 1; j >= 0; j-- {
			if oldLines[i] == newLines[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				if dp[i+1][j] > dp[i][j+1] {
					dp[i][j] = dp[i+1][j]
				} else {
					dp[i][j] = dp[i][j+1]
				}
			}
		}
	}

	edits := make([]int, 0, oldLen+newLen)
	i, j := 0, 0
	for i < oldLen || j < newLen {
		if i < oldLen && j < newLen && oldLines[i] == newLines[j] {
			edits = append(edits, 0)
			i++
			j++
		} else if j < newLen && (i >= oldLen || dp[i][j+1] >= dp[i+1][j]) {
			edits = append(edits, 1)
			j++
		} else {
			edits = append(edits, -1)
			i++
		}
	}

	return edits
}

func formatHunk(h hunk) string {
	var sb strings.Builder
	sb.WriteString("@@ -")
	sb.WriteString(formatRange(h.oldStart, h.oldCount))
	sb.WriteString(" +")
	sb.WriteString(formatRange(h.newStart, h.newCount))
	sb.WriteString(" @@\n")

	for _, l := range h.lines {
		sb.WriteByte(l.kind)
		sb.WriteString(l.content)
		sb.WriteByte('\n')
	}

	return sb.String()
}

func formatRange(start, count int) string {
	if count == 1 {
		return itoa(start)
	}
	return itoa(start) + "," + itoa(count)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
