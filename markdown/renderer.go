package markdown

import (
	"bufio"
	"os"
	"strings"
)

// ANSI escape codes
const (
	ansiReset     = "\033[0m"
	ansiBold      = "\033[1m"
	ansiDim       = "\033[2m"
	ansiItalic    = "\033[3m"
	ansiUnderline = "\033[4m"
	ansiFgBlack   = "\033[30m"
	ansiFgRed     = "\033[31m"
	ansiFgGreen   = "\033[32m"
	ansiFgYellow  = "\033[33m"
	ansiFgBlue    = "\033[34m"
	ansiFgMagenta = "\033[35m"
	ansiFgCyan    = "\033[36m"
	ansiFgWhite   = "\033[37m"
	ansiBgBlack   = "\033[40m"
	ansiBgRed     = "\033[41m"
	ansiBgGreen   = "\033[42m"
	ansiBgYellow  = "\033[43m"
	ansiBgBlue    = "\033[44m"
	ansiBgMagenta = "\033[45m"
	ansiBgCyan    = "\033[46m"
	ansiBgWhite   = "\033[47m"
)

// isTerminal checks if stdout is a terminal (TTY)
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// RenderMarkdown renders basic Markdown to ANSI-formatted terminal output
// Supports: headers (# ## ###), bold (**text**), code blocks (```), inline code (`code`), and lists
func RenderMarkdown(markdown string) {
	if !isTerminal() {
		// If not a terminal, just print plain text (strip markdown)
		renderPlainText(markdown)
		return
	}

	os.Stdout.WriteString("\n")

	scanner := bufio.NewScanner(strings.NewReader(markdown))
	inCodeBlock := false
	codeBlockLang := ""
	codeBlockLines := []string{}
	prevLineEmpty := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Handle code blocks
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				// End of code block - render the box
				renderCodeBlockBox(codeBlockLang, codeBlockLines)
				inCodeBlock = false
				codeBlockLang = ""
				codeBlockLines = []string{}
				prevLineEmpty = false
				continue
			} else {
				// Start of code block
				inCodeBlock = true
				codeBlockLang = strings.TrimPrefix(trimmed, "```")
				codeBlockLang = strings.TrimSpace(codeBlockLang)
				prevLineEmpty = false
				continue
			}
		}

		if inCodeBlock {
			// Collect code block lines
			codeBlockLines = append(codeBlockLines, line)
			continue
		}

		// Empty lines
		if trimmed == "" {
			if !prevLineEmpty {
				os.Stdout.WriteString("\n")
				prevLineEmpty = true
			}
			continue
		}
		prevLineEmpty = false

		// Headers
		if strings.HasPrefix(trimmed, "# ") {
			// H1
			text := strings.TrimPrefix(trimmed, "# ")
			text = renderInlineMarkdown(text)
			os.Stdout.WriteString(ansiBold + ansiFgCyan + text + ansiReset + "\n")
			os.Stdout.WriteString(ansiBold + strings.Repeat("=", len(text)) + ansiReset + "\n")
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			// H2
			text := strings.TrimPrefix(trimmed, "## ")
			text = renderInlineMarkdown(text)
			os.Stdout.WriteString("\n" + ansiBold + ansiFgCyan + text + ansiReset + "\n")
			os.Stdout.WriteString(ansiBold + strings.Repeat("-", len(text)) + ansiReset + "\n")
			continue
		}
		if strings.HasPrefix(trimmed, "### ") {
			// H3
			text := strings.TrimPrefix(trimmed, "### ")
			text = renderInlineMarkdown(text)
			os.Stdout.WriteString("\n" + ansiBold + ansiFgYellow + text + ansiReset + "\n")
			continue
		}
		if strings.HasPrefix(trimmed, "#### ") {
			// H4
			text := strings.TrimPrefix(trimmed, "#### ")
			text = renderInlineMarkdown(text)
			os.Stdout.WriteString(ansiBold + ansiFgYellow + text + ansiReset + "\n")
			continue
		}

		// Lists
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			text := strings.TrimPrefix(trimmed, "- ")
			text = strings.TrimPrefix(text, "* ")
			text = renderInlineMarkdown(text)
			os.Stdout.WriteString("  " + ansiFgGreen + "•" + ansiReset + " " + text + "\n")
			continue
		}
		if strings.HasPrefix(trimmed, "  - ") || strings.HasPrefix(trimmed, "  * ") {
			text := strings.TrimPrefix(trimmed, "  - ")
			text = strings.TrimPrefix(text, "  * ")
			text = renderInlineMarkdown(text)
			os.Stdout.WriteString("    " + ansiFgGreen + "◦" + ansiReset + " " + text + "\n")
			continue
		}

		// Regular paragraph
		rendered := renderInlineMarkdown(line)
		os.Stdout.WriteString(rendered + "\n")
	}
}

// renderInlineMarkdown processes inline markdown formatting (bold, code, etc.)
func renderInlineMarkdown(text string) string {
	var result strings.Builder
	i := 0

	for i < len(text) {
		// Inline code `code`
		if i < len(text)-1 && text[i] == '`' {
			// Find closing backtick
			end := strings.IndexByte(text[i+1:], '`')
			if end != -1 {
				end += i + 1
				code := text[i+1 : end]
				result.WriteString(ansiBgBlack + ansiFgCyan + code + ansiReset)
				i = end + 1
				continue
			}
		}

		// Bold **text** or __text__
		if i < len(text)-1 && text[i] == '*' && text[i+1] == '*' {
			// Find closing **
			end := strings.Index(text[i+2:], "**")
			if end != -1 {
				end += i + 2
				boldText := text[i+2 : end]
				result.WriteString(ansiBold + boldText + ansiReset)
				i = end + 2
				continue
			}
		}

		// Italic *text* or _text_ (but not **text**)
		if i < len(text)-1 && text[i] == '*' && text[i+1] != '*' {
			// Find closing *
			end := strings.IndexByte(text[i+1:], '*')
			if end != -1 {
				end += i + 1
				italicText := text[i+1 : end]
				result.WriteString(ansiItalic + italicText + ansiReset)
				i = end + 1
				continue
			}
		}

		// Regular character
		result.WriteByte(text[i])
		i++
	}

	return result.String()
}

// renderCodeBlockBox renders a code block enclosed in a box using box-drawing characters
func renderCodeBlockBox(lang string, lines []string) {
	if len(lines) == 0 {
		return
	}

	// Calculate maximum content width (without borders)
	contentWidth := 0
	for _, line := range lines {
		if len(line) > contentWidth {
			contentWidth = len(line)
		}
	}

	// Ensure minimum content width
	if contentWidth < 20 {
		contentWidth = 20
	}

	// Ensure the content is wide enough to accommodate the language name starting at position 5
	// The language name starts at position 5 in the top border: ┌───[lang]───┐
	// So we need: 3 (───) + len(lang) + at least 1 more dash = 4 + len(lang) minimum
	if lang != "" {
		minContentWidth := 4 + len(lang)
		if contentWidth < minContentWidth {
			contentWidth = minContentWidth
		}
	}

	// Total box width = content width + 2 (for left and right borders)
	maxWidth := contentWidth + 2

	// Top border with language name starting at position 5
	// Format: ┌───[lang]───┐
	// Positions: 1=┌, 2-4=───, 5+=lang, then ─, last=┐
	os.Stdout.WriteString(ansiFgCyan + "┌")
	if lang != "" {
		// Fill up to position 5 (we already have ┌, so we need 3 more ─)
		os.Stdout.WriteString("───")
		os.Stdout.WriteString(ansiBold + lang + ansiReset + ansiFgCyan)
		// Fill remaining width: maxWidth total - 1 (┌) - 3 (───) - len(lang) - 1 (┐)
		remaining := maxWidth - 5 - len(lang)
		if remaining > 0 {
			os.Stdout.WriteString(strings.Repeat("─", remaining))
		}
	} else {
		// No language, just fill with dashes (maxWidth - 2 for borders)
		os.Stdout.WriteString(strings.Repeat("─", maxWidth-2))
	}
	os.Stdout.WriteString("┐" + ansiReset + "\n")

	// Code block lines with vertical borders
	// Content width is maxWidth - 2 (for left and right borders)
	codeContentWidth := maxWidth - 2
	for _, line := range lines {
		os.Stdout.WriteString(ansiFgCyan + "│" + ansiReset)
		os.Stdout.WriteString(ansiBgBlack + ansiFgWhite + line)
		// Pad line to codeContentWidth
		if len(line) < codeContentWidth {
			os.Stdout.WriteString(strings.Repeat(" ", codeContentWidth-len(line)))
		}
		os.Stdout.WriteString(ansiReset + ansiFgCyan + "│" + ansiReset + "\n")
	}

	// Bottom border
	os.Stdout.WriteString(ansiFgCyan + "└" + strings.Repeat("─", maxWidth-2) + "┘" + ansiReset + "\n")
}

// renderPlainText strips markdown and renders as plain text (for non-terminal output)
func renderPlainText(markdown string) {
	scanner := bufio.NewScanner(strings.NewReader(markdown))
	inCodeBlock := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Handle code blocks
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			os.Stdout.WriteString(line + "\n")
			continue
		}

		// Strip markdown formatting
		line = strings.TrimPrefix(line, "# ")
		line = strings.TrimPrefix(line, "## ")
		line = strings.TrimPrefix(line, "### ")
		line = strings.TrimPrefix(line, "#### ")
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")

		// Remove inline formatting
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "__", "")
		line = strings.ReplaceAll(line, "*", "")
		line = strings.ReplaceAll(line, "_", "")
		line = strings.ReplaceAll(line, "`", "")

		os.Stdout.WriteString(line + "\n")
	}
}
