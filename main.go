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
	"awsutil/markdown"
	"bufio"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
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

type Configuration struct {
	DefaultProfile string             `json:"defaultProfile,omitempty"`
	Profiles       map[string]Profile `json:"profiles,omitempty"`
}

type Profile struct {
	Name           string             `json:"name,omitempty"`
	Instance       string             `json:"instance,omitempty"`
	Bastion        Bastion            `json:"bastion,omitempty"`        // Deprecated: use Bastions instead
	Bastions       map[string]Bastion `json:"bastions,omitempty"`       // New: multiple named bastions
	DefaultBastion string             `json:"defaultBastion,omitempty"` // Default bastion name
}

type Bastion struct {
	Name      string `json:"name,omitempty"`
	Instance  string `json:"instance,omitempty"`
	Host      string `json:"host,omitempty"`
	Port      int    `json:"port,omitempty"`
	LocalPort int    `json:"localPort,omitempty"`
}

type RDSDatabase struct {
	DBInstanceIdentifier string `json:"ID"`
	Endpoint             string `json:"Endpoint"`
	Port                 int    `json:"Port"`
	Engine               string `json:"Engine"`
}

type EC2Instance struct {
	Instance string `json:"Instance"`
	Name     string `json:"Name"`
	AZ       string `json:"AZ"`
}

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
		err = listInstances(os.Args[2:], &config)
	case "terminal":
		err = startSSMSession(os.Args[2:], &config)
	case "configure":
		saveConfiguration(configFile, &config, os.Args[2:]...)
		return
	case "bastion":
		err = startBastionTunnel(os.Args[2:], &config)
	case "bastions":
		err = listBastions(os.Args[2:], &config)
	case "docs":
		showDocs()
		return
	default:
		fmt.Printf("Invalid option: %s\n", command)
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

		if profileInfo.Bastions == nil {
			profileInfo.Bastions = make(map[string]Bastion)
		}

		if strings.Contains(instanceList[0][0]["Name"], "bastion") {
			// Update default bastion instance if it exists, otherwise create one
			defaultBastionName := "default"
			if profileInfo.DefaultBastion != "" {
				defaultBastionName = profileInfo.DefaultBastion
			}
			bastion := profileInfo.Bastions[defaultBastionName]
			bastion.Instance = instanceList[0][0]["Instance"]
			profileInfo.Bastions[defaultBastionName] = bastion
			if profileInfo.DefaultBastion == "" {
				profileInfo.DefaultBastion = defaultBastionName
			}
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
	// Initialize Profiles map if nil
	if config.Profiles == nil {
		config.Profiles = make(map[string]Profile)
	}

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
		// Ensure profile exists in map
		if _, exists := config.Profiles[currentProfile]; !exists {
			config.Profiles[currentProfile] = Profile{
				Name:     currentProfile,
				Bastions: make(map[string]Bastion),
			}
		}
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
func listBastions(args []string, config *Configuration) error {
	flagSet := flag.NewFlagSet("bastions", flag.ExitOnError)
	profile := flagSet.String("profile", "", "--profile <aws cli profile>")
	profileShort := flagSet.String("p", "", "--profile <aws cli profile>")
	flagSet.Usage = func() {
		fmt.Println("USAGE:\n    awsutil bastions [--profile <aws cli profile>]")
	}

	if err := flagSet.Parse(args); err != nil {
		flagSet.Usage()
		return fmt.Errorf("failed to parse options")
	}

	currentProfile, err := ensureProfile(config, profile, profileShort)
	if err != nil {
		return err
	}

	// Ensure that we're logged in before running the command
	if !isLoggedIn(currentProfile) {
		loginArgs := []string{"--profile", currentProfile}
		if err := login(loginArgs, config); err != nil {
			return err
		}
	}

	profileInfo := config.Profiles[currentProfile]
	if profileInfo.Bastions == nil {
		profileInfo.Bastions = make(map[string]Bastion)
	}

	// List existing bastions
	fmt.Printf("\nBastions for profile '%s':\n", currentProfile)
	if len(profileInfo.Bastions) == 0 {
		fmt.Println("  No bastions configured.")
	} else {
		fmt.Printf("%-20s %-20s %-50s %-8s %-12s\n", "Name", "Instance ID", "Host", "Port", "Local Port")
		fmt.Println(strings.Repeat("-", 112))
		for name, bastion := range profileInfo.Bastions {
			defaultMarker := ""
			if profileInfo.DefaultBastion == name {
				defaultMarker = " (default)"
			}
			fmt.Printf("%-20s %-20s %-50s %-8d %-12d%s\n",
				name, bastion.Instance, bastion.Host, bastion.Port, bastion.LocalPort, defaultMarker)
		}
	}

	// Interactive configuration
	fmt.Println("\nWould you like to configure a new bastion? (y/n): ")
	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(strings.ToLower(response))

	if response != "y" && response != "yes" {
		return nil
	}

	// Query RDS databases
	fmt.Println("\nQuerying RDS databases...")
	databases, err := queryRDSDatabases(currentProfile)
	if err != nil {
		return fmt.Errorf("failed to query RDS databases: %v", err)
	}

	if len(databases) == 0 {
		fmt.Println("No RDS databases found.")
		return nil
	}

	// Display databases and let user select
	fmt.Println("\nAvailable RDS databases:")
	for i, db := range databases {
		fmt.Printf("  %d. %s (%s) - %s:%d\n", i+1, db.DBInstanceIdentifier, db.Engine, db.Endpoint, db.Port)
	}

	fmt.Print("\nSelect database number (or 0 to skip): ")
	dbSelection, _ := reader.ReadString('\n')
	dbIndex, err := strconv.Atoi(strings.TrimSpace(dbSelection))
	if err != nil || dbIndex < 0 || dbIndex > len(databases) {
		return fmt.Errorf("invalid selection")
	}

	var selectedDB *RDSDatabase
	if dbIndex > 0 {
		selectedDB = &databases[dbIndex-1]
	}

	// Query bastion instances
	fmt.Println("\nQuerying bastion instances...")
	bastionInstances, err := queryBastionInstances(currentProfile)
	if err != nil {
		return fmt.Errorf("failed to query bastion instances: %v", err)
	}

	if len(bastionInstances) == 0 {
		fmt.Println("No bastion instances found.")
		return nil
	}

	// Display bastion instances and let user select
	fmt.Println("\nAvailable bastion instances:")
	for i, inst := range bastionInstances {
		fmt.Printf("  %d. %s (%s)\n", i+1, inst.Name, inst.Instance)
	}

	fmt.Print("\nSelect bastion instance number: ")
	instSelection, _ := reader.ReadString('\n')
	instIndex, err := strconv.Atoi(strings.TrimSpace(instSelection))
	if err != nil || instIndex < 1 || instIndex > len(bastionInstances) {
		return fmt.Errorf("invalid selection")
	}

	selectedBastionInstance := bastionInstances[instIndex-1]

	// Get bastion name
	fmt.Print("\nEnter bastion name: ")
	bastionName, _ := reader.ReadString('\n')
	bastionName = strings.TrimSpace(bastionName)
	if bastionName == "" {
		// Generate a default name from database identifier
		if selectedDB != nil {
			bastionName = selectedDB.DBInstanceIdentifier
		} else {
			bastionName = fmt.Sprintf("bastion-%d", len(profileInfo.Bastions)+1)
		}
	}

	// Create bastion configuration
	newBastion := Bastion{
		Name:     bastionName,
		Instance: selectedBastionInstance.Instance,
	}

	if selectedDB != nil {
		newBastion.Host = selectedDB.Endpoint
		newBastion.Port = selectedDB.Port
	} else {
		// Prompt for host and port
		fmt.Print("Enter remote host: ")
		host, _ := reader.ReadString('\n')
		newBastion.Host = strings.TrimSpace(host)

		fmt.Print("Enter remote port: ")
		portStr, _ := reader.ReadString('\n')
		port, err := strconv.Atoi(strings.TrimSpace(portStr))
		if err != nil {
			return fmt.Errorf("invalid port: %v", err)
		}
		newBastion.Port = port
	}

	// Find available local port
	localPort, err := findAvailableLocalPort(7000)
	if err != nil {
		return fmt.Errorf("failed to find available local port: %v", err)
	}

	fmt.Printf("Using local port: %d\n", localPort)
	fmt.Print("Enter local port (or press Enter to use suggested): ")
	localPortStr, _ := reader.ReadString('\n')
	localPortStr = strings.TrimSpace(localPortStr)
	if localPortStr != "" {
		customPort, err := strconv.Atoi(localPortStr)
		if err == nil {
			localPort = customPort
		}
	}

	newBastion.LocalPort = localPort

	// Save to configuration
	profileInfo.Bastions[bastionName] = newBastion
	if profileInfo.DefaultBastion == "" {
		profileInfo.DefaultBastion = bastionName
	}
	profileInfo.Name = currentProfile
	config.Profiles[currentProfile] = profileInfo

	fmt.Printf("\nBastion '%s' configured successfully!\n", bastionName)

	return nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func startBastionTunnel(args []string, config *Configuration) error {
	flagSet := flag.NewFlagSet("bastion", flag.ExitOnError)
	profile := flagSet.String("profile", "", "--profile <aws cli profile>")
	profileShort := flagSet.String("p", "", "--profile <aws cli profile>")
	bastionName := flagSet.String("name", "", "--name <bastion name>")
	bastionInstsance := flagSet.String("instance", "", "--instance <aws instance ID>")
	bastionHost := flagSet.String("host", "", "--host <bastion host name>")
	bastionPort := flagSet.Int("port", 0, "--port <port to forward>")
	bastionLocalPort := flagSet.Int("local", 0, "--local <local port>")
	flagSet.Usage = func() {
		fmt.Println("USAGE:")
		fmt.Println("    awsutil bastion [--profile <aws cli profile>] [--name <bastion name>]")
		fmt.Println("                    [--instance <instance id>] [--host <remote host>]")
		fmt.Println("                    [--port <remote port>] [--local <local port>]")
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
	if profileInfo.Bastions == nil {
		profileInfo.Bastions = make(map[string]Bastion)
	}

	// Try to get bastion from saved configuration
	var bastion Bastion
	if *bastionName != "" || len(profileInfo.Bastions) > 0 {
		selectedBastion, err := selectBastionByName(profileInfo, *bastionName)
		if err == nil {
			bastion = selectedBastion
		}
	}

	// Override with command line arguments if provided
	if len(*bastionInstsance) != 0 {
		bastion.Instance = *bastionInstsance
	}

	if len(*bastionHost) != 0 {
		bastion.Host = *bastionHost
	}

	if *bastionPort != 0 {
		bastion.Port = *bastionPort
	}

	if *bastionLocalPort != 0 {
		bastion.LocalPort = *bastionLocalPort
	}

	// Verify required configuration
	if bastion.Instance == "" {
		return fmt.Errorf("bastion instance ID must be specified")
	}

	if bastion.Host == "" {
		return fmt.Errorf("bastion host must be specified")
	}

	if bastion.Port == 0 {
		return fmt.Errorf("remote port must be specified")
	}

	if bastion.LocalPort == 0 {
		// Auto-assign local port if not specified
		localPort, err := findAvailableLocalPort(7000)
		if err != nil {
			return fmt.Errorf("failed to find available local port: %v", err)
		}
		bastion.LocalPort = localPort
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
		bastion.Instance,
		"--document-name",
		"AWS-StartPortForwardingSessionToRemoteHost",
		"--parameters",
		fmt.Sprintf(`host="%s",portNumber="%d",localPortNumber="%d"`, bastion.Host, bastion.Port, bastion.LocalPort),
	}

	if len(currentProfile) != 0 {
		commandArgs = append(commandArgs, "--profile", currentProfile)
	}

	// Ensure that we're logged in before running the command
	if !isLoggedIn(currentProfile) {
		args := []string{}

		if len(currentProfile) != 0 {
			args = append(args, "--profile", currentProfile)
		}

		login(args, config)
	}

	fmt.Printf("\nStarting port forwarding session to %s:%d via bastion %s...\n", bastion.Host, bastion.Port, bastion.Instance)

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
func queryRDSDatabases(profile string) ([]RDSDatabase, error) {
	commandArgs := []string{
		"rds",
		"describe-db-instances",
		"--query",
		"DBInstances[*].{ID:DBInstanceIdentifier,Endpoint:Endpoint.Address,Port:Endpoint.Port,Engine:Engine}",
		"--output=json",
	}

	if len(profile) != 0 {
		commandArgs = append(commandArgs, "--profile", profile)
	}

	command := exec.Command("aws", commandArgs...)
	outputStream, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}

	errorStream, err := command.StderrPipe()
	if err != nil {
		return nil, err
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
		return nil, err
	}

	scanner := bufio.NewScanner(outputStream)
	scanner.Split(bufio.ScanLines)
	outputDoc := strings.Builder{}

	for scanner.Scan() {
		outputDoc.WriteString(strings.Trim(scanner.Text(), " "))
	}

	command.Wait()
	output := outputDoc.String()

	if len(output) == 0 {
		return []RDSDatabase{}, nil
	}

	var databases []RDSDatabase
	if err := json.Unmarshal([]byte(output), &databases); err != nil {
		return nil, fmt.Errorf("failed to parse RDS database list: %v", err)
	}

	return databases, nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func queryBastionInstances(profile string) ([]EC2Instance, error) {
	commandArgs := []string{
		"ec2",
		"describe-instances",
		"--query",
		"Reservations[*].Instances[*].{Instance:InstanceId,AZ:Placement.AvailabilityZone,Name:Tags[?Key=='Name']|[0].Value}",
		"--filters",
		"Name=tag:Name,Values=*bastion*",
		"--output=json",
	}

	if len(profile) != 0 {
		commandArgs = append(commandArgs, "--profile", profile)
	}

	command := exec.Command("aws", commandArgs...)
	outputStream, err := command.StdoutPipe()
	if err != nil {
		return nil, err
	}

	errorStream, err := command.StderrPipe()
	if err != nil {
		return nil, err
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
		return nil, err
	}

	scanner := bufio.NewScanner(outputStream)
	scanner.Split(bufio.ScanLines)
	outputDoc := strings.Builder{}

	for scanner.Scan() {
		outputDoc.WriteString(strings.Trim(scanner.Text(), " "))
	}

	command.Wait()
	output := outputDoc.String()

	if len(output) == 0 {
		return []EC2Instance{}, nil
	}

	var instanceList [][]EC2Instance
	if err := json.Unmarshal([]byte(output), &instanceList); err != nil {
		return nil, fmt.Errorf("failed to parse EC2 instance list: %v", err)
	}

	var instances []EC2Instance
	for _, reservation := range instanceList {
		for _, instance := range reservation {
			if instance.Instance != "" {
				instances = append(instances, instance)
			}
		}
	}

	return instances, nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func findAvailableLocalPort(startPort int) (int, error) {
	for port := startPort; port < startPort+1000; port++ {
		addr, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			addr.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("could not find available port starting from %d", startPort)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func selectBastionByName(profileInfo Profile, name string) (Bastion, error) {
	if len(profileInfo.Bastions) == 0 {
		return Bastion{}, fmt.Errorf("no bastions configured for this profile")
	}

	// If name is provided, use it
	if name != "" {
		if bastion, exists := profileInfo.Bastions[name]; exists {
			return bastion, nil
		}
		return Bastion{}, fmt.Errorf("bastion '%s' not found", name)
	}

	// If no name provided, try default
	if profileInfo.DefaultBastion != "" {
		if bastion, exists := profileInfo.Bastions[profileInfo.DefaultBastion]; exists {
			return bastion, nil
		}
	}

	// If only one bastion exists, use it
	if len(profileInfo.Bastions) == 1 {
		for _, bastion := range profileInfo.Bastions {
			return bastion, nil
		}
	}

	// Multiple bastions exist, need to specify name
	return Bastion{}, fmt.Errorf("multiple bastions available, please specify --name")
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

	// Migrate old single-bastion config to new multi-bastion format
	migrateBastionConfig(&config)

	return config, nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func migrateBastionConfig(config *Configuration) {
	if config.Profiles == nil {
		config.Profiles = make(map[string]Profile)
	}

	for profileName, profile := range config.Profiles {
		// Check if old Bastion field exists and has data, but Bastions map doesn't exist or is empty
		if profile.Bastion.Instance != "" || profile.Bastion.Host != "" {
			if profile.Bastions == nil {
				profile.Bastions = make(map[string]Bastion)
			}

			// Only migrate if "default" doesn't already exist
			if _, exists := profile.Bastions["default"]; !exists {
				profile.Bastions["default"] = profile.Bastion
				if profile.DefaultBastion == "" {
					profile.DefaultBastion = "default"
				}
			}
		}

		config.Profiles[profileName] = profile
	}
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func saveConfiguration(fileName string, config *Configuration, options ...string) {
	// Update the configuration with any supplied arguements before saving
	if len(options) != 0 {
		usageText := `USAGE:
		    awsutil configure [--profile <aws cli profile>] [--instance <instance id>] [--bastion-name <name>] [--bastion-instance <value>] [--bastion-host <value>] [--bastion-port <value>] [--bastion-local <value>]
		`

		flagSet := flag.NewFlagSet("configure", flag.ExitOnError)

		profile := flagSet.String("profile", "", "--profile <aws cli profile>")
		profileShort := flagSet.String("p", "", "--profile <aws cli profile>")

		instance := flagSet.String("instance", "", "--instance <aws instance ID>")
		instanceShort := flagSet.String("i", "", "--instance <aws instance ID>")

		bastionName := flagSet.String("bastion-name", "", "--bastion-name <bastion name>")
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
		if profileInfo.Bastions == nil {
			profileInfo.Bastions = make(map[string]Bastion)
		}

		if len(*instance) != 0 {
			profileInfo.Instance = *instance
		} else if len(*instanceShort) != 0 {
			profileInfo.Instance = *instanceShort
		}

		// Determine which bastion to update
		bastionKey := *bastionName
		if bastionKey == "" {
			// Use default bastion if no name specified
			if profileInfo.DefaultBastion != "" {
				bastionKey = profileInfo.DefaultBastion
			} else {
				bastionKey = "default"
			}
		}

		// Get or create bastion
		bastion := profileInfo.Bastions[bastionKey]
		bastion.Name = bastionKey

		if len(*bastionInstsance) != 0 {
			bastion.Instance = *bastionInstsance
		}

		if len(*bastionHost) != 0 {
			bastion.Host = *bastionHost
		}

		if *bastionPort != 0 {
			bastion.Port = *bastionPort
		}

		if *bastionLocalPort != 0 {
			bastion.LocalPort = *bastionLocalPort
		}

		// Save bastion back to map
		profileInfo.Bastions[bastionKey] = bastion
		if profileInfo.DefaultBastion == "" {
			profileInfo.DefaultBastion = bastionKey
		}

		config.Profiles[currentProfile] = profileInfo
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
