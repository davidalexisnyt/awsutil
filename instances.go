package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func findInstances(args []string, config *Configuration) error {
	flagSet := flag.NewFlagSet("instances find", flag.ExitOnError)
	profile := flagSet.String("profile", "", "--profile <aws cli profile>")
	profileShort := flagSet.String("p", "", "--profile <aws cli profile>")

	flagSet.Usage = func() {
		fmt.Println("USAGE:\n    awsutil instances find [--profile <aws cli profile>] <filter string>")
	}

	if err := flagSet.Parse(args); err != nil {
		flagSet.Usage()
		return fmt.Errorf("failed to parse options")
	}

	if len(flagSet.Args()) == 0 {
		flagSet.Usage()
		return fmt.Errorf("must specify instance filter string")
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
		fmt.Sprintf("Name=tag:Name,Values=*%s*", filter),
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

	if len(instanceList) == 1 && !strings.Contains(filter, "bastion") {
		profileInfo := config.Profiles[currentProfile]
		profileInfo.Name = currentProfile

		// Initialize Instances map if nil
		if profileInfo.Instances == nil {
			profileInfo.Instances = make(map[string]Instance)
		}

		// Create a "default" entry in Instances map
		instanceID := instanceList[0][0]["Instance"]
		instanceName := instanceList[0][0]["Name"]
		if instanceName == "" {
			instanceName = "default"
		}

		// Query for host (private IP) - we'll use a simple approach for now
		// In a full implementation, we'd query for the private IP
		host := instanceID // Placeholder - could be improved to query for actual private IP

		profileInfo.Instances["default"] = Instance{
			Name:    "default",
			ID:      instanceID,
			Profile: currentProfile,
			Host:    host,
		}

		// Set DefaultInstance to "default"
		profileInfo.DefaultInstance = "default"

		// Keep backward compatibility with old Instance field
		profileInfo.Instance = instanceID

		config.Profiles[currentProfile] = profileInfo
	}

	for i := range len(instanceList) {
		fmt.Printf("    %s\t %s\n", instanceList[i][0]["Instance"], instanceList[i][0]["Name"])
	}

	fmt.Println()

	return nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func listInstances(args []string, config *Configuration) error {
	flagSet := flag.NewFlagSet("instances list", flag.ExitOnError)
	profile := flagSet.String("profile", "", "--profile <aws cli profile>")
	profileShort := flagSet.String("p", "", "--profile <aws cli profile>")

	flagSet.Usage = func() {
		fmt.Println("USAGE:\n    awsutil instances list [--profile <aws cli profile>]")
	}

	if err := flagSet.Parse(args); err != nil {
		flagSet.Usage()
		return fmt.Errorf("failed to parse options")
	}

	// List all instances across all profiles
	if config.Profiles == nil {
		fmt.Println("\nNo instances configured.")
		fmt.Println()
		return nil
	}

	hasInstances := false

	for profileName, profileInfo := range config.Profiles {
		// If profile filter is specified, only show that profile
		if *profile != "" || *profileShort != "" {
			targetProfile := *profile

			if *profileShort != "" {
				targetProfile = *profileShort
			}

			if profileName != targetProfile {
				continue
			}
		}

		if len(profileInfo.Instances) > 0 {
			hasInstances = true
			fmt.Printf("\nInstances for profile '%s':\n", profileName)
			for name, instance := range profileInfo.Instances {
				defaultMarker := ""

				if profileInfo.DefaultInstance == name {
					defaultMarker = " (default)"
				}

				fmt.Printf("\n  Name:    %s%s\n", name, defaultMarker)
				fmt.Printf("  ID:      %s\n", instance.ID)
				fmt.Printf("  Profile: %s\n", instance.Profile)
				fmt.Printf("  Host:    %s\n", instance.Host)
			}
		}
	}

	if !hasInstances {
		fmt.Println("\nNo instances configured.")
	}

	fmt.Println()

	return nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func addInstance(args []string, config *Configuration) error {
	flagSet := flag.NewFlagSet("instances add", flag.ExitOnError)
	profile := flagSet.String("profile", "", "--profile <aws cli profile>")
	profileShort := flagSet.String("p", "", "--profile <aws cli profile>")
	instanceName := flagSet.String("name", "", "--name <instance name>")
	instanceNameShort := flagSet.String("n", "", "--name <instance name>")

	flagSet.Usage = func() {
		fmt.Println("USAGE:\n    awsutil instances add [--profile <aws cli profile>] [--name <instance name>] <filter string>")
	}

	if err := flagSet.Parse(args); err != nil {
		flagSet.Usage()
		return fmt.Errorf("failed to parse options")
	}

	if len(flagSet.Args()) == 0 {
		flagSet.Usage()
		return fmt.Errorf("must specify instance filter string")
	}

	filter := flagSet.Args()[0]
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

	if profileInfo.Instances == nil {
		profileInfo.Instances = make(map[string]Instance)
	}

	reader := bufio.NewReader(os.Stdin)

	// Query EC2 instances
	fmt.Println("\nQuerying EC2 instances...")
	instances, err := queryEC2Instances(currentProfile, filter)
	if err != nil {
		return fmt.Errorf("failed to query EC2 instances: %v", err)
	}

	if len(instances) == 0 {
		return fmt.Errorf("no EC2 instances found matching filter '%s'", filter)
	}

	// Display instances and let user select
	fmt.Println("\nAvailable EC2 instances:")
	for i, inst := range instances {
		fmt.Printf("  %2d: %s - %s\n", i+1, inst.Instance, inst.Name)
	}

	fmt.Print("\nSelect instance number: ")
	instSelection, _ := reader.ReadString('\n')
	instIndex, err := strconv.Atoi(strings.TrimSpace(instSelection))

	if err != nil || instIndex < 1 || instIndex > len(instances) {
		return fmt.Errorf("invalid selection")
	}

	selectedInstance := instances[instIndex-1]

	// Get instance name
	var targetInstanceName string
	if *instanceName != "" {
		targetInstanceName = *instanceName
	} else if *instanceNameShort != "" {
		targetInstanceName = *instanceNameShort
	} else {
		fmt.Print("\nEnter instance name: ")
		nameInput, _ := reader.ReadString('\n')
		targetInstanceName = strings.TrimSpace(nameInput)

		if targetInstanceName == "" {
			// Generate a default name from instance name
			targetInstanceName = selectedInstance.Name
			if targetInstanceName == "" {
				targetInstanceName = fmt.Sprintf("instance-%d", len(profileInfo.Instances)+1)
			}
		}
	}

	// Check if name already exists
	if _, exists := profileInfo.Instances[targetInstanceName]; exists {
		return fmt.Errorf("instance '%s' already exists in profile '%s'", targetInstanceName, currentProfile)
	}

	// Get host (private IP)
	host := selectedInstance.Name
	if host == "" {
		// Fallback to instance ID if no private IP available
		host = selectedInstance.Instance
	}

	// Create instance configuration
	newInstance := Instance{
		Name:    targetInstanceName,
		ID:      selectedInstance.Instance,
		Profile: currentProfile,
		Host:    host,
	}

	// Save to configuration
	profileInfo.Instances[targetInstanceName] = newInstance
	profileInfo.Name = currentProfile
	config.Profiles[currentProfile] = profileInfo

	fmt.Printf("\nInstance '%s' (ID: %s) added successfully!\n", targetInstanceName, selectedInstance.Instance)

	return nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func removeInstance(args []string, config *Configuration) error {
	flagSet := flag.NewFlagSet("instances remove", flag.ExitOnError)
	profile := flagSet.String("profile", "", "--profile <aws cli profile>")
	profileShort := flagSet.String("p", "", "--profile <aws cli profile>")
	instanceName := flagSet.String("name", "", "--name <instance name>")
	instanceNameShort := flagSet.String("n", "", "--name <instance name>")

	flagSet.Usage = func() {
		fmt.Println("USAGE:\n    awsutil instances remove [--profile <aws cli profile>] [--name <instance name>]")
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

	if len(profileInfo.Instances) == 0 {
		return fmt.Errorf("no instances configured for profile '%s'", currentProfile)
	}

	reader := bufio.NewReader(os.Stdin)

	// Get instance name
	var targetInstanceName string

	if *instanceName != "" {
		targetInstanceName = *instanceName
	} else if *instanceNameShort != "" {
		targetInstanceName = *instanceNameShort
	} else {
		// Prompt for instance name
		fmt.Print("Enter instance name to remove: ")
		nameInput, _ := reader.ReadString('\n')
		targetInstanceName = strings.TrimSpace(nameInput)

		if targetInstanceName == "" {
			return fmt.Errorf("instance name is required")
		}
	}

	// Check if instance exists
	existingInstance, exists := profileInfo.Instances[targetInstanceName]
	if !exists {
		return fmt.Errorf("instance '%s' not found in profile '%s'", targetInstanceName, currentProfile)
	}

	// Display instance information
	fmt.Printf("\nInstance to remove:\n")
	fmt.Printf("  Name:    %s\n", targetInstanceName)
	fmt.Printf("  ID:      %s\n", existingInstance.ID)
	fmt.Printf("  Profile: %s\n", existingInstance.Profile)
	fmt.Printf("  Host:    %s\n", existingInstance.Host)

	// Ask for confirmation
	fmt.Print("\nAre you sure you want to remove this instance? (yes/no): ")
	confirmation, _ := reader.ReadString('\n')
	confirmation = strings.TrimSpace(strings.ToLower(confirmation))

	if confirmation != "yes" && confirmation != "y" {
		fmt.Println("Removal cancelled.")
		return nil
	}

	// Remove from Instances map
	delete(profileInfo.Instances, targetInstanceName)

	// If this was the default instance, clear the DefaultInstance field
	if profileInfo.DefaultInstance == targetInstanceName {
		profileInfo.DefaultInstance = ""
	}

	// Update profile in config
	profileInfo.Name = currentProfile
	config.Profiles[currentProfile] = profileInfo

	fmt.Printf("\nInstance '%s' removed successfully!\n", targetInstanceName)

	return nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
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
	if len(currentProfile) != 0 && currentProfile != config.DefaultProfile {
		config.DefaultProfile = currentProfile

		// Ensure profile exists in map
		if _, exists := config.Profiles[currentProfile]; !exists {
			config.Profiles[currentProfile] = Profile{
				Name:      currentProfile,
				Bastions:  make(map[string]Bastion),
				Instances: make(map[string]Instance),
			}
		}
	} else if len(config.DefaultProfile) == 0 {
		return "", fmt.Errorf("must specify the target profile")
	}

	return currentProfile, nil
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func selectInstanceByName(profileInfo Profile, name string) (Instance, error) {
	if len(profileInfo.Instances) == 0 {
		return Instance{}, fmt.Errorf("no instances configured for this profile")
	}

	// If name is provided, use it
	if name != "" {
		if instance, exists := profileInfo.Instances[name]; exists {
			return instance, nil
		}

		return Instance{}, fmt.Errorf("instance '%s' not found", name)
	}

	// If no name provided, try default
	if profileInfo.DefaultInstance != "" {
		if instance, exists := profileInfo.Instances[profileInfo.DefaultInstance]; exists {
			return instance, nil
		}
	}

	// If only one instance exists, use it
	if len(profileInfo.Instances) == 1 {
		for _, instance := range profileInfo.Instances {
			return instance, nil
		}
	}

	// Multiple instances exist, need to specify name
	return Instance{}, fmt.Errorf("multiple instances available, please specify instance name")
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func selectInstanceByHost(profileInfo Profile, host string) (Instance, error) {
	if len(profileInfo.Instances) == 0 {
		return Instance{}, fmt.Errorf("no instances configured for this profile")
	}

	// Search for instance with matching host
	for _, instance := range profileInfo.Instances {
		if instance.Host == host {
			return instance, nil
		}
	}

	return Instance{}, fmt.Errorf("instance with host '%s' not found", host)
}
