package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Configuration struct {
	DefaultProfile string                   `json:"defaultProfile,omitempty"`
	Profiles       map[string]Profile       `json:"profiles,omitempty"`
	BastionLookup  map[string]BastionLookup `json:"-"` // Map of bastion ID to profile and name
}

type BastionLookup struct {
	Profile string `json:"profile,omitempty"`
	Name    string `json:"name,omitempty"`
}

type Profile struct {
	Name            string              `json:"name,omitempty"`
	Instance        string              `json:"instance,omitempty"`        // Deprecated: use DefaultInstance instead
	DefaultInstance string              `json:"defaultInstance,omitempty"` // Default instance name
	Bastion         Bastion             `json:"bastion,omitempty"`         // Deprecated: use Bastions instead
	Bastions        map[string]Bastion  `json:"bastions,omitempty"`        // New: multiple named bastions
	DefaultBastion  string              `json:"defaultBastion,omitempty"`  // Default bastion name
	Instances       map[string]Instance `json:"instances,omitempty"`       // Named EC2 instances
}

type Instance struct {
	Name    string `json:"name,omitempty"`
	ID      string `json:"id,omitempty"`
	Profile string `json:"profile,omitempty"`
	Host    string `json:"host,omitempty"`
}

type Bastion struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Profile   string `json:"profile,omitempty"`
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
	Host     string `json:"Host"`
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

	// Initialize BastionLookup if nil
	if config.BastionLookup == nil {
		config.BastionLookup = make(map[string]BastionLookup)
	}

	// Populate Profile field in each bastion and build ID lookup map
	if config.Profiles != nil {
		for profileName, profile := range config.Profiles {
			// Initialize Instances map if nil
			if profile.Instances == nil {
				profile.Instances = make(map[string]Instance)
			}

			// Populate Profile field in each instance
			for instanceName, instance := range profile.Instances {
				// Set Profile field if not already set
				if instance.Profile == "" {
					instance.Profile = profileName
				}
				// Update instance in profile
				profile.Instances[instanceName] = instance
			}

			if profile.Bastions != nil {
				for bastionName, bastion := range profile.Bastions {
					// Set Profile field if not already set
					if bastion.Profile == "" {
						bastion.Profile = profileName
					}

					// Generate ID if not present
					if bastion.ID == "" {
						newID, err := generateBastionID()

						if err != nil {
							return Configuration{}, fmt.Errorf("failed to generate bastion ID: %v", err)
						}

						bastion.ID = newID
					}

					// Add to lookup map
					config.BastionLookup[bastion.ID] = BastionLookup{
						Profile: profileName,
						Name:    bastionName,
					}

					// Update bastion in profile
					profile.Bastions[bastionName] = bastion
				}
			}

			config.Profiles[profileName] = profile
		}
	}

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

		// Initialize Instances map if nil
		if profile.Instances == nil {
			profile.Instances = make(map[string]Instance)
		}

		// Migrate old Instance field to DefaultInstance if needed
		if profile.Instance != "" && profile.DefaultInstance == "" {
			// Create a "default" entry in Instances map
			profile.Instances["default"] = Instance{
				Name:    "default",
				ID:      profile.Instance,
				Profile: profileName,
				Host:    profile.Instance, // Placeholder - could be improved
			}
			profile.DefaultInstance = "default"
		}

		config.Profiles[profileName] = profile
	}
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func saveConfiguration(fileName string, config *Configuration) {
	// Rebuild BastionLookup map before saving
	rebuildBastionLookup(config)

	// Save the configuration file
	configBytes, _ := json.MarshalIndent(config, "", "    ")
	os.WriteFile(fileName, configBytes, 0644)
}

// - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -
func rebuildBastionLookup(config *Configuration) {
	// Initialize lookup map if nil
	if config.BastionLookup == nil {
		config.BastionLookup = make(map[string]BastionLookup)
	}

	// Clear existing lookup
	config.BastionLookup = make(map[string]BastionLookup)

	// Rebuild lookup from all profiles
	if config.Profiles != nil {
		for profileName, profile := range config.Profiles {
			if profile.Bastions != nil {
				for bastionName, bastion := range profile.Bastions {
					// Ensure Profile field is set
					if bastion.Profile == "" {
						bastion.Profile = profileName
					}

					// Generate ID if not present
					if bastion.ID == "" {
						newID, err := generateBastionID()

						if err == nil {
							bastion.ID = newID
						}
					}

					// Add to lookup map
					if bastion.ID != "" {
						config.BastionLookup[bastion.ID] = BastionLookup{
							Profile: profileName,
							Name:    bastionName,
						}
					}

					// Update bastion in profile
					profile.Bastions[bastionName] = bastion
				}
			}

			config.Profiles[profileName] = profile
		}
	}
}
