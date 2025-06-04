package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/doganarif/portfinder/internal/config"
	"github.com/doganarif/portfinder/internal/process"
	"github.com/doganarif/portfinder/internal/ui"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "portfinder [port]",
		Short: "Find and manage processes using network ports",
		Long: `portfinder helps you identify what's using your ports and take action.
        
Examples:
  portfinder 3000           # Check what's using port 3000
  portfinder check          # Check common development ports
  portfinder list           # List all active ports
  portfinder kill 3000      # Kill process using port 3000`,
		Args: cobra.MaximumNArgs(1),
		Run:  runPortCheck,
	}

	var checkCmd = &cobra.Command{
		Use:   "check",
		Short: "Check common development ports",
		Run:   runCheckCommon,
	}

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List all ports in use",
		Run:   runListAll,
	}

	var killCmd = &cobra.Command{
		Use:   "kill [port]",
		Short: "Kill process using specified port",
		Args:  cobra.ExactArgs(1),
		Run:   runKillProcess,
	}

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("portfinder %s (%s) built at %s\n", version, commit, date)
		},
	}

	rootCmd.AddCommand(checkCmd, listCmd, killCmd, versionCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runPortCheck(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		return
	}

	port, err := strconv.Atoi(args[0])
	if err != nil {
		ui.ErrorMsg("Invalid port number: %s", args[0])
		os.Exit(1)
	}

	finder := process.NewFinder()
	proc, err := finder.FindByPort(port)
	if err != nil {
		ui.ErrorMsg("Error checking port: %v", err)
		os.Exit(1)
	}

	if proc == nil {
		ui.SuccessMsg("Port %d is free!", port)
		return
	}

	ui.ShowProcessDetail(proc, true)
}

func runCheckCommon(cmd *cobra.Command, args []string) {
	cfg := config.Load()
	finder := process.NewFinder()

	results := make(map[int]*process.Process)
	for _, port := range cfg.CommonPorts {
		proc, _ := finder.FindByPort(port)
		results[port] = proc
	}

	if err := ui.ShowPortCheck(results); err != nil {
		ui.ErrorMsg("Error: %v", err)
		os.Exit(1)
	}
}

func runListAll(cmd *cobra.Command, args []string) {
	finder := process.NewFinder()
	processes, err := finder.ListAll()
	if err != nil {
		ui.ErrorMsg("Error listing ports: %v", err)
		os.Exit(1)
	}

	if err := ui.ShowProcessList(processes); err != nil {
		ui.ErrorMsg("Error: %v", err)
		os.Exit(1)
	}
}

func runKillProcess(cmd *cobra.Command, args []string) {
	port, err := strconv.Atoi(args[0])
	if err != nil {
		ui.ErrorMsg("Invalid port number: %s", args[0])
		os.Exit(1)
	}

	finder := process.NewFinder()
	proc, err := finder.FindByPort(port)
	if err != nil {
		ui.ErrorMsg("Error checking port: %v", err)
		os.Exit(1)
	}

	if proc == nil {
		ui.InfoMsg("Port %d is not in use", port)
		return
	}

	if err := proc.Kill(); err != nil {
		ui.ErrorMsg("Failed to kill process: %v", err)
		os.Exit(1)
	}

	ui.SuccessMsg("Killed process %s (PID: %d) on port %d", proc.Name, proc.PID, port)
}
