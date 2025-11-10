package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
)

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func login(args []string, config *Configuration) error {
	flagSet := flag.NewFlagSet("login", flag.ExitOnError)
	profileFlag := flagSet.String("profile", "", "--profile <aws cli profile>")
	profileShort := flagSet.String("p", "", "--profile <aws cli profile>")

	if err := flagSet.Parse(args); err != nil {
		return fmt.Errorf("USAGE: awsutil login [--profile <aws cli profile>]")
	}

	commandArgs := []string{"sso", "login"}

	if len(*profileFlag) != 0 {
		commandArgs = append(commandArgs, "--profile", *profileFlag)
	} else if len(*profileShort) != 0 {
		commandArgs = append(commandArgs, "--profile", *profileShort)
	} else if len(config.DefaultProfile) != 0 {
		commandArgs = append(commandArgs, "--profile", config.DefaultProfile)
	}

	command := exec.Command("aws", commandArgs...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin
	err := command.Start()

	if err != nil {
		return err
	}

	if err := command.Wait(); err != nil {
		return err
	}

	return nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func isLoggedIn(profile string) bool {
	//aws sts get-caller-identity --profile spg --query Account
	// if exit code is non-zero, then we're not logged in.

	args := []string{"sts", "get-caller-identity", "--query", "Account"}

	if len(profile) != 0 {
		args = append(args, "--profile", profile)
	}

	command := exec.Command("aws", args...)

	if err := command.Start(); err != nil {
		fmt.Printf("Failed to authenticate %s", err.Error())
		os.Exit(1)
	}

	if err := command.Wait(); err != nil {
		return false
	}

	return true
}

