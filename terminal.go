package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
)

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// startSSMSession starts a remote SSM terminal session against the specified instance.
func startSSMSession(args []string, config *Configuration) error {
	flagSet := flag.NewFlagSet("terminal", flag.ExitOnError)
	profile := flagSet.String("profile", "", "--profile <aws cli profile>")
	profileShort := flagSet.String("p", "", "--profile <aws cli profile>")

	flagSet.Usage = func() {
		fmt.Println("USAGE:\n    awsutil terminal [--profile <aws cli profile>] [<instance ID>]")
	}

	if err := flagSet.Parse(args); err != nil {
		flagSet.Usage()
		return fmt.Errorf("failed to parse options")
	}

	currentProfile, err := ensureProfile(config, profile, profileShort)
	if err != nil {
		return err
	}

	profileInfo := config.Profiles[currentProfile]

	if len(flagSet.Args()) == 0 && len(profileInfo.Instance) == 0 {
		flagSet.Usage()
		return fmt.Errorf("must specify the target instance ID")
	}

	config.DefaultProfile = currentProfile

	if len(flagSet.Args()) != 0 {
		profileInfo.Instance = flagSet.Args()[0]
		config.IsDirty = true
	}

	commandArgs := []string{
		"ssm",
		"start-session",
		"--target",
		profileInfo.Instance,
	}

	if len(config.DefaultProfile) != 0 {
		commandArgs = append(commandArgs, "--profile", currentProfile)
	}

	// Ensure that we're logged in before running the command.
	if !isLoggedIn(config.DefaultProfile) {
		args := []string{}

		if len(config.DefaultProfile) != 0 {
			args = append(args, "--profile", config.DefaultProfile)
		}

		login(args, config)
	}

	// Let's set up to prevent Ctrl-C from killing the program. Instead, it must
	// be handled with the SSM session.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)

	defer func() {
		signal.Stop(signalChan)
	}()

	select {
	case <-signalChan:
	default:
	}

	fmt.Println("\nStarting SSM session...")

	command := exec.Command("aws", commandArgs...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin

	if err = command.Run(); err != nil {
		return err
	}

	return nil
}
