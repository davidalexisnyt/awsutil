# Terminal Rendering Options for Go (Standard Library + ANSI)

## Overview

This document outlines options for rendering formatted text to the terminal in Go using only the standard library and ANSI escape codes.

## ANSI Escape Codes

ANSI escape codes are sequences of characters that control text formatting, colors, and cursor positioning in terminal emulators. They work on:
- **Windows 10+**: Native support (Windows Terminal, PowerShell, CMD)
- **Linux/Unix**: All modern terminals
- **macOS**: Terminal.app and iTerm2

### Basic ANSI Codes

```go
const (
    ansiReset     = "\033[0m"  // Reset all formatting
    ansiBold      = "\033[1m"   // Bold text
    ansiDim       = "\033[2m"   // Dim/faint text
    ansiItalic    = "\033[3m"   // Italic text
    ansiUnderline = "\033[4m"   // Underline

    // Foreground colors
    ansiFgBlack   = "\033[30m"
    ansiFgRed     = "\033[31m"
    ansiFgGreen   = "\033[32m"
    ansiFgYellow  = "\033[33m"
    ansiFgBlue    = "\033[34m"
    ansiFgMagenta = "\033[35m"
    ansiFgCyan    = "\033[36m"
    ansiFgWhite   = "\033[37m"

    // Background colors
    ansiBgBlack   = "\033[40m"
    ansiBgRed     = "\033[41m"
    ansiBgGreen   = "\033[42m"
    ansiBgYellow  = "\033[43m"
    ansiBgBlue    = "\033[44m"
    ansiBgMagenta = "\033[45m"
    ansiBgCyan    = "\033[46m"
    ansiBgWhite   = "\033[47m"
)
```

### Terminal Detection

Before using ANSI codes, check if output is going to a terminal:

```go
func isTerminal() bool {
    fileInfo, err := os.Stdout.Stat()
    if err != nil {
        return false
    }
    return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
```

This prevents ANSI codes from appearing in redirected output (e.g., `program > file.txt`).

## Source Format Options

### 1. Markdown (Recommended)

**Pros:**
- Widely recognized and readable
- Standard format
- Easy to maintain
- Can be viewed in GitHub, editors, etc.

**Cons:**
- Requires parsing (but basic parsing is simple)
- Some features don't translate well to terminal (tables, images)

**Supported Features:**
- Headers (`#`, `##`, `###`)
- Bold (`**text**`)
- Italic (`*text*`)
- Inline code (`` `code` ``)
- Code blocks (`` ``` ``)
- Lists (`-`, `*`)

### 2. Plain Text with Embedded ANSI

**Pros:**
- No parsing needed
- Full control over formatting

**Cons:**
- Not readable in source
- Hard to maintain
- Not portable (ANSI codes visible in editors)

**Example:**
```
\033[1m\033[36mHEADER\033[0m
\033[1mBold text\033[0m
```

### 3. Custom Lightweight Markup

**Pros:**
- Easy to parse
- Tailored to your needs

**Cons:**
- Non-standard
- Learning curve for contributors

**Example:**
```
[HEADER]Title[/HEADER]
[BOLD]Bold text[/BOLD]
[CODE]code[/CODE]
```

## Implementation

The `markdown_renderer.go` file provides a complete implementation that:

1. **Detects terminal capability** - Only uses ANSI if output is a TTY
2. **Parses basic Markdown** - Headers, bold, code blocks, lists
3. **Renders with ANSI** - Color-coded, formatted output
4. **Falls back gracefully** - Strips markdown for non-terminal output

### Usage

```go
// Embed markdown file
//go:embed help/general.md
var helpGeneral string

// Render it
func showHelp(command string) {
    RenderMarkdown(helpGeneral)
}
```

### Converting Existing Help Files

Your current `.txt` files can be converted to `.md` with minimal changes:

**Before (help/bastion.txt):**
```
awsutil bastion - Start a port forwarding session

USAGE:
    awsutil bastion [--profile <aws cli profile>]
```

**After (help/bastion.md):**
```markdown
# awsutil bastion

Start a port forwarding session through a bastion host.

## USAGE

    awsutil bastion [--profile <aws cli profile>]
```

## Advanced Features

### Extended Colors (256-color mode)

For more color options, use 256-color codes:

```go
ansiFg256 := func(color int) string {
    return fmt.Sprintf("\033[38;5;%dm", color)
}
```

### Windows Compatibility

On older Windows versions (pre-10), you may need to enable ANSI support:

```go
import "golang.org/x/sys/windows"

func enableANSIConsole() {
    // This requires golang.org/x/sys/windows (not standard library)
    // For standard library only, rely on Windows 10+ native support
}
```

However, since you're using standard library only, Windows 10+ native support is sufficient.

## Recommendations

1. **Use Markdown** - It's the most maintainable and standard format
2. **Implement basic parser** - The provided `RenderMarkdown` handles common cases
3. **Detect terminal** - Always check `isTerminal()` before using ANSI
4. **Graceful fallback** - Strip formatting for non-terminal output
5. **Keep it simple** - Focus on headers, bold, code blocks - these cover 90% of use cases

## Testing

Test your rendering with:

```bash
# Terminal output (with ANSI)
./awsutil help

# Redirected output (should strip ANSI)
./awsutil help > help.txt

# Pipe to less (should preserve ANSI)
./awsutil help | less -R
```

The `-R` flag in `less` preserves ANSI color codes.

