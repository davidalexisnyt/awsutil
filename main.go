package main

/*
	- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
	awsdo
	Author: David Alexis

	This application wraps calls to the AWS CLI to simplify certain key tasks, like logging in,
	listing instances that match a given pattern, launching an SSM session, and starting a
	bastion host tunnel.
	- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
*/

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var Version = "0.1.0"

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func main() {
	exePath, _ := os.Executable()
	configFile := filepath.Join(filepath.Dir(exePath), "awsdo_config.json")

	if len(os.Args) < 2 {
		showHelp("")
		os.Exit(1)
	}

	config, err := loadConfiguration(configFile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	command := strings.ToLower(os.Args[1])

	switch command {
	case "help":
		if len(os.Args) > 2 {
			showHelp(strings.ToLower(os.Args[2]))
		} else {
			showHelp("")
		}
		return
	case "login":
		err = login(os.Args[2:], &config)
	case "instances":
		if len(os.Args) < 3 {
			// Default to 'list' if no subcommand provided
			err = listInstances([]string{}, &config)
		} else {
			subcommand := strings.ToLower(os.Args[2])
			switch subcommand {
			case "find":
				err = findInstances(os.Args[3:], &config)
			case "list", "ls":
				err = listInstances(os.Args[3:], &config)
			case "add":
				err = addInstance(os.Args[3:], &config)
			case "update":
				err = updateInstance(os.Args[3:], &config)
			case "remove", "rm":
				err = removeInstance(os.Args[3:], &config)
			default:
				fmt.Printf("Invalid instances subcommand: %s\n", subcommand)
				fmt.Println("Use 'awsdo instances find' to find instances, 'awsdo instances list' to list configured instances, 'awsdo instances add' to add an instance, 'awsdo instances update' to update an instance, 'awsdo instances remove' to remove an instance, or 'awsdo help instances' for more information.")
				os.Exit(1)
			}
		}
	case "terminal":
		err = startSSMSession(os.Args[2:], &config)
	case "bastion":
		err = startBastionTunnel(os.Args[2:], &config)
	case "bastions":
		if len(os.Args) < 3 {
			// Default to 'list' if no subcommand provided
			err = listBastions([]string{}, &config)
		} else {
			subcommand := strings.ToLower(os.Args[2])
			switch subcommand {
			case "list", "ls":
				err = listBastions(os.Args[3:], &config)
			case "add":
				err = addBastion(os.Args[3:], &config)
			case "update", "up":
				err = updateBastion(os.Args[3:], &config)
			case "remove", "rm":
				err = removeBastion(os.Args[3:], &config)
			default:
				fmt.Printf("Invalid bastions subcommand: %s\n", subcommand)
				fmt.Println("Use 'awsdo bastions list' to list bastions, 'awsdo bastions add' to add a new bastion, 'awsdo bastions update' to update an existing bastion, or 'awsdo bastions remove' to remove a bastion.")
				os.Exit(1)
			}
		}
	case "docs":
		showDocs()
		return
	case "repl":
		startREPL(configFile, &config)
		return
	case "init":
		err = initCommand(&config)
	default:
		fmt.Printf("Invalid command: %s\n", command)
		fmt.Println("Use 'awsdo help' to see available commands.")
		os.Exit(1)
	}

	if err != nil {
		fmt.Println(err.Error())
		fmt.Println()
	} else {
		saveConfiguration(configFile, &config)
	}
}
