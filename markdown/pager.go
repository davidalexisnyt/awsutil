package markdown

// import (
// 	"bufio"
// 	"fmt"
// 	"os"
// 	"runtime"
// 	"strconv"
// 	"strings"
// 	"syscall"
// 	"unsafe"
// )

// // Terminal size structure for Windows
// type windowsCoord struct {
// 	X, Y int16
// }

// // ANSI escape codes for screen control
// const (
// 	ansiClearScreen    = "\033[2J\033[H"
// 	ansiHideCursor     = "\033[?25l"
// 	ansiShowCursor     = "\033[?25h"
// 	ansiSaveCursor     = "\033[s"
// 	ansiRestoreCursor  = "\033[u"
// 	ansiQueryCursorPos = "\033[6n"
// 	ansiQuerySize      = "\033[18t"
// )

// // getTerminalSize gets the terminal size using platform-specific methods
// func getTerminalSize() (rows, cols int, err error) {
// 	if runtime.GOOS == "windows" {
// 		// Windows: Use kernel32.dll to get console size
// 		kernel32 := syscall.NewLazyDLL("kernel32.dll")
// 		getConsoleScreenBufferInfo := kernel32.NewProc("GetConsoleScreenBufferInfo")
// 		var csbi struct {
// 			dwSize           windowsCoord
// 			dwCursorPosition windowsCoord
// 			wAttributes      uint16
// 			srWindow         struct {
// 				Left   int16
// 				Top    int16
// 				Right  int16
// 				Bottom int16
// 			}
// 			dwMaximumWindowSize windowsCoord
// 		}
// 		ret, _, _ := getConsoleScreenBufferInfo.Call(uintptr(syscall.Stdout), uintptr(unsafe.Pointer(&csbi)))
// 		if ret != 0 {
// 			rows = int(csbi.srWindow.Bottom - csbi.srWindow.Top + 1)
// 			cols = int(csbi.srWindow.Right - csbi.srWindow.Left + 1)
// 			return rows, cols, nil
// 		}
// 		// Fallback to default
// 		return 24, 80, nil
// 	}

// 	// Unix/Linux: Try environment variables first
// 	if rowsStr := os.Getenv("LINES"); rowsStr != "" {
// 		if r, err := strconv.Atoi(rowsStr); err == nil {
// 			rows = r
// 		}
// 	}
// 	if colsStr := os.Getenv("COLUMNS"); colsStr != "" {
// 		if c, err := strconv.Atoi(colsStr); err == nil {
// 			cols = c
// 		}
// 	}

// 	// Default fallback
// 	if rows == 0 {
// 		rows = 24
// 	}
// 	if cols == 0 {
// 		cols = 80
// 	}

// 	return rows, cols, nil
// }

// // clearScreen clears the terminal screen
// func clearScreen() {
// 	os.Stdout.WriteString(ansiClearScreen)
// }

// // RenderMarkdownPaged renders markdown with paging support
// func RenderMarkdownPaged(markdown string) {
// 	if !isTerminal() {
// 		// If not a terminal, just render normally
// 		RenderMarkdown(markdown)
// 		return
// 	}

// 	// Clear screen
// 	clearScreen()

// 	// Get terminal size
// 	rows, cols, err := getTerminalSize()
// 	if err != nil || rows < 3 {
// 		// Fallback: render without paging
// 		RenderMarkdown(markdown)
// 		return
// 	}

// 	// Reserve one line for navigation indicators
// 	usableRows := rows - 1

// 	// Render markdown to lines
// 	lines := renderMarkdownToLines(markdown, cols)

// 	// If content fits in one screen, just display it
// 	if len(lines) <= usableRows {
// 		for _, line := range lines {
// 			os.Stdout.WriteString(line + "\n")
// 		}
// 		os.Stdout.WriteString("\nPress any key to exit...")
// 		os.Stdout.Sync()
// 		readKey()
// 		clearScreen()
// 		return
// 	}

// 	// Paging mode
// 	currentLine := 0
// 	for {
// 		// Clear and display current page
// 		clearScreen()
// 		endLine := currentLine + usableRows
// 		if endLine > len(lines) {
// 			endLine = len(lines)
// 		}

// 		// Display lines
// 		for i := currentLine; i < endLine; i++ {
// 			os.Stdout.WriteString(lines[i] + "\n")
// 		}

// 		// Display navigation indicators
// 		displayNavigation(currentLine, len(lines), usableRows, cols)

// 		os.Stdout.Sync()

// 		// Read key
// 		key := readKey()
// 		switch key {
// 		case "pgdn", "down", "space":
// 			// Next page
// 			currentLine += usableRows
// 			if currentLine >= len(lines) {
// 				currentLine = len(lines) - usableRows
// 				if currentLine < 0 {
// 					currentLine = 0
// 				}
// 			}
// 		case "pgup", "up":
// 			// Previous page
// 			currentLine -= usableRows
// 			if currentLine < 0 {
// 				currentLine = 0
// 			}
// 		case "esc", "q":
// 			// Exit
// 			clearScreen()
// 			return
// 		case "home":
// 			// First page
// 			currentLine = 0
// 		case "end":
// 			// Last page
// 			currentLine = len(lines) - usableRows
// 			if currentLine < 0 {
// 				currentLine = 0
// 			}
// 		}
// 	}
// }

// // displayNavigation shows navigation indicators at the bottom
// func displayNavigation(currentLine, totalLines, pageSize, cols int) {
// 	// Calculate page info
// 	currentPage := (currentLine / pageSize) + 1
// 	totalPages := (totalLines + pageSize - 1) / pageSize
// 	if totalPages == 0 {
// 		totalPages = 1
// 	}

// 	// Build navigation line
// 	navText := fmt.Sprintf("Page %d/%d (PgDn: Next, PgUp: Prev, Esc: Exit)", currentPage, totalPages)

// 	// Center or align navigation
// 	if len(navText) < cols {
// 		// Center the text
// 		padding := (cols - len(navText)) / 2
// 		navText = strings.Repeat(" ", padding) + navText
// 	}

// 	// Display with reverse video or bold
// 	os.Stdout.WriteString(ansiBold + ansiFgCyan + navText + ansiReset)
// }

// // renderMarkdownToLines renders markdown and returns it as a slice of lines (with ANSI codes)
// func renderMarkdownToLines(markdown string, maxWidth int) []string {
// 	var lines []string

// 	scanner := bufio.NewScanner(strings.NewReader(markdown))
// 	inCodeBlock := false
// 	codeBlockLang := ""
// 	codeBlockLines := []string{}
// 	prevLineEmpty := false

// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		trimmed := strings.TrimSpace(line)

// 		// Handle code blocks
// 		if strings.HasPrefix(trimmed, "```") {
// 			if inCodeBlock {
// 				// End of code block - render the box
// 				boxLines := renderCodeBlockBoxToLines(codeBlockLang, codeBlockLines, maxWidth)
// 				lines = append(lines, boxLines...)
// 				inCodeBlock = false
// 				codeBlockLang = ""
// 				codeBlockLines = []string{}
// 				prevLineEmpty = false
// 				continue
// 			} else {
// 				// Start of code block
// 				inCodeBlock = true
// 				codeBlockLang = strings.TrimPrefix(trimmed, "```")
// 				codeBlockLang = strings.TrimSpace(codeBlockLang)
// 				prevLineEmpty = false
// 				continue
// 			}
// 		}

// 		if inCodeBlock {
// 			// Collect code block lines
// 			codeBlockLines = append(codeBlockLines, line)
// 			continue
// 		}

// 		// Empty lines
// 		if trimmed == "" {
// 			if !prevLineEmpty {
// 				lines = append(lines, "")
// 				prevLineEmpty = true
// 			}
// 			continue
// 		}
// 		prevLineEmpty = false

// 		// Headers
// 		if strings.HasPrefix(trimmed, "# ") {
// 			text := strings.TrimPrefix(trimmed, "# ")
// 			text = renderInlineMarkdown(text)
// 			lines = append(lines, ansiBold+ansiFgCyan+text+ansiReset)
// 			lines = append(lines, ansiBold+strings.Repeat("=", len(text))+ansiReset)
// 			continue
// 		}
// 		if strings.HasPrefix(trimmed, "## ") {
// 			text := strings.TrimPrefix(trimmed, "## ")
// 			text = renderInlineMarkdown(text)
// 			lines = append(lines, "")
// 			lines = append(lines, ansiBold+ansiFgCyan+text+ansiReset)
// 			lines = append(lines, ansiBold+strings.Repeat("-", len(text))+ansiReset)
// 			continue
// 		}
// 		if strings.HasPrefix(trimmed, "### ") {
// 			text := strings.TrimPrefix(trimmed, "### ")
// 			text = renderInlineMarkdown(text)
// 			lines = append(lines, "")
// 			lines = append(lines, ansiBold+ansiFgYellow+text+ansiReset)
// 			continue
// 		}
// 		if strings.HasPrefix(trimmed, "#### ") {
// 			text := strings.TrimPrefix(trimmed, "#### ")
// 			text = renderInlineMarkdown(text)
// 			lines = append(lines, ansiBold+ansiFgYellow+text+ansiReset)
// 			continue
// 		}

// 		// Lists
// 		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
// 			text := strings.TrimPrefix(trimmed, "- ")
// 			text = strings.TrimPrefix(text, "* ")
// 			text = renderInlineMarkdown(text)
// 			lines = append(lines, "  "+ansiFgGreen+"•"+ansiReset+" "+text)
// 			continue
// 		}
// 		if strings.HasPrefix(trimmed, "  - ") || strings.HasPrefix(trimmed, "  * ") {
// 			text := strings.TrimPrefix(trimmed, "  - ")
// 			text = strings.TrimPrefix(text, "  * ")
// 			text = renderInlineMarkdown(text)
// 			lines = append(lines, "    "+ansiFgGreen+"◦"+ansiReset+" "+text)
// 			continue
// 		}

// 		// Regular paragraph
// 		rendered := renderInlineMarkdown(line)
// 		lines = append(lines, rendered)
// 	}

// 	return lines
// }

// // renderCodeBlockBoxToLines renders a code block box and returns it as lines
// func renderCodeBlockBoxToLines(lang string, codeLines []string, maxWidth int) []string {
// 	if len(codeLines) == 0 {
// 		return []string{}
// 	}

// 	var boxLines []string

// 	// Calculate maximum content width (without borders)
// 	contentWidth := 0
// 	for _, line := range codeLines {
// 		if len(line) > contentWidth {
// 			contentWidth = len(line)
// 		}
// 	}

// 	// Ensure minimum content width
// 	if contentWidth < 20 {
// 		contentWidth = 20
// 	}

// 	// Ensure the content is wide enough to accommodate the language name starting at position 5
// 	if lang != "" {
// 		minContentWidth := 4 + len(lang)
// 		if contentWidth < minContentWidth {
// 			contentWidth = minContentWidth
// 		}
// 	}

// 	// Limit to terminal width
// 	if contentWidth > maxWidth-2 {
// 		contentWidth = maxWidth - 2
// 	}

// 	// Total box width = content width + 2 (for left and right borders)
// 	boxWidth := contentWidth + 2

// 	// Top border with language name starting at position 5
// 	topLine := ansiFgCyan + "┌"
// 	if lang != "" {
// 		topLine += "───"
// 		topLine += ansiBold + lang + ansiReset + ansiFgCyan
// 		remaining := boxWidth - 5 - len(lang)
// 		if remaining > 0 {
// 			topLine += strings.Repeat("─", remaining)
// 		}
// 	} else {
// 		topLine += strings.Repeat("─", boxWidth-2)
// 	}
// 	topLine += "┐" + ansiReset
// 	boxLines = append(boxLines, topLine)

// 	// Code block lines with vertical borders
// 	codeContentWidth := boxWidth - 2
// 	for _, line := range codeLines {
// 		boxLine := ansiFgCyan + "│" + ansiReset
// 		boxLine += ansiBgBlack + ansiFgWhite + line
// 		if len(line) < codeContentWidth {
// 			boxLine += strings.Repeat(" ", codeContentWidth-len(line))
// 		}
// 		boxLine += ansiReset + ansiFgCyan + "│" + ansiReset
// 		boxLines = append(boxLines, boxLine)
// 	}

// 	// Bottom border
// 	bottomLine := ansiFgCyan + "└" + strings.Repeat("─", boxWidth-2) + "┘" + ansiReset
// 	boxLines = append(boxLines, bottomLine)

// 	return boxLines
// }

// // readKey reads a single keypress and returns the key name
// func readKey() string {
// 	// Enable raw mode for reading single keypresses
// 	// This is platform-specific, so we'll use a simpler approach
// 	// that works on most terminals

// 	// For Windows, we need to use different approach
// 	if runtime.GOOS == "windows" {
// 		return readKeyWindows()
// 	}

// 	return readKeyUnix()
// }

// // readKeyWindows reads a keypress on Windows
// func readKeyWindows() string {
// 	var mode uint32
// 	stdin := syscall.Handle(os.Stdin.Fd())

// 	// Get current console mode
// 	kernel32 := syscall.NewLazyDLL("kernel32.dll")
// 	getConsoleMode := kernel32.NewProc("GetConsoleMode")
// 	setConsoleMode := kernel32.NewProc("SetConsoleMode")
// 	readConsoleInput := kernel32.NewProc("ReadConsoleInputW")

// 	getConsoleMode.Call(uintptr(stdin), uintptr(unsafe.Pointer(&mode)))

// 	// Enable raw mode (disable echo and line input)
// 	rawMode := mode &^ (0x0004 | 0x0002) // Disable ENABLE_ECHO_INPUT and ENABLE_LINE_INPUT
// 	setConsoleMode.Call(uintptr(stdin), uintptr(rawMode))
// 	defer setConsoleMode.Call(uintptr(stdin), uintptr(mode))

// 	// Try to read using ReadConsoleInput first (for special keys)
// 	var inputRecord struct {
// 		EventType uint16
// 		_         [2]byte // padding
// 		KeyEvent  struct {
// 			KeyDown         int32
// 			RepeatCount     uint16
// 			VirtualKeyCode  uint16
// 			VirtualScanCode uint16
// 			UnicodeChar     uint16
// 			ControlKeyState uint32
// 		}
// 	}
// 	var numRead uint32

// 	ret, _, _ := readConsoleInput.Call(
// 		uintptr(stdin),
// 		uintptr(unsafe.Pointer(&inputRecord)),
// 		1,
// 		uintptr(unsafe.Pointer(&numRead)),
// 	)

// 	if ret != 0 && numRead > 0 && inputRecord.EventType == 1 { // KEY_EVENT
// 		if inputRecord.KeyEvent.KeyDown != 0 {
// 			vk := inputRecord.KeyEvent.VirtualKeyCode
// 			// VK_PRIOR = 0x21, VK_NEXT = 0x22, VK_ESCAPE = 0x1B, VK_SPACE = 0x20
// 			switch vk {
// 			case 0x21: // VK_PRIOR (Page Up)
// 				return "pgup"
// 			case 0x22: // VK_NEXT (Page Down)
// 				return "pgdn"
// 			case 0x1B: // VK_ESCAPE
// 				return "esc"
// 			case 0x20: // VK_SPACE
// 				return "space"
// 			case 0x25: // VK_LEFT (not used, but handle gracefully)
// 				return "left"
// 			case 0x26: // VK_UP
// 				return "up"
// 			case 0x27: // VK_RIGHT (not used, but handle gracefully)
// 				return "right"
// 			case 0x28: // VK_DOWN
// 				return "down"
// 			case 0x24: // VK_HOME
// 				return "home"
// 			case 0x23: // VK_END
// 				return "end"
// 			default:
// 				// Check for 'q' or 'Q'
// 				ch := inputRecord.KeyEvent.UnicodeChar
// 				if ch == 'q' || ch == 'Q' {
// 					return "q"
// 				}
// 				if ch >= 32 && ch < 127 {
// 					return string(rune(ch))
// 				}
// 			}
// 		}
// 	}

// 	// Fallback: try reading as ANSI escape sequence (for modern terminals)
// 	reader := bufio.NewReader(os.Stdin)
// 	ch, err := reader.ReadByte()
// 	if err != nil {
// 		return ""
// 	}

// 	// Check for escape sequence
// 	if ch == 0x1B { // ESC
// 		// Try to read more bytes for escape sequence
// 		ch2, err := reader.ReadByte()
// 		if err == nil {
// 			if ch2 == '[' {
// 				// ANSI escape sequence
// 				seq := []byte{ch, ch2}
// 				for i := 0; i < 10; i++ {
// 					b, err := reader.ReadByte()
// 					if err != nil {
// 						break
// 					}
// 					seq = append(seq, b)
// 					if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '~' {
// 						break
// 					}
// 				}
// 				parsed := parseEscapeSequence(string(seq))
// 				if parsed != "unknown" {
// 					return parsed
// 				}
// 			}
// 		}
// 		return "esc"
// 	}

// 	// Check for space
// 	if ch == ' ' {
// 		return "space"
// 	}

// 	// Check for 'q'
// 	if ch == 'q' || ch == 'Q' {
// 		return "q"
// 	}

// 	return string(ch)
// }

// // readKeyUnix reads a keypress on Unix-like systems
// func readKeyUnix() string {
// 	// Use a simpler approach with bufio
// 	// Note: This won't work perfectly without raw mode, but it's a reasonable fallback
// 	reader := bufio.NewReader(os.Stdin)

// 	// Try to read escape sequence
// 	ch, err := reader.ReadByte()
// 	if err != nil {
// 		return ""
// 	}

// 	if ch == 0x1B { // ESC
// 		// Try to read more for escape sequence
// 		ch2, err := reader.ReadByte()
// 		if err == nil && ch2 == '[' {
// 			seq := []byte{ch, ch2}
// 			for i := 0; i < 10; i++ {
// 				b, err := reader.ReadByte()
// 				if err != nil {
// 					break
// 				}
// 				seq = append(seq, b)
// 				if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') {
// 					break
// 				}
// 			}
// 			return parseEscapeSequence(string(seq))
// 		}
// 		return "esc"
// 	}

// 	if ch == ' ' {
// 		return "space"
// 	}

// 	if ch == 'q' || ch == 'Q' {
// 		return "q"
// 	}

// 	return string(ch)
// }

// // parseEscapeSequence parses ANSI escape sequences and returns key name
// func parseEscapeSequence(seq string) string {
// 	if strings.HasPrefix(seq, "\033[") {
// 		suffix := strings.TrimPrefix(seq, "\033[")

// 		// Page Down: [6~ or [6;~ (with modifiers)
// 		if strings.Contains(suffix, "6") && strings.Contains(suffix, "~") {
// 			return "pgdn"
// 		}
// 		// Page Up: [5~ or [5;~ (with modifiers)
// 		if strings.Contains(suffix, "5") && strings.Contains(suffix, "~") {
// 			return "pgup"
// 		}

// 		// Arrow keys
// 		if strings.HasSuffix(suffix, "A") && !strings.Contains(suffix, "~") {
// 			return "up"
// 		}
// 		if strings.HasSuffix(suffix, "B") && !strings.Contains(suffix, "~") {
// 			return "down"
// 		}
// 		if strings.HasSuffix(suffix, "H") {
// 			return "home"
// 		}
// 		if strings.HasSuffix(suffix, "F") {
// 			return "end"
// 		}
// 	}

// 	return "unknown"
// }
