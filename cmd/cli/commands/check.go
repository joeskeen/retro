package commands

import (
	"fmt"
	"os/exec"
)

func RunCheck() error {
	fmt.Println("Platform readiness check:")
	fmt.Println()

	platforms := []struct {
		name    string
		check   func() bool
		install string
	}{
		{"git", checkGit, "sudo apt install git"},
		{"git-lfs", checkGitLFS, "git lfs install"},
		{"dosbox", checkDOSBox, "sudo apt install dosbox"},
		{"dosbox-x", checkDOSBoxX, "sudo apt install dosbox-x"},
		{"wine", checkWine, "sudo apt install wine"},
	}

	for _, p := range platforms {
		status := "✗ not found"
		if p.check() {
			status = "✓ ready"
		}
		fmt.Printf("  %s: %s\n", p.name, status)
	}

	fmt.Println()
	fmt.Println("Git and Git LFS are required for pushing/pulling game images:")
	fmt.Println("  git:      https://git-scm.com")
	fmt.Println("  git lfs:  https://git-lfs.github.com")
	fmt.Println("  Install: git lfs install")
	fmt.Println()
	fmt.Println("For DOS games, install DOSBox to get started:")
	fmt.Println("  sudo apt install dosbox")
	fmt.Println()
	fmt.Println("Run 'retro run <game>' to play")

	return nil
}

func checkGit() bool {
	cmd := exec.Command("git", "version")
	return cmd.Run() == nil
}

func checkGitLFS() bool {
	cmd := exec.Command("git", "lfs", "version")
	return cmd.Run() == nil
}

func checkDOSBox() bool {
	cmd := exec.Command("dosbox", "-version")
	return cmd.Run() == nil
}

func checkDOSBoxX() bool {
	cmd := exec.Command("dosbox-x", "-version")
	return cmd.Run() == nil
}

func checkWine() bool {
	cmd := exec.Command("wine", "--version")
	return cmd.Run() == nil
}
