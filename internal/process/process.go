package process

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Process represents a process using a network port
type Process struct {
	PID         int
	Name        string
	Port        int
	Command     string
	ProjectPath string
	StartTime   time.Time
	IsDocker    bool
	DockerID    string
}

// Finder interface for finding processes
type Finder interface {
	FindByPort(port int) (*Process, error)
	ListAll() ([]*Process, error)
}

// NewFinder creates a platform-specific process finder
func NewFinder() Finder {
	return &platformFinder{}
}

// Kill terminates the process
func (p *Process) Kill() error {
	// Try graceful shutdown first
	process, err := os.FindProcess(p.PID)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}

	// Send SIGTERM for graceful shutdown
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait a moment for graceful shutdown
	time.Sleep(2 * time.Second)

	// Check if process still exists
	if err := process.Signal(syscall.Signal(0)); err == nil {
		// Process still running, force kill
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	return nil
}

// detectProject tries to determine the project directory
func detectProject(pid int, cwd string) string {
	if cwd == "" {
		return "unknown"
	}

	// Clean up the path
	cwd = filepath.Clean(cwd)

	// Look for common project indicators
	indicators := []string{
		"package.json",
		"go.mod",
		"Cargo.toml",
		"pom.xml",
		"build.gradle",
		"requirements.txt",
		"Gemfile",
		".git",
	}

	current := cwd
	for {
		for _, indicator := range indicators {
			if _, err := os.Stat(filepath.Join(current, indicator)); err == nil {
				return current
			}
		}

		parent := filepath.Dir(current)
		if parent == current || parent == "/" || parent == "." {
			break
		}
		current = parent
	}

	// If no project found, return the working directory
	if strings.Contains(cwd, "home") || strings.Contains(cwd, "Users") {
		parts := strings.Split(cwd, string(filepath.Separator))
		if len(parts) > 4 {
			// Return a reasonable subset of the path
			return filepath.Join(parts[len(parts)-2:]...)
		}
	}

	return filepath.Base(cwd)
}

// isDockerProcess checks if a process is running in Docker
func isDockerProcess(pid int) (bool, string) {
	// Check if process is in a container by examining cgroup
	cgroupPath := fmt.Sprintf("/proc/%d/cgroup", pid)
	data, err := os.ReadFile(cgroupPath)
	if err != nil {
		return false, ""
	}

	content := string(data)
	if strings.Contains(content, "docker") {
		// Try to extract container ID
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.Contains(line, "docker") {
				parts := strings.Split(line, "/")
				if len(parts) > 0 {
					containerID := parts[len(parts)-1]
					if len(containerID) >= 12 {
						return true, containerID[:12]
					}
				}
			}
		}
		return true, "unknown"
	}

	return false, ""
}
