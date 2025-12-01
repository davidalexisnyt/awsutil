package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	greenColor  = "\033[32m"
	resetColor  = "\033[0m"
	clearScreen = "\033[2J\033[H" // Clear screen and move cursor to home
	prompt      = "awsdo>> "
	ctrlL       = '\f' // Form feed character (Ctrl-L)
	ctrlD       = 0x04 // Ctrl-D character
)

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// startREPL starts the interactive REPL mode
func startREPL(configFile string, config *Configuration) {
	// Print intro text
	fmt.Println("\nWelcome to the awsdo REPL!")
	fmt.Println("Type 'help' for available commands, or 'exit'/'quit' to exit.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	for {
		// Print green prompt
		fmt.Print(greenColor + prompt + resetColor)

		// Read input character by character to detect Ctrl-L
		var line strings.Builder

		for {
			char, err := reader.ReadByte()
			if err != nil {
				// Handle EOF
				if err == io.EOF {
					fmt.Println()
					fmt.Println("Goodbye!")
					fmt.Println()
					return
				}

				// Handle other errors
				fmt.Printf("Error reading input: %v\n", err)
				fmt.Println()
				return
			}

			// Check for Ctrl-D
			if char == ctrlD {
				fmt.Println()
				fmt.Println("Goodbye!")
				fmt.Println()
				return
			}

			// Check for Ctrl-L (form feed)
			if char == ctrlL {
				// Clear screen and discard any partial input
				fmt.Print(clearScreen)
				line.Reset() // Clear any characters typed before Ctrl-L
				break        // Break inner loop to show prompt again
			}

			// Check for newline (end of line)
			if char == '\n' {
				break
			}

			// Check for carriage return (Windows line ending)
			if char == '\r' {
				// Peek at next character to see if it's newline
				nextChar, err := reader.Peek(1)

				if err == nil && len(nextChar) > 0 && nextChar[0] == '\n' {
					reader.ReadByte() // Consume the newline
				}

				break
			}

			// Append character to line
			line.WriteByte(char)
		}

		// If line is empty (Ctrl-L was pressed), continue to next iteration
		if line.Len() == 0 {
			continue
		}

		inputLine := strings.TrimSpace(line.String())

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

		// Handle exit commands
		if command == "quit" || command == "q" || command == ":q" || command == ".q" || command == "exit" || command == ":exit" || command == ".exit" {
			fmt.Println()
			fmt.Println("Goodbye!")
			fmt.Println()
			return
		}

		// Execute command
		err := executeREPLCommand(command, args[1:], config)
		if err != nil {
			fmt.Println(err.Error())
			fmt.Println()
		} else {
			// Save configuration after successful command
			saveConfiguration(configFile, config)
		}

		// fmt.Println()
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
	default:
		fmt.Printf("Invalid command: %s\n", command)
		fmt.Println("Use 'help' to see available commands.")
		return nil
	}
}
