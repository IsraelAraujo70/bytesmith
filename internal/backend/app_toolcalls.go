package backend

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bytesmith/internal/acp"
	"bytesmith/internal/session"
)

func normalizeMessageType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "thought", "reasoning":
		return "thought"
	default:
		return "text"
	}
}

func prettyJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return strings.TrimSpace(string(raw))
	}

	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return strings.TrimSpace(string(raw))
	}
	return string(formatted)
}

func formatToolCallContent(parts []session.ToolCallPart, update acp.SessionUpdate) string {
	sections := make([]string, 0, 6)

	for _, part := range parts {
		switch part.Type {
		case "content":
			if strings.TrimSpace(part.Text) != "" {
				sections = append(sections, "Content:\n"+part.Text)
			}
		case "diff":
			var b strings.Builder
			if strings.TrimSpace(part.Path) != "" {
				b.WriteString("Diff: " + part.Path + "\n")
			} else {
				b.WriteString("Diff:\n")
			}
			if part.OldText != "" || part.NewText != "" {
				b.WriteString("--- old\n")
				b.WriteString(part.OldText)
				if !strings.HasSuffix(part.OldText, "\n") {
					b.WriteString("\n")
				}
				b.WriteString("+++ new\n")
				b.WriteString(part.NewText)
			}
			diff := strings.TrimSpace(b.String())
			if diff != "" {
				sections = append(sections, diff)
			}
		case "terminal":
			terminalText := part.Text
			switch {
			case strings.TrimSpace(part.TerminalID) != "" && strings.TrimSpace(terminalText) != "":
				sections = append(sections, fmt.Sprintf("Terminal (%s):\n%s", part.TerminalID, terminalText))
			case strings.TrimSpace(part.TerminalID) != "":
				sections = append(sections, fmt.Sprintf("Terminal: %s", part.TerminalID))
			case strings.TrimSpace(terminalText) != "":
				sections = append(sections, "Terminal:\n"+terminalText)
			}
		default:
			if strings.TrimSpace(part.Text) != "" {
				sections = append(sections, part.Text)
			}
		}
	}

	if len(update.Locations) > 0 {
		lines := make([]string, 0, len(update.Locations))
		for _, loc := range update.Locations {
			if loc.Line > 0 {
				lines = append(lines, fmt.Sprintf("- %s:%d", loc.Path, loc.Line))
			} else {
				lines = append(lines, fmt.Sprintf("- %s", loc.Path))
			}
		}
		sections = append(sections, "Locations:\n"+strings.Join(lines, "\n"))
	}

	if input := prettyJSON(update.RawInput); input != "" {
		sections = append(sections, "Input:\n"+input)
	}
	if output := prettyJSON(update.RawOutput); output != "" {
		sections = append(sections, "Output:\n"+output)
	}

	return strings.TrimSpace(strings.Join(sections, "\n\n"))
}

func normalizeToolCallParts(parts []acp.ToolCallContent) []session.ToolCallPart {
	result := make([]session.ToolCallPart, 0, len(parts))
	for _, part := range parts {
		p := session.ToolCallPart{
			Type: strings.ToLower(strings.TrimSpace(part.Type)),
			Path: part.Path,
		}

		if p.Type == "" {
			p.Type = "content"
		}

		if part.Content != nil {
			p.Text = part.Content.Text
		}

		switch p.Type {
		case "diff":
			p.OldText = part.OldText
			p.NewText = part.NewText
		case "terminal":
			p.TerminalID = part.TerminalID
		}

		result = append(result, p)
	}
	return result
}

func summarizeDiffParts(parts []session.ToolCallPart) session.ToolCallDiffSummary {
	summary := session.ToolCallDiffSummary{}
	for _, part := range parts {
		if part.Type != "diff" {
			continue
		}
		summary.Files++
		additions, deletions := countLineDiff(part.OldText, part.NewText)
		summary.Additions += additions
		summary.Deletions += deletions
	}
	return summary
}

func countLineDiff(oldText, newText string) (additions, deletions int) {
	oldLines := splitLines(oldText)
	newLines := splitLines(newText)

	if len(oldLines) == 0 && len(newLines) == 0 {
		return 0, 0
	}

	dp := make([][]int, len(oldLines)+1)
	for i := range dp {
		dp[i] = make([]int, len(newLines)+1)
	}

	for i := len(oldLines) - 1; i >= 0; i-- {
		for j := len(newLines) - 1; j >= 0; j-- {
			if oldLines[i] == newLines[j] {
				dp[i][j] = 1 + dp[i+1][j+1]
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	lcs := dp[0][0]
	return len(newLines) - lcs, len(oldLines) - lcs
}

func splitLines(input string) []string {
	trimmed := strings.ReplaceAll(input, "\r\n", "\n")
	lines := strings.Split(trimmed, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 1 && lines[0] == "" {
		return []string{}
	}
	return lines
}

func toToolCallInfo(tc session.ToolCallRecord) ToolCallInfo {
	var parts []ToolCallPartInfo
	for _, part := range tc.Parts {
		parts = append(parts, ToolCallPartInfo{
			Type:       part.Type,
			Text:       part.Text,
			Path:       part.Path,
			OldText:    part.OldText,
			NewText:    part.NewText,
			TerminalID: part.TerminalID,
		})
	}

	var diffSummary *ToolCallDiffSummaryInfo
	if tc.DiffSummary.Additions > 0 || tc.DiffSummary.Deletions > 0 || tc.DiffSummary.Files > 0 {
		diffSummary = &ToolCallDiffSummaryInfo{
			Additions: tc.DiffSummary.Additions,
			Deletions: tc.DiffSummary.Deletions,
			Files:     tc.DiffSummary.Files,
		}
	}

	return ToolCallInfo{
		ID:          tc.ID,
		Title:       tc.Title,
		Kind:        tc.Kind,
		Status:      tc.Status,
		Content:     tc.Content,
		Parts:       parts,
		DiffSummary: diffSummary,
		Timestamp:   tc.Timestamp.Format(time.RFC3339),
	}
}
