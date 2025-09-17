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
	"os/signal"
	"path/filepath"
	"strings"
)

type Configuration struct {
	DefaultProfile string             `json:"defaultProfile,omitempty"`
	Profiles       map[string]Profile `json:"profiles,omitempty"`
}

type Profile struct {
	Name     string  `json:"name,omitempty"`
	Instance string  `json:"instance,omitempty"`
	Bastion  Bastion `json:"bastion,omitempty"`
}

type Bastion struct {
	Instance  string `json:"instance,omitempty"`
	Host      string `json:"host,omitempty"`
	Port      int    `json:"port,omitempty"`
	LocalPort int    `json:"localPort,omitempty"`
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func main() {
	exePath, _ := os.Executable()
	configFile := filepath.Join(filepath.Dir(exePath), "awsutil_config.json")

	if len(os.Args) < 2 {
		fmt.Println("USAGE: awsutil login --profile <aws cli profile>")
		fmt.Println("       awsutil instances [--profile <aws cli profile>] <filter prefix")
		fmt.Println("       awsutil terminal [--profile <aws cli profile>] [<instance id>]")
		fmt.Println("       awsutil bastion [--profile <aws cli profile>] [--instance <bastion instance id>] [--host <remote host>] [--port <remote port>] [--local <local port>]")
		fmt.Println("       awsutil configure [--profile <aws cli profile>] [--instance <instance id>] [--bastion-instance <value>] [--bastion-host <value>] [--bastion-port <value>] [--bastion-local <value>]")
		fmt.Println()
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
	case "bastion":
		err = startBastionTunnel(os.Args[2:], &config)
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
func listInstances(args []string, config *Configuration) error {
	flagSet := flag.NewFlagSet("instances", flag.ExitOnError)
	profile := flagSet.String("profile", "", "--profile <aws cli profile>")
	profileShort := flagSet.String("p", "", "--profile <aws cli profile>")
	flagSet.Usage = func() {
		fmt.Println("USAGE:\n    awsutil instances [--profile <aws cli profile>] <filter string>")
	}

	if err := flagSet.Parse(args); err != nil {
		flagSet.Usage()
		return fmt.Errorf("failed to parse options")
	}

	if len(flagSet.Args()) == 0 {
		flagSet.Usage()
		return fmt.Errorf("must specify instance filter prefix")
	}

	filter := flagSet.Args()[0]
	currentProfile, err := ensureProfile(config, profile, profileShort)
	if err != nil {
		return err
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

	commandArgs = append(commandArgs, "--profile", currentProfile)

	fmt.Printf("\nInstances (%s)\n", currentProfile)

	// Ensure that we're logged in before running the command.
	if !isLoggedIn(currentProfile) {
		args := []string{}
		args = append(args, "--profile", currentProfile)

		login(args, config)
	}

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
		profileInfo := config.Profiles[currentProfile]
		profileInfo.Name = currentProfile

		if strings.Contains(instanceList[0][0]["Name"], "bastion") {
			profileInfo.Bastion.Instance = instanceList[0][0]["Instance"]
		} else {
			profileInfo.Instance = instanceList[0][0]["Instance"]
		}

		config.Profiles[currentProfile] = profileInfo
	}

	for i := range len(instanceList) {
		fmt.Printf("    %s: %s\n", instanceList[i][0]["Name"], instanceList[i][0]["Instance"])
	}

	fmt.Println()

	return nil
}

func ensureProfile(config *Configuration, profile *string, profileShort *string) (string, error) {
	currentProfile := config.DefaultProfile

	if len(*profile) != 0 {
		currentProfile = *profile
	} else if len(*profileShort) != 0 {
		currentProfile = *profileShort
	}

	// Set the default profile in the configuration if currentProfile is not empty
	// Otherewise fail with an error
	if len(currentProfile) != 0 {
		config.DefaultProfile = currentProfile
	} else if len(config.DefaultProfile) == 0 {
		return "", fmt.Errorf("must specify the target profile")
	}
	return currentProfile, nil
}

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

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func startBastionTunnel(args []string, config *Configuration) error {
	flagSet := flag.NewFlagSet("bastion", flag.ExitOnError)
	profile := flagSet.String("profile", "", "--profile <aws cli profile>")
	profileShort := flagSet.String("p", "", "--profile <aws cli profile>")
	bastionInstsance := flagSet.String("instance", "", "--instance <aws instance ID>")
	bastionHost := flagSet.String("host", "", "--host <bastion host name>")
	bastionPort := flagSet.Int("port", 0, "--port <port to forward>")
	bastionLocalPort := flagSet.Int("local", 0, "-local <local port>")
	flagSet.Usage = func() {
		fmt.Println("USAGE:")
		fmt.Println("    awsutil bastion [--profile <aws cli profile>] [--instance <instance id>]")
		fmt.Println("                    [--host <remote host>] [--port <remote port>] [--local <local port>]")
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

	// Update bastion configuration from command line arguments
	if len(*bastionInstsance) != 0 {
		profileInfo.Bastion.Instance = *bastionInstsance
	}

	if len(*bastionHost) != 0 {
		profileInfo.Bastion.Host = *bastionHost
	}

	if *bastionPort != 0 {
		profileInfo.Bastion.Port = *bastionPort
	}

	if *bastionLocalPort != 0 {
		profileInfo.Bastion.LocalPort = *bastionLocalPort
	}

	// Set instance from command line arguments if provided
	if len(flagSet.Args()) != 0 {
		profileInfo.Instance = flagSet.Args()[0]
	}

	// Verify required configuration
	if profileInfo.Bastion.Host == "" {
		return fmt.Errorf("bastion host must be specified")
	}

	if profileInfo.Bastion.Port == 0 {
		return fmt.Errorf("remote port must be specified")
	}

	if profileInfo.Bastion.LocalPort == 0 {
		return fmt.Errorf("local port must be specified")
	}

	// Check if Session Manager plugin is installed
	pluginCheck := exec.Command("session-manager-plugin")
	if err := pluginCheck.Run(); err != nil {
		return fmt.Errorf("AWS Session Manager plugin is not installed. Please install it first")
	}

	commandArgs := []string{
		"ssm",
		"start-session",
		"--target",
		profileInfo.Bastion.Instance,
		"--document-name",
		"AWS-StartPortForwardingSessionToRemoteHost",
		"--parameters",
		fmt.Sprintf(`host="%s",portNumber="%d",localPortNumber="%d"`, profileInfo.Bastion.Host, profileInfo.Bastion.Port, profileInfo.Bastion.LocalPort),
	}

	if len(config.DefaultProfile) != 0 {
		commandArgs = append(commandArgs, "--profile", config.DefaultProfile)
	}

	// Ensure that we're logged in before running the command
	if !isLoggedIn(config.DefaultProfile) {
		args := []string{}

		if len(config.DefaultProfile) != 0 {
			args = append(args, "--profile", config.DefaultProfile)
		}

		login(args, config)
	}

	fmt.Printf("\nStarting port forwarding session to %s:%d via bastion...\n", profileInfo.Bastion.Host, profileInfo.Bastion.Port)

	command := exec.Command("aws", commandArgs...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin

	if err := command.Start(); err != nil {
		return fmt.Errorf("failed to start session: %v", err)
	}

	// Wait for the command to complete or be interrupted
	if err := command.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// If the process was terminated by a signal (e.g. Ctrl+C), don't treat it as an error
			if exitErr.ExitCode() == -1 {
				return nil
			}
		}
		return fmt.Errorf("session ended with error: %v", err)
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
		    awsutil configure [--profile <aws cli profile>] [--instance <instance id>] [--bastion-instance <value>] [--bastion-host <value>] [--bastion-port <value>] [--bastion-local <value>]
		`

		flagSet := flag.NewFlagSet("configure", flag.ExitOnError)

		profile := flagSet.String("profile", "", "--profile <aws cli profile>")
		profileShort := flagSet.String("p", "", "--profile <aws cli profile>")

		instance := flagSet.String("instance", "", "--instance <aws instance ID>")
		instanceShort := flagSet.String("i", "", "--instance <aws instance ID>")

		bastionInstsance := flagSet.String("bastion-instance", "", "--bastion-instance <aws instance ID>")
		bastionHost := flagSet.String("bastion-host", "", "--bastion-host <bastion host name>")
		bastionPort := flagSet.Int("bastion-port", 0, "--bastion-port <port to forward>")
		bastionLocalPort := flagSet.Int("bastion-local", 0, "--bastion-local <local port>")

		if err := flagSet.Parse(options); err != nil {
			fmt.Print(usageText)
			return
		}

		currentProfile, err := ensureProfile(config, profile, profileShort)
		if err != nil {
			return
		}

		profileInfo := config.Profiles[currentProfile]

		if len(*instance) != 0 {
			profileInfo.Instance = *instance
		} else if len(*instanceShort) != 0 {
			profileInfo.Instance = *instanceShort
		}

		if len(*bastionInstsance) != 0 {
			profileInfo.Bastion.Instance = *bastionInstsance
		}

		if len(*bastionHost) != 0 {
			profileInfo.Bastion.Host = *bastionHost
		}

		if *bastionPort != 0 {
			profileInfo.Bastion.Port = *bastionPort
		}
		if *bastionLocalPort != 0 {
			profileInfo.Bastion.LocalPort = *bastionLocalPort
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
		return false
	}

	return true
}
