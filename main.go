package main

/*
	- - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
	awsutil
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

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func main() {
	exePath, _ := os.Executable()
	configFile := filepath.Join(filepath.Dir(exePath), "awsutil_config.json")

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
			fmt.Printf("Invalid instances command: subcommand required\n")
			fmt.Println("Use 'awsutil instances find' to find instances, 'awsutil instances list' to list configured instances, 'awsutil instances add' to add an instance, 'awsutil instances update' to update an instance, 'awsutil instances remove' to remove an instance, or 'awsutil help instances' for more information.")
			os.Exit(1)
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
				fmt.Println("Use 'awsutil instances find' to find instances, 'awsutil instances list' to list configured instances, 'awsutil instances add' to add an instance, 'awsutil instances update' to update an instance, 'awsutil instances remove' to remove an instance, or 'awsutil help instances' for more information.")
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
			err = listBastions(os.Args[2:], &config)
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
				fmt.Println("Use 'awsutil bastions list' to list bastions, 'awsutil bastions add' to add a new bastion, 'awsutil bastions update' to update an existing bastion, or 'awsutil bastions remove' to remove a bastion.")
				os.Exit(1)
			}
		}
	case "docs":
		showDocs()
		return
	case "repl":
		startREPL(configFile, &config)
		return
	default:
		fmt.Printf("Invalid command: %s\n", command)
		fmt.Println("Use 'awsutil help' to see available commands.")
		os.Exit(1)
	}

	if err != nil {
		fmt.Println(err.Error())
		fmt.Println()
	} else {
		saveConfiguration(configFile, &config)
	}
}
