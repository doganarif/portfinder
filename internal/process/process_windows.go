//go:build windows

package process

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type platformFinder struct{}

func (f *platformFinder) FindByPort(port int) (*Process, error) {
	// Use netstat on Windows to find process by port
	cmd := exec.Command("netstat", "-ano", "-p", "tcp")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("netstat failed: %w", err)
	}

	pid := f.findPIDByPort(string(output), port)
	if pid == 0 {
		return nil, nil // Port not in use
	}

	// Get process details
	return f.getProcessDetails(pid, port)
}

func (f *platformFinder) ListAll() ([]*Process, error) {
	cmd := exec.Command("netstat", "-ano", "-p", "tcp")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("netstat failed: %w", err)
	}

	return f.parseNetstatOutput(string(output))
}

func (f *platformFinder) findPIDByPort(output string, port int) int {
	lines := strings.Split(output, "\n")
	portPattern := fmt.Sprintf(`:%d\s+`, port)
	re := regexp.MustCompile(portPattern)

	for _, line := range lines {
		if !strings.Contains(line, "LISTENING") {
			continue
		}

		if re.MatchString(line) {
			// Extract PID from the end of the line
			fields := strings.Fields(line)
			if len(fields) >= 5 {
				pid, err := strconv.Atoi(fields[len(fields)-1])
				if err == nil {
					return pid
				}
			}
		}
	}

	return 0
}

func (f *platformFinder) parseNetstatOutput(output string) ([]*Process, error) {
	lines := strings.Split(output, "\n")
	processMap := make(map[string]*Process)

	// Regex to match port number in address (e.g., 0.0.0.0:3000 or 127.0.0.1:8080)
	portRegex := regexp.MustCompile(`:(\d+)\s+`)

	for _, line := range lines {
		if !strings.Contains(line, "LISTENING") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// Extract port from local address
		matches := portRegex.FindStringSubmatch(fields[1])
		if len(matches) < 2 {
			continue
		}

		port, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		// Extract PID
		pid, err := strconv.Atoi(fields[len(fields)-1])
		if err != nil || pid == 0 {
			continue
		}

		key := fmt.Sprintf("%d-%d", pid, port)
		if _, exists := processMap[key]; exists {
			continue
		}

		proc, err := f.getProcessDetails(pid, port)
		if err != nil || proc == nil {
			continue
		}

		processMap[key] = proc
	}

	processes := make([]*Process, 0, len(processMap))
	for _, p := range processMap {
		processes = append(processes, p)
	}

	return processes, nil
}

func (f *platformFinder) getProcessDetails(pid int, port int) (*Process, error) {
	if pid == 0 {
		return nil, nil
	}

	proc := &Process{
		PID:  pid,
		Port: port,
	}

	// Get process name and details using tasklist
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/V")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("tasklist failed: %w", err)
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("no process found for PID %d", pid)
	}

	// Parse CSV output
	// Header: "Image Name","PID","Session Name","Session#","Mem Usage","Status","User Name","CPU Time","Window Title"
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		fields := f.parseCSVLine(line)
		if len(fields) >= 9 {
			proc.Name = strings.Trim(fields[0], "\"")

			// Try to get command line using wmic
			f.enrichProcessInfo(proc)

			return proc, nil
		}
	}

	return nil, fmt.Errorf("could not parse process details for PID %d", pid)
}

func (f *platformFinder) parseCSVLine(line string) []string {
	var fields []string
	var current strings.Builder
	inQuotes := false

	for _, char := range line {
		switch char {
		case '"':
			inQuotes = !inQuotes
			current.WriteRune(char)
		case ',':
			if inQuotes {
				current.WriteRune(char)
			} else {
				fields = append(fields, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		fields = append(fields, current.String())
	}

	return fields
}

func (f *platformFinder) enrichProcessInfo(proc *Process) {
	// Get command line using wmic
	cmd := exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", proc.PID), "get", "CommandLine", "/format:list")
	output, err := cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "CommandLine=") {
				proc.Command = strings.TrimPrefix(line, "CommandLine=")
				proc.Command = strings.TrimSpace(proc.Command)
				break
			}
		}
	}

	// Get process start time
	cmd = exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", proc.PID), "get", "CreationDate", "/format:list")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "CreationDate=") {
				dateStr := strings.TrimPrefix(line, "CreationDate=")
				dateStr = strings.TrimSpace(dateStr)
				// Parse WMI datetime format: 20231228103045.123456+060
				if len(dateStr) >= 14 {
					year, _ := strconv.Atoi(dateStr[0:4])
					month, _ := strconv.Atoi(dateStr[4:6])
					day, _ := strconv.Atoi(dateStr[6:8])
					hour, _ := strconv.Atoi(dateStr[8:10])
					minute, _ := strconv.Atoi(dateStr[10:12])
					second, _ := strconv.Atoi(dateStr[12:14])

					proc.StartTime = time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
				}
				break
			}
		}
	}

	// If start time is not set, use current time as fallback
	if proc.StartTime.IsZero() {
		proc.StartTime = time.Now()
	}

	// Get working directory (more complex on Windows, using current directory as fallback)
	cmd = exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", proc.PID), "get", "ExecutablePath", "/format:list")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "ExecutablePath=") {
				exePath := strings.TrimPrefix(line, "ExecutablePath=")
				exePath = strings.TrimSpace(exePath)
				if exePath != "" {
					proc.ProjectPath = detectProject(proc.PID, exePath)
				}
				break
			}
		}
	}

	// If project path is still empty, try to detect from command
	if proc.ProjectPath == "" && proc.Command != "" {
		// Extract potential path from command
		parts := strings.Fields(proc.Command)
		for _, part := range parts {
			if strings.Contains(part, "\\") || strings.Contains(part, "/") {
				proc.ProjectPath = detectProject(proc.PID, part)
				if proc.ProjectPath != "" && proc.ProjectPath != "unknown" {
					break
				}
			}
		}
	}

	// Simple Docker detection on Windows
	if strings.Contains(strings.ToLower(proc.Name), "docker") ||
		strings.Contains(strings.ToLower(proc.Command), "docker") {
		proc.IsDocker = true
	}
}
