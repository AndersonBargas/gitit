package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	branchName := getBranchName()
	fmt.Println("Checked on", branchName)

	ticker := time.NewTicker(10 * time.Second)
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)
	signal.Notify(quit, syscall.SIGTERM)

	for {
		select {
		case <-ticker.C:
			localHash := getLocalHash()
			remoteHash := getRemoteHash()
			if localHash != remoteHash {
				updateLocalBranch()
				rebuild()

			}
		case <-quit:
			ticker.Stop()
			fmt.Print("\r")
			fmt.Print("Git-it stopped.", "Exiting...")
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

func rebuild() {
	fmt.Println("Hash changed.", "Buiding...")
	buildCMD := exec.Command("go", "build")
	_, _ = buildCMD.Output()
}

func updateLocalBranch() {
	execGitCommand("pull")
}