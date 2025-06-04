//go:build linux

package process

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type platformFinder struct{}

func (f *platformFinder) FindByPort(port int) (*Process, error) {
	// First try ss (socket statistics)
	proc, err := f.findUsingSS(port)
	if err == nil && proc != nil {
		return proc, nil
	}

	// Fallback to netstat
	return f.findUsingNetstat(port)
}

func (f *platformFinder) ListAll() ([]*Process, error) {
	processes := make([]*Process, 0)

	// Try ss first
	cmd := exec.Command("ss", "-tulnp")
	output, err := cmd.Output()
	if err == nil {
		procs := f.parseSSOutput(string(output))
		processes = append(processes, procs...)
	} else {
		// Fallback to netstat
		cmd = exec.Command("netstat", "-tulnp")
		output, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to list ports: %w", err)
		}
		procs := f.parseNetstatOutput(string(output))
		processes = append(processes, procs...)
	}

	return processes, nil
}

func (f *platformFinder) findUsingSS(port int) (*Process, error) {
	cmd := exec.Command("ss", "-tulnp", fmt.Sprintf("sport = :%d", port))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines[1:] { // Skip header
		if strings.Contains(line, fmt.Sprintf(":%d", port)) && strings.Contains(line, "LISTEN") {
			return f.parseSSLine(line, port)
		}
	}

	return nil, nil
}

func (f *platformFinder) findUsingNetstat(port int) (*Process, error) {
	cmd := exec.Command("netstat", "-tulnp")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, fmt.Sprintf(":%d", port)) && strings.Contains(line, "LISTEN") {
			return f.parseNetstatLine(line, port)
		}
	}

	return nil, nil
}

func (f *platformFinder) parseSSLine(line string, port int) (*Process, error) {
	// Parse ss output format
	fields := strings.Fields(line)
	if len(fields) < 7 {
		return nil, nil
	}

	// Extract PID/Program from last field (format: "users:(("nginx",pid=1234,fd=6))")
	pidProg := fields[len(fields)-1]
	if !strings.Contains(pidProg, "pid=") {
		return nil, nil
	}

	pidStart := strings.Index(pidProg, "pid=") + 4
	pidEnd := strings.Index(pidProg[pidStart:], ",")
	if pidEnd == -1 {
		pidEnd = strings.Index(pidProg[pidStart:], ")")
	}

	pid, err := strconv.Atoi(pidProg[pidStart : pidStart+pidEnd])
	if err != nil {
		return nil, nil
	}

	proc := &Process{
		PID:  pid,
		Port: port,
	}

	f.enrichProcessInfo(proc)
	return proc, nil
}

func (f *platformFinder) parseNetstatLine(line string, port int) (*Process, error) {
	fields := strings.Fields(line)
	if len(fields) < 7 {
		return nil, nil
	}

	// Parse PID/Program name
	pidProg := fields[6]
	if pidProg == "-" {
		return nil, nil
	}

	parts := strings.Split(pidProg, "/")
	if len(parts) != 2 {
		return nil, nil
	}

	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, nil
	}

	proc := &Process{
		PID:  pid,
		Name: parts[1],
		Port: port,
	}

	f.enrichProcessInfo(proc)
	return proc, nil
}

func (f *platformFinder) parseSSOutput(output string) []*Process {
	processes := make([]*Process, 0)
	lines := strings.Split(output, "\n")

	for _, line := range lines[1:] { // Skip header
		if !strings.Contains(line, "LISTEN") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		// Extract port from address
		addr := fields[4]
		parts := strings.Split(addr, ":")
		if len(parts) < 2 {
			continue
		}

		port, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			continue
		}

		proc, err := f.parseSSLine(line, port)
		if err == nil && proc != nil {
			processes = append(processes, proc)
		}
	}

	return processes
}

func (f *platformFinder) parseNetstatOutput(output string) []*Process {
	processes := make([]*Process, 0)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if !strings.Contains(line, "LISTEN") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		// Extract port
		addr := fields[3]
		parts := strings.Split(addr, ":")
		if len(parts) < 2 {
			continue
		}

		port, err := strconv.Atoi(parts[len(parts)-1])
		if err != nil {
			continue
		}

		proc, err := f.parseNetstatLine(line, port)
		if err == nil && proc != nil {
			processes = append(processes, proc)
		}
	}

	return processes
}

// getProcessStartTime gets the actual start time of a process on Linux
func getProcessStartTime(pid int) (time.Time, error) {
	// Read /proc/[pid]/stat
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return time.Time{}, err
	}

	// Parse the stat file
	// The start time is the 22nd field (21 when 0-indexed)
	// It's in clock ticks since boot
	content := string(data)

	// Find the last ) to handle process names with spaces/parentheses
	lastParen := strings.LastIndex(content, ")")
	if lastParen == -1 {
		return time.Time{}, fmt.Errorf("invalid stat format")
	}

	// Fields after the command name
	fields := strings.Fields(content[lastParen+1:])
	if len(fields) < 20 {
		return time.Time{}, fmt.Errorf("not enough fields in stat")
	}

	// Start time is the 20th field after the command name (0-indexed)
	startTicks, err := strconv.ParseInt(fields[19], 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	// Get system boot time
	var info syscall.Sysinfo_t
	if err := syscall.Sysinfo(&info); err != nil {
		return time.Time{}, err
	}

	// Get clock ticks per second
	clockTicks := int64(100) // Default, usually correct for Linux

	// Calculate start time
	bootTime := time.Now().Add(-time.Duration(info.Uptime) * time.Second)
	startTime := bootTime.Add(time.Duration(startTicks*1000/clockTicks) * time.Millisecond)

	return startTime, nil
}

func (f *platformFinder) enrichProcessInfo(proc *Process) {
	// Get process name if not already set
	if proc.Name == "" {
		if cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", proc.PID)); err == nil {
			proc.Name = strings.TrimSpace(string(cmdline))
		}
	}

	// Get command line
	if cmdline, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", proc.PID)); err == nil {
		proc.Command = strings.ReplaceAll(string(cmdline), "\x00", " ")
		proc.Command = strings.TrimSpace(proc.Command)
	}

	// Get working directory
	if cwd, err := os.Readlink(fmt.Sprintf("/proc/%d/cwd", proc.PID)); err == nil {
		proc.ProjectPath = detectProject(proc.PID, cwd)
	}

	// Get actual start time
	if startTime, err := getProcessStartTime(proc.PID); err == nil {
		proc.StartTime = startTime
	} else {
		// Fallback to stat time
		if stat, err := os.Stat(fmt.Sprintf("/proc/%d", proc.PID)); err == nil {
			proc.StartTime = stat.ModTime()
		}
	}

	// Check if Docker
	proc.IsDocker, proc.DockerID = isDockerProcess(proc.PID)
}
