package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	greenColor  = "\033[32m"
	resetColor  = "\033[0m"
	clearScreen = "\033[2J\033[H" // Clear screen and move cursor to home
	prompt      = "awsdo>> "
	ctrlL       = '\f' // Form feed character (Ctrl-L)
	ctrlD       = 0x04 // Ctrl-D character
	backspace   = '\b' // Backspace character
	del         = 0x7F // DEL character (also used for backspace on some systems)
	esc         = 0x1B // Escape character
)

// lineEditor handles line editing with cursor movement and history
type lineEditor struct {
	line      []rune   // Current line as runes
	cursorPos int      // Cursor position in runes
	history   []string // Command history
	histIndex int      // Current history index (-1 = not browsing history)
}

// newLineEditor creates a new line editor
func newLineEditor() *lineEditor {
	return &lineEditor{
		line:      make([]rune, 0),
		cursorPos: 0,
		history:   make([]string, 0),
		histIndex: -1,
	}
}

// addToHistory adds a command to history (if not empty and not duplicate of last)
func (le *lineEditor) addToHistory(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}

	// Don't add if it's the same as the last command
	if len(le.history) > 0 && le.history[len(le.history)-1] == cmd {
		return
	}

	le.history = append(le.history, cmd)

	// Keep history limited to last 100 commands
	if len(le.history) > 100 {
		le.history = le.history[len(le.history)-100:]
	}
}

// redrawLine redraws the current line with cursor at correct position
func (le *lineEditor) redrawLine() {
	// Move cursor to beginning of line
	fmt.Print("\033[1G")

	// Clear to end of line
	fmt.Print("\033[K")

	// Print prompt
	fmt.Print(greenColor + prompt + resetColor)

	// Print line content
	fmt.Print(string(le.line))

	// Move cursor back to correct position
	if le.cursorPos < len(le.line) {
		// Calculate how many characters to move back
		charsToMove := len(le.line) - le.cursorPos
		fmt.Printf("\033[%dD", charsToMove)
	}
}

// insertRune inserts a rune at the current cursor position
func (le *lineEditor) insertRune(r rune) {
	if le.cursorPos == len(le.line) {
		// Append at end
		le.line = append(le.line, r)
		le.cursorPos++
		fmt.Print(string(r))
	} else {
		// Insert in middle
		le.line = append(le.line[:le.cursorPos], append([]rune{r}, le.line[le.cursorPos:]...)...)
		le.cursorPos++
		le.redrawLine()
	}
}

// deleteChar deletes the character before the cursor (backspace)
func (le *lineEditor) deleteChar() bool {
	if le.cursorPos == 0 {
		return false
	}

	le.line = append(le.line[:le.cursorPos-1], le.line[le.cursorPos:]...)
	le.cursorPos--
	le.redrawLine()
	return true
}

// deleteCharForward deletes the character at the cursor (Delete key)
func (le *lineEditor) deleteCharForward() bool {
	if le.cursorPos >= len(le.line) {
		return false
	}

	le.line = append(le.line[:le.cursorPos], le.line[le.cursorPos+1:]...)
	le.redrawLine()
	return true
}

// moveCursorLeft moves cursor left
func (le *lineEditor) moveCursorLeft() {
	if le.cursorPos > 0 {
		le.cursorPos--
		fmt.Print("\033[D") // Move cursor left
	}
}

// moveCursorRight moves cursor right
func (le *lineEditor) moveCursorRight() {
	if le.cursorPos < len(le.line) {
		le.cursorPos++
		fmt.Print("\033[C") // Move cursor right
	}
}

// setLine sets the current line content
func (le *lineEditor) setLine(s string) {
	le.line = []rune(s)
	le.cursorPos = len(le.line)
	le.redrawLine()
}

// getLine returns the current line as a string
func (le *lineEditor) getLine() string {
	return string(le.line)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// readLineWithEditing reads a line from stdin with proper handling of backspace, arrow keys, and history
// Note: terminal should already be in raw mode when this is called
func readLineWithEditing(reader *bufio.Reader, editor *lineEditor) (string, error) {
	// Reset editor state for new line
	editor.setLine("")
	editor.histIndex = -1

	for {
		// Try to read a rune first (handles UTF-8 properly)
		r, size, err := reader.ReadRune()
		if err != nil {
			return "", err
		}

		// Handle single-byte control characters
		if size == 1 {
			char := byte(r)

			// Check for Ctrl-D
			if char == ctrlD {
				return "", io.EOF
			}

			// Check for Ctrl-L (form feed)
			if char == ctrlL {
				// Clear screen and discard any partial input
				fmt.Print(clearScreen)
				return "", nil // Signal to caller to continue
			}

			// Check for escape sequence (arrow keys, etc.)
			if char == esc {
				// Read the bracket
				nextChar, err := reader.ReadByte()
				if err != nil {
					continue
				}

				if nextChar == '[' {
					// Read the direction/action character
					action, err := reader.ReadByte()
					if err != nil {
						continue
					}

					switch action {
					case 'A': // Up arrow - history previous
						if len(editor.history) > 0 {
							if editor.histIndex == -1 {
								// Start browsing from end
								editor.histIndex = len(editor.history) - 1
							} else if editor.histIndex > 0 {
								editor.histIndex--
							}
							editor.setLine(editor.history[editor.histIndex])
						}
					case 'B': // Down arrow - history next
						if editor.histIndex >= 0 {
							if editor.histIndex < len(editor.history)-1 {
								editor.histIndex++
								editor.setLine(editor.history[editor.histIndex])
							} else {
								// Go back to current (empty) line
								editor.histIndex = -1
								editor.setLine("")
							}
						}
					case 'C': // Right arrow
						editor.moveCursorRight()
					case 'D': // Left arrow
						editor.moveCursorLeft()
					case '3': // Delete key (ESC[3~)
						nextChar, err := reader.ReadByte()
						if err == nil && nextChar == '~' {
							editor.deleteCharForward()
						}
					}
				}
				continue
			}

			// Handle backspace
			if char == backspace || char == del {
				editor.deleteChar()
				continue
			}

			// Check for newline or carriage return (end of line)
			if char == '\r' {
				// Print newline and return
				fmt.Print("\033[K")
				line := editor.getLine()
				editor.addToHistory(line)

				return line, nil
			}

			if char == '\n' {
				// Print newline and return
				fmt.Print("\n")
				line := editor.getLine()
				editor.addToHistory(line)
				return line, nil
			}
		}

		// Handle printable characters (including multi-byte UTF-8)
		if r >= 32 || r == '\t' {
			editor.insertRune(r)
		}
		// Ignore other control characters
	}
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// startREPL starts the interactive REPL mode
func startREPL(configFile string, config *Configuration) {
	// Print intro text
	fmt.Println("\nWelcome to the awsdo REPL!")
	fmt.Println("Type 'help' for available commands, or 'exit'/'quit' to exit.")
	fmt.Println()

	// Check if stdin is a terminal
	isTerminal := term.IsTerminal(int(os.Stdin.Fd()))
	fd := int(os.Stdin.Fd())

	var originalState *term.State // Original cooked mode state
	var err error

	// Put terminal in raw mode for the entire session if it's a terminal
	if isTerminal {
		originalState, err = term.MakeRaw(fd)
		if err != nil {
			// If we can't put terminal in raw mode, fall back to simple mode
			isTerminal = false
		} else {
			// Save the raw mode state so we can restore it after commands
			// The terminal is now in raw mode, so MakeRaw would return the raw state
			// But we need to get the raw state differently - actually, we can
			// just call MakeRaw again after restoring, or we can save it now
			// Actually, let's use a different approach - save original, and
			// we'll restore to original then make raw again after each command
			// Restore terminal state when we exit
			defer term.Restore(fd, originalState)
		}
	}

	reader := bufio.NewReader(os.Stdin)
	editor := newLineEditor()

	for {
		// Print green prompt (line editor will handle redrawing)
		fmt.Print(greenColor + prompt + resetColor)

		var inputLine string

		if isTerminal {
			// Use line editing for terminals (already in raw mode)
			inputLine, err = readLineWithEditing(reader, editor)
		} else {
			// Fall back to simple ReadString for non-terminals (pipes, etc.)
			inputLine, err = reader.ReadString('\n')
			if err == nil {
				inputLine = strings.TrimRight(inputLine, "\r\n")
			}
		}

		if err != nil {
			// Handle EOF
			if err == io.EOF {
				if isTerminal && originalState != nil {
					term.Restore(fd, originalState)
				}

				fmt.Println("\033[K")
				fmt.Println("\033[KGoodbye!")
				fmt.Println()
				return
			}

			// Handle other errors
			fmt.Printf("\033[KError reading input: %v\n", err)
			fmt.Println()
			return
		}

		// Handle Ctrl-L (returns empty string)
		if inputLine == "" {
			continue
		}

		inputLine = strings.TrimSpace(inputLine)

		// Handle empty input
		if inputLine == "" {
			continue
		}

		// Parse command and arguments
		args := strings.Fields(inputLine)
		if len(args) == 0 {
			continue
		}

		command := strings.ToLower(args[0])

		// Restore terminal to normal (cooked) mode for command output
		// This allows proper line wrapping and carriage return handling
		if isTerminal && originalState != nil {
			term.Restore(fd, originalState)
		}

		// Handle exit commands
		if command == "quit" || command == "q" || command == ":q" || command == ".q" || command == "exit" || command == ":exit" || command == ".exit" {
			fmt.Println("\033[K")
			fmt.Println("Goodbye!!")
			fmt.Println()
			return
		}

		// Execute command
		err = executeREPLCommand(command, args[1:], config)
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println()
		} else {
			// Save configuration after successful command
			saveConfiguration(configFile, config)
		}

		// Put terminal back in raw mode for next input
		if isTerminal && originalState != nil {
			// MakeRaw will put terminal in raw mode and return the cooked state
			// (which should be the same as originalState)
			_, _ = term.MakeRaw(fd)
		}
	}
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// executeREPLCommand routes commands to the appropriate handlers (similar to main.go)
func executeREPLCommand(command string, args []string, config *Configuration) error {
	switch command {
	case "help", ":help", ".h":
		if len(args) > 0 {
			showHelp(strings.ToLower(args[0]))
		} else {
			showHelp("")
		}

		fmt.Println()
		return nil
	case "login":
		return login(args, config)
	case "instances":
		if len(args) < 1 {
			return listInstances(args, config)
		}

		subcommand := strings.ToLower(args[0])

		switch subcommand {
		case "find":
			return findInstances(args[1:], config)
		case "list", "ls":
			return listInstances(args[1:], config)
		case "add":
			return addInstance(args[1:], config)
		case "update":
			return updateInstance(args[1:], config)
		case "remove", "rm":
			return removeInstance(args[1:], config)
		default:
			fmt.Printf("Invalid instances subcommand: %s\n", subcommand)
			fmt.Println("Use 'instances find' to find instances, 'instances list' to list configured instances, 'instances add' to add an instance, 'instances update' to update an instance, 'instances remove' to remove an instance, or 'help instances' for more information.")
			return nil
		}
	case "terminal":
		return startSSMSession(args, config)
	case "bastion":
		return startBastionTunnel(args, config)
	case "bastions":
		if len(args) < 1 {
			// Default to 'list' if no subcommand provided
			return listBastions(args, config)
		}

		subcommand := strings.ToLower(args[0])

		switch subcommand {
		case "list", "ls":
			return listBastions(args[1:], config)
		case "add":
			return addBastion(args[1:], config)
		case "update", "up":
			return updateBastion(args[1:], config)
		case "remove", "rm":
			return removeBastion(args[1:], config)
		default:
			fmt.Printf("Invalid bastions subcommand: %s\n", subcommand)
			fmt.Println("Use 'bastions list' to list bastions, 'bastions add' to add a new bastion, 'bastions update' to update an existing bastion, or 'bastions remove' to remove a bastion.")
			return nil
		}
	case "docs":
		showDocs()
		return nil
	case "clear", "cls", "clr", ".c":
		fmt.Print(clearScreen)
		return nil
	case "ls", "list":
		if len(args) < 1 {
			fmt.Println("Usage: ls <instances|bastions> [options]")
			fmt.Println("   or: list <instances|bastions> [options]")
			return nil
		}
		object := strings.ToLower(args[0])
		switch object {
		case "instances", "instance":
			return listInstances(args[1:], config)
		case "bastions", "bastion":
			return listBastions(args[1:], config)
		default:
			fmt.Printf("Invalid object: %s\n", object)
			fmt.Println("Use 'ls instances' or 'ls bastions'")
			return nil
		}
	case "add":
		if len(args) < 1 {
			fmt.Println("Usage: add <instance|bastion> [options]")
			return nil
		}
		object := strings.ToLower(args[0])
		switch object {
		case "instance", "instances":
			return addInstance(args[1:], config)
		case "bastion", "bastions":
			return addBastion(args[1:], config)
		default:
			fmt.Printf("Invalid object: %s\n", object)
			fmt.Println("Use 'add instance' or 'add bastion'")
			return nil
		}
	case "rm":
		if len(args) < 1 {
			fmt.Println("Usage: rm <instance|bastion> [options]")
			return nil
		}
		object := strings.ToLower(args[0])
		switch object {
		case "instance", "instances":
			return removeInstance(args[1:], config)
		case "bastion", "bastions":
			return removeBastion(args[1:], config)
		default:
			fmt.Printf("Invalid object: %s\n", object)
			fmt.Println("Use 'rm instance' or 'rm bastion'")
			return nil
		}
	case "find":
		if len(args) < 1 {
			fmt.Println("Usage: find <instance> [options]")
			return nil
		}
		object := strings.ToLower(args[0])
		switch object {
		case "instance", "instances":
			return findInstances(args[1:], config)
		default:
			fmt.Printf("Invalid object: %s\n", object)
			fmt.Println("Use 'find instance'")
			return nil
		}
	default:
		fmt.Printf("Invalid command: %s\n", command)
		fmt.Println("Use 'help' to see available commands.")
		return nil
	}
}
