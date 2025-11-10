package main

import (
	"awsutil/markdown"
	"fmt"
	_ "embed"
)

//go:embed help/general.txt
var helpGeneral string

//go:embed help/login.txt
var helpLogin string

//go:embed help/instances.txt
var helpInstances string

//go:embed help/terminal.txt
var helpTerminal string

//go:embed help/bastion.txt
var helpBastion string

//go:embed help/bastions.txt
var helpBastions string

//go:embed help/configure.txt
var helpConfigure string

//go:embed help/help.txt
var helpHelp string

//go:embed help/docs.txt
var helpDocs string

//go:embed help/unknown.txt
var helpUnknown string

//go:embed README.md
var readmeContent string

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func showDocs() {
	markdown.RenderMarkdown(readmeContent)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func showHelp(command string) {
	if command == "" {
		// General help - list all commands
		fmt.Print(helpGeneral)
		return
	}

	// Command-specific help
	switch command {
	case "login":
		fmt.Print(helpLogin)
	case "instances":
		fmt.Print(helpInstances)
	case "terminal":
		fmt.Print(helpTerminal)
	case "bastion":
		fmt.Print(helpBastion)
	case "bastions":
		fmt.Print(helpBastions)
	case "bastions list":
		fmt.Print(helpBastions)
	case "bastions add":
		fmt.Print(helpBastions)
	case "configure":
		fmt.Print(helpConfigure)
	case "docs":
		fmt.Print(helpDocs)
	case "help":
		fmt.Print(helpHelp)
	default:
		fmt.Printf(helpUnknown, command)
	}
}

