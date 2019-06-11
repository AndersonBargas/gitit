package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var currentConfig config

type config struct {
	General struct {
		CheckIntervalSeconds uint   `json:"checkIntervalSeconds"`
		ContextPath          string `json:"contextPath"`
	} `json:"general"`
	Git struct {
		BinaryPath                     string `json:"binaryPath"`
		ConsecutiveGitErrorsBeforeStop uint   `json:"consecutiveGitErrorsBeforeStop"`
		LocalCommandsTimeout           uint   `json:"localCommandsTimeout"`
		OriginCommandsTimeout          uint   `json:"originCommandsTimeout"`
		ResetBeforePull                bool   `json:"resetBeforePull"`
	} `json:"git"`
	Rebuild struct {
		ConsecutiveBuildErrorsBeforeStop uint `json:"consecutiveBuildErrorsBeforeStop"`
		Commands                         []struct {
			Command string `json:"command"`
			Timeout uint   `json:"timeout"`
		} `json:"commands"`
	} `json:"rebuild"`
}

func main() {
	/** Flags **/
	generateConfigFile := flag.Bool("gc", false, "generates a config file and exit")
	flag.Parse()

	/** Configuration **/
	const configFilePath = "gitit.json"
	if *generateConfigFile == true {
		currentConfig = newDefaultConfig()
		err := saveConfigFile(configFilePath)
		log.Println("Exiting...")
		if err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}
	cfg, err := loadConfigFromFile(configFilePath)
	if err != nil {
		cfg = newDefaultConfig()
	}
	currentConfig = cfg

	/** Git **/
	branchName := getBranchName()
	log.Print("Checked on branch ", branchName)

	interval := time.Duration(currentConfig.General.CheckIntervalSeconds)
	ticker := time.NewTicker(interval * time.Second)
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	signal.Notify(quit, syscall.SIGTERM)

	for {
		select {
		/** Loop **/
		case <-ticker.C:
			localHash := getLocalHash()
			remoteHash := getRemoteHash()
			if localHash == remoteHash {
				log.Print("Same hash, nothing changed...")
				continue
			}
			log.Println("Hash changed.")
			if currentConfig.Git.ResetBeforePull {
				resetLocalBranch()
			}
			updateLocalBranch()
			rebuild()
		/** Gracefully shutdown **/
		case <-quit:
			ticker.Stop()
			fmt.Print("\r")
			log.Print("Git-it stopped.", "Exiting...")
			os.Exit(0)
			return
		}
	}
}

func execGitCommand(args ...string) (string, error) {
	gitCMD := exec.Command("git", args...)
	cmdOUT, err := gitCMD.Output()
	if err != nil {
		return "", err
	}
	outputSTR := strings.TrimSuffix(string(cmdOUT), "\n")
	return outputSTR, nil
}

func getBranchName() string {
	branchName, err := execGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		log.Fatalf("Error obtaining the branch name")
	}
	return branchName
}

func getLocalHash() string {
	localHash, err := execGitCommand("rev-parse", "HEAD")
	if err != nil {
		log.Fatalf("Error obtaining the local hash")
	}
	return localHash
}

func getRemoteHash() string {
	remoteHash, err := execGitCommand("rev-parse", "master@{upstream}")
	if err != nil {
		log.Fatalf("Error obtaining the remote hash")
	}
	return remoteHash
}

func loadConfigFromFile(filePath string) (config, error) {
	loadedConfig := config{}
	log.Printf("Trying to load config from file...\n")
	configResource, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error loading the configuration file\n")
		return loadedConfig, err
	}
	defer configResource.Close()
	log.Printf("Parsing the config loaded from file...\n")
	jsonDecoder := json.NewDecoder(configResource)
	err = jsonDecoder.Decode(&loadedConfig)
	if err != nil {
		log.Printf("Error parsing the configuration file")
		return loadedConfig, err
	}
	log.Printf("Config succesfully loaded from file")
	return loadedConfig, nil
}

func newDefaultConfig() config {
	log.Printf("Initializing with default configuration")
	defaultConfig := config{}
	defaultJSON := `{
		"general": {
			"checkIntervalSeconds": 10,
			"contextPath": "."
		},
		"git": {
			"binaryPath": "",
			"consecutiveGitErrorsBeforeStop": 3,
			"localCommandsTimeout": 2,
			"originCommandsTimeout": 10,
			"resetBeforePull": true
		},
		"rebuild": {
			"consecutiveBuildErrorsBeforeStop": 5,
			"commands": [
				{
					"command": "go version",
					"timeout": 0
				}
			]
		}
	}`
	json.Unmarshal([]byte(defaultJSON), &defaultConfig)
	return defaultConfig
}

func rebuild() {
	log.Println("Rebuilding...")
	commands := currentConfig.Rebuild.Commands
	for _, cmd := range commands {
		// buildCMD := exec.Command(cmd.Command)
		// out, _ = buildCMD.Output()
		out, err := exec.Command(cmd.Command).Output()
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Build output: %s\n", out)
	}

}

func resetLocalBranch() {
	log.Println("Reseting local branch...")
	execGitCommand("reset", "--hard")
	log.Println("Local branch succesfully reseted")
}

func saveConfigFile(filePath string) error {
	configResource, err := json.MarshalIndent(currentConfig, "", "    ")
	if err != nil {
		log.Println("Error while encoding the configuration to be saved in a file")
		return err
	}

	err = ioutil.WriteFile(filePath, configResource, 0644)
	if err != nil {
		log.Println("Error while saving the configuration file")
		return err
	}
	log.Println("Configuration file succesfully generated")
	return nil
}

func updateLocalBranch() {
	log.Println("Updating local branch...")
	execGitCommand("pull")
}
