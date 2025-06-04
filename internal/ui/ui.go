package ui

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/doganarif/portfinder/internal/process"
	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/olekukonko/tablewriter"
)

var (
	successColor = color.New(color.FgGreen)
	errorColor   = color.New(color.FgRed)
	infoColor    = color.New(color.FgCyan)
	warnColor    = color.New(color.FgYellow)
)

// SuccessMsg prints a success message
func SuccessMsg(format string, args ...interface{}) {
	successColor.Printf("✅ "+format+"\n", args...)
}

// ErrorMsg prints an error message
func ErrorMsg(format string, args ...interface{}) {
	errorColor.Printf("❌ "+format+"\n", args...)
}

// InfoMsg prints an info message
func InfoMsg(format string, args ...interface{}) {
	infoColor.Printf("ℹ️  "+format+"\n", args...)
}

// WarnMsg prints a warning message
func WarnMsg(format string, args ...interface{}) {
	warnColor.Printf("⚠️  "+format+"\n", args...)
}

// DisplayProcess displays detailed information about a process
func DisplayProcess(p *process.Process) {
	fmt.Println()
	errorColor.Printf("🔍 Port %d is in use by:\n", p.Port)
	fmt.Println()

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Property", "Value"})
	table.SetBorder(false)
	table.SetColumnSeparator("")
	table.SetHeaderLine(false)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	data := [][]string{
		{"Process", p.Name},
		{"PID", fmt.Sprintf("%d", p.PID)},
		{"Command", truncateCommand(p.Command)},
		{"Project", formatProject(p.ProjectPath)},
		{"Started", formatDuration(time.Since(p.StartTime)) + " ago"},
	}

	if p.IsDocker {
		data = append(data, []string{"Docker", fmt.Sprintf("Yes (Container: %s)", p.DockerID)})
	}

	table.AppendBulk(data)
	table.Render()
	fmt.Println()
}

// DisplayPortSummary displays a summary of common ports
func DisplayPortSummary(ports map[int]*process.Process) {
	fmt.Println()
	infoColor.Println("📊 Common Development Ports:")
	fmt.Println()

	// Group ports by category
	categories := map[string][]int{
		"Frontend":  {3000, 3001, 4200, 5173, 8080},
		"Backend":   {4000, 5000, 8000, 9000},
		"Databases": {3306, 5432, 6379, 27017},
		"Tools":     {9200, 9090, 3100, 8983},
	}

	for category, categoryPorts := range categories {
		fmt.Printf("\n%s:\n", category)
		for _, port := range categoryPorts {
			if proc, exists := ports[port]; exists {
				if proc != nil {
					errorColor.Printf("  ❌ %d: %s", port, proc.Name)
					if proc.ProjectPath != "" && proc.ProjectPath != "unknown" {
						fmt.Printf(" (%s)", proc.ProjectPath)
					}
					fmt.Println()
				} else {
					successColor.Printf("  ✅ %d: free\n", port)
				}
			}
		}
	}
}

// DisplayProcessList displays a list of all processes
func DisplayProcessList(processes []*process.Process) {
	if len(processes) == 0 {
		InfoMsg("No processes are using network ports")
		return
	}

	fmt.Println()
	infoColor.Printf("📋 Found %d processes using network ports:\n", len(processes))
	fmt.Println()

	// Sort by port number
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].Port < processes[j].Port
	})

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Port", "Process", "PID", "Project", "Running For"})
	table.SetBorder(false)
	table.SetHeaderLine(true)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, p := range processes {
		table.Append([]string{
			fmt.Sprintf("%d", p.Port),
			p.Name,
			fmt.Sprintf("%d", p.PID),
			formatProject(p.ProjectPath),
			formatDuration(time.Since(p.StartTime)),
		})
	}

	table.Render()
}

// ConfirmKill asks for confirmation before killing a process
func ConfirmKill() bool {
	prompt := promptui.Select{
		Label: "Kill this process?",
		Items: []string{"Yes", "No"},
	}

	_, result, err := prompt.Run()
	if err != nil {
		return false
	}

	return result == "Yes"
}

// SimpleConfirm asks a yes/no question without external dependencies
func SimpleConfirm(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s [y/n]: ", question)
		response, err := reader.ReadString('\n')
		if err != nil {
			return false
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}

// Helper functions

func truncateCommand(cmd string) string {
	if len(cmd) > 60 {
		return cmd[:57] + "..."
	}
	return cmd
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "< 1 minute"
	} else if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%d hours", int(d.Hours()))
	} else {
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	}
}
