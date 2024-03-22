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
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Configuration struct {
	Profile  string `json:"defaultProfile"`
	Instance string `json:"defaultInstance"`
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func main() {
	// os.Args = []string{"awsutil", "instances", "--profile", "spg", "zuoraetl"}
	exePath, _ := os.Executable()
	configFile := filepath.Join(filepath.Dir(exePath), "awsutil_config.json")

	if len(os.Args) < 2 {
		fmt.Println("USAGE: awsutil [login | instances] --profile <aws cli profile>")
		os.Exit(1)
	}

	config, err := loadConfiguration(configFile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	command := strings.ToLower(os.Args[1])

	switch command {
	case "login":
		err = login(os.Args[2:], &config)
	case "instances":
		err = listInstances(os.Args[2:], &config)
	case "terminal":
		err = startSSMSession(os.Args[2:], &config)
	case "configure":
		saveConfiguration(configFile, &config, os.Args[2:]...)
		return
	default:
		fmt.Printf("Invalid option: %s\n", command)
		fmt.Println("USAGE: awsutil [login | instances]")
		os.Exit(1)
	}

	if err != nil {
		fmt.Println(err.Error())
		fmt.Println()
	} else {
		saveConfiguration(configFile, &config)
	}
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func login(args []string, config *Configuration) error {
	flagSet := flag.NewFlagSet("login", flag.ExitOnError)
	profileFlag := flagSet.String("profile", "", "--profile <aws cli profile>")

	if err := flagSet.Parse(args); err != nil {
		return fmt.Errorf("USAGE: awsutil login [--profile <aws cli profile>]")
	}

	commandArgs := []string{"sso", "login"}

	if len(*profileFlag) != 0 {
		commandArgs = append(commandArgs, "--profile", *profileFlag)
	} else if len(config.Profile) != 0 {
		commandArgs = append(commandArgs, "--profile", config.Profile)
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
func listInstances(args []string, config *Configuration) error {
	const usageText string = "USAGE: awsutil instances [--profile <aws cli profile>] <filter string>"

	flagSet := flag.NewFlagSet("instances", flag.ExitOnError)
	profileFlag := flagSet.String("profile", "", "--profile <aws cli profile>")

	if err := flagSet.Parse(args); err != nil {
		return fmt.Errorf(usageText)
	}

	if len(flagSet.Args()) == 0 {
		return fmt.Errorf("must specify instance filter prefix\n%s", usageText)
	}

	filter := flagSet.Args()[0]

	if len(*profileFlag) != 0 {
		config.Profile = string(*profileFlag)
	}

	commandArgs := []string{
		"ec2",
		"describe-instances",
		"--query",
		"Reservations[*].Instances[*].{Instance:InstanceId,AZ:Placement.AvailabilityZone,Name:Tags[?Key=='Name']|[0].Value}",
		"--filters",
		fmt.Sprintf("Name=tag:Name,Values=%s*", filter),
		"--output=json",
	}

	if len(config.Profile) != 0 {
		commandArgs = append(commandArgs, "--profile", config.Profile)
	}

	// Ensure that we're logged in before running the command.
	if !isLoggedIn(config.Profile) {
		args := []string{}

		if len(config.Profile) != 0 {
			args = append(args, "--profile", config.Profile)
		}

		login(args, config)
	}

	fmt.Println("\nInstances")

	command := exec.Command("aws", commandArgs...)
	outputStream, err := command.StdoutPipe()
	if err != nil {
		return err
	}

	errorStream, err := command.StderrPipe()
	if err != nil {
		return err
	}

	go func() {
		scanner := bufio.NewScanner(errorStream)
		scanner.Split(bufio.ScanLines)

		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

	err = command.Start()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(outputStream)
	scanner.Split(bufio.ScanLines)
	outputDoc := strings.Builder{}

	for scanner.Scan() {
		outputDoc.WriteString(strings.Trim(scanner.Text(), " "))
	}

	command.Wait()
	output := outputDoc.String()

	/* Output is an array of an array of instance documents like below.
	[
		[
			{
				"Instance": "i-0001",
				"AZ": "us-east-1a",
				"Name": "my-instance-1"
			}
		],
		[
			{
				"Instance": "i-0002",
				"AZ": "us-east-1a",
				"Name": "my-instance-2"
			}
		]
	]
	*/

	if len(output) == 0 {
		fmt.Println("AWS command failed to return data")
	}

	var instanceList [][]map[string]string

	if err := json.Unmarshal([]byte(output), &instanceList); err != nil {
		return err
	}

	if len(instanceList) == 1 {
		config.Instance = instanceList[0][0]["Instance"]
	}

	for i := range len(instanceList) {
		fmt.Printf("    %s: %s\n", instanceList[i][0]["Name"], instanceList[i][0]["Instance"])
	}

	fmt.Println()

	return nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
// startSSMSession starts a remote SSM terminal session against the specified instance.
func startSSMSession(args []string, config *Configuration) error {
	const usageText string = "USAGE: awsutil terminal [--profile <aws cli profile>] [<instance ID>]"

	flagSet := flag.NewFlagSet("ssm_session", flag.ExitOnError)
	profileFlag := flagSet.String("profile", "", "--profile <aws cli profile>")

	if err := flagSet.Parse(args); err != nil {
		return fmt.Errorf(usageText)
	}

	if len(flagSet.Args()) == 0 && len(config.Instance) == 0 {
		return fmt.Errorf("must specify the target instance ID\n%s", usageText)
	}

	if len(*profileFlag) != 0 {
		config.Profile = string(*profileFlag)
	}

	if len(flagSet.Args()) != 0 {
		config.Instance = flagSet.Args()[0]
	}

	if len(flagSet.Args()) != 0 {
		config.Instance = flagSet.Args()[0]
	}

	commandArgs := []string{
		"ssm",
		"start-session",
		"--target",
		config.Instance,
	}

	if len(config.Profile) != 0 {
		commandArgs = append(commandArgs, "--profile", config.Profile)
	}

	// Ensure that we're logged in before running the command.
	if !isLoggedIn(config.Profile) {
		args := []string{}

		if len(config.Profile) != 0 {
			args = append(args, "--profile", config.Profile)
		}

		login(args, config)
	}

	fmt.Println("\nStarting SSM session...")

	command := exec.Command("aws", commandArgs...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin

	err := command.Run()
	if err != nil {
		return fmt.Errorf(err.Error())
	}

	return nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func loadConfiguration(fileName string) (Configuration, error) {
	if _, err := os.Stat(fileName); err != nil {
		return Configuration{}, nil
	}

	var config Configuration
	configBytes, err := os.ReadFile(fileName)

	if err != nil {
		return Configuration{}, fmt.Errorf("could not read config.json file")
	}

	if err := json.Unmarshal(configBytes, &config); err != nil {
		return Configuration{}, fmt.Errorf("could not read config.json file")
	}

	return config, nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func saveConfiguration(fileName string, config *Configuration, options ...string) {
	// Update the configuration with any supplied arguements before saving
	if len(options) != 0 {
		usageText := `USAGE:
			awsutil configure --profile <profile> --instance <instance ID>
		`

		flagSet := flag.NewFlagSet("config", flag.ExitOnError)
		profile := flagSet.String("profile", "", "--profile <aws cli profile>")
		instance := flagSet.String("instance", "", "--instance <aws instance ID>")

		if err := flagSet.Parse(options); err != nil {
			fmt.Print(usageText)
			return
		}

		if len(*profile) != 0 {
			config.Profile = *profile
		}

		if len(*instance) != 0 {
			config.Instance = *instance
		}
	}

	// Save the configuration file
	configBytes, _ := json.MarshalIndent(config, "", "    ")
	os.WriteFile(fileName, configBytes, 0644)
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
		fmt.Printf("Failed to authenticate %s", err.Error())
		return false
	}

	return true
}
