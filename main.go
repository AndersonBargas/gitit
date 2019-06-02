package main

import (
	"fmt"
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

	//go func() {
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
	//}()
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
	branchName, _ := execGitCommand("rev-parse", "--abbrev-ref", "HEAD")
	return branchName
}

func getLocalHash() string {
	localHash, _ := execGitCommand("rev-parse", "HEAD")
	return localHash
}

func getRemoteHash() string {
	remoteHash, _ := execGitCommand("rev-parse", "master@{upstream}")
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
