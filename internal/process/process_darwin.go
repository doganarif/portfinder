//go:build darwin

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
	// Use lsof on macOS
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-n", "-P")
	output, err := cmd.Output()
	if err != nil {
		// No process found is not an error
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("lsof failed: %w", err)
	}

	return f.parseLsofOutput(string(output), port)
}

func (f *platformFinder) ListAll() ([]*Process, error) {
	cmd := exec.Command("lsof", "-i", "-n", "-P")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lsof failed: %w", err)
	}

	return f.parseLsofOutputMultiple(string(output))
}

func (f *platformFinder) parseLsofOutput(output string, port int) (*Process, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return nil, nil
	}

	// Skip header
	for i := 1; i < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if len(fields) < 9 {
			continue
		}

		// Check if it's a LISTEN state
		if !strings.Contains(lines[i], "LISTEN") {
			continue
		}

		proc := &Process{
			Name: fields[0],
			Port: port,
		}

		// Parse PID
		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		proc.PID = pid

		// Get additional process info
		f.enrichProcessInfo(proc)

		return proc, nil
	}

	return nil, nil
}

func (f *platformFinder) parseLsofOutputMultiple(output string) ([]*Process, error) {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return nil, nil
	}

	portRegex := regexp.MustCompile(`:(\d+)\s+\(LISTEN\)`)
	processMap := make(map[string]*Process)

	for i := 1; i < len(lines); i++ {
		fields := strings.Fields(lines[i])
		if len(fields) < 9 {
			continue
		}

		if !strings.Contains(lines[i], "LISTEN") {
			continue
		}

		matches := portRegex.FindStringSubmatch(lines[i])
		if len(matches) < 2 {
			continue
		}

		port, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		key := fmt.Sprintf("%d-%d", pid, port)
		if _, exists := processMap[key]; exists {
			continue
		}

		proc := &Process{
			Name: fields[0],
			PID:  pid,
			Port: port,
		}

		f.enrichProcessInfo(proc)
		processMap[key] = proc
	}

	processes := make([]*Process, 0, len(processMap))
	for _, p := range processMap {
		processes = append(processes, p)
	}

	return processes, nil
}

func (f *platformFinder) enrichProcessInfo(proc *Process) {
	// Get process info using ps
	cmd := exec.Command("ps", "-p", strconv.Itoa(proc.PID), "-o", "comm=,command=")
	output, err := cmd.Output()
	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 {
		parts := strings.SplitN(lines[0], " ", 2)
		if len(parts) > 1 {
			proc.Command = strings.TrimSpace(parts[1])
		}
	}

	// Get process start time properly on macOS
	cmd = exec.Command("ps", "-p", strconv.Itoa(proc.PID), "-o", "lstart=")
	output, err = cmd.Output()
	if err == nil {
		startTimeStr := strings.TrimSpace(string(output))
		// Parse macOS lstart format: "Thu Dec 28 10:30:45 2023"
		if t, err := time.Parse("Mon Jan _2 15:04:05 2006", startTimeStr); err == nil {
			proc.StartTime = t
		} else {
			// Fallback to current time if parsing fails
			proc.StartTime = time.Now()
		}
	}

	// Get working directory
	cmd = exec.Command("lsof", "-p", strconv.Itoa(proc.PID), "-d", "cwd", "-a")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "cwd") {
				fields := strings.Fields(line)
				if len(fields) > 8 {
					cwd := fields[len(fields)-1]
					proc.ProjectPath = detectProject(proc.PID, cwd)
				}
			}
		}
	}

	// Simple Docker detection on macOS
	if strings.Contains(proc.Command, "docker") || strings.Contains(proc.Name, "com.docker") {
		proc.IsDocker = true
	}
}
