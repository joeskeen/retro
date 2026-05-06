package main

import (
	"fmt"
	"os"

	"retrogame/cmd/cli/commands"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "build":
		err = commands.RunBuild()
	case "run":
		err = commands.RunRun()
	case "images":
		err = commands.RunImages()
	case "push":
		err = commands.RunPush()
	case "pull":
		err = commands.RunPull()
	case "clone":
		err = commands.RunClone()
	case "check":
		err = commands.RunCheck()
	case "rm":
		err = commands.RunRm()
	case "prune":
		err = commands.RunPrune()
	case "remote":
		err = runRemote()
	case "help":
		printUsage()
		os.Exit(0)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runRemote() error {
	if len(os.Args) < 3 {
		printRemoteUsage()
		os.Exit(1)
	}

	var err error
	switch os.Args[2] {
	case "add":
		err = commands.RunRemoteAdd()
	case "list":
		err = commands.RunRemoteList()
	case "remove":
		err = commands.RunRemoteRemove()
	case "default":
		err = commands.RunRemoteDefault()
	default:
		fmt.Fprintf(os.Stderr, "Unknown remote command: %s\n", os.Args[2])
		printRemoteUsage()
		os.Exit(1)
	}
	return err
}

func printUsage() {
	fmt.Print(`RetroGame CLI - Docker-like game image system

Usage:
  retro <command> [options]

Commands:
  build <path>       Build an image from a Retrofile
  run <image>        Run an image (format: name:tag)
  rm <image>         Remove a local image
  clone <git-url>    Clone a registry from git
  push <image>       Push image to a remote registry
  pull <image>       Pull image from a remote registry
  check              Check platform readiness (DOSBox, Wine, etc.)
  images             List available images
  remote             Manage remote registries

Run 'retro help' for more details.
`)
}

func printRemoteUsage() {
	fmt.Print(`Usage:
  retro remote add <name> <url>     Add a remote registry
  retro remote list                 List configured remotes
  retro remote remove <name>        Remove a remote
  retro remote default <name>       Set default remote
`)
}
