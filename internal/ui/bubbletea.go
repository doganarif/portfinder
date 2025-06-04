package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/doganarif/portfinder/internal/process"
)

var (
	baseStyle = lipgloss.NewStyle().
			PaddingLeft(1).
			PaddingRight(1)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true).
			Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Bold(true).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)

	portUsedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	portFreeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86"))

	dockerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")).
			Bold(true)
)

type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Kill   key.Binding
	Quit   key.Binding
	Help   key.Binding
	Reload key.Binding
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Kill, k.Reload},
		{k.Help, k.Quit},
	}
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Kill: key.NewBinding(
		key.WithKeys("delete", "d"),
		key.WithHelp("del/d", "kill process"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Reload: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "reload"),
	),
}

// ProcessListModel represents the process list view
type ProcessListModel struct {
	processes    []*process.Process
	table        table.Model
	spinner      spinner.Model
	loading      bool
	err          error
	help         help.Model
	showHelp     bool
	width        int
	height       int
	message      string
	messageTimer *time.Timer
}

// ProcessDetailModel represents a single process detail view
type ProcessDetailModel struct {
	process *process.Process
	width   int
	height  int
}

// NewProcessListModel creates a new process list model
func NewProcessListModel(processes []*process.Process) ProcessListModel {
	columns := []table.Column{
		{Title: "Port", Width: 8},
		{Title: "Process", Width: 15},
		{Title: "PID", Width: 8},
		{Title: "Project", Width: 30},
		{Title: "Running For", Width: 15},
		{Title: "Type", Width: 10},
	}

	rows := make([]table.Row, len(processes))
	for i, p := range processes {
		rows[i] = processToRow(p)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(15),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return ProcessListModel{
		processes: processes,
		table:     t,
		spinner:   sp,
		help:      help.New(),
	}
}

func processToRow(p *process.Process) table.Row {
	projectPath := p.ProjectPath
	if projectPath == "" || projectPath == "unknown" {
		projectPath = "-"
	}

	processType := "Native"
	if p.IsDocker {
		processType = "Docker"
	}

	return table.Row{
		fmt.Sprintf("%d", p.Port),
		p.Name,
		fmt.Sprintf("%d", p.PID),
		truncate(projectPath, 30),
		formatDuration(time.Since(p.StartTime)),
		processType,
	}
}

func (m ProcessListModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m ProcessListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.table.SetHeight(msg.Height - 10)
		m.table.SetWidth(msg.Width - 4)

	case tea.KeyMsg:
		if m.loading {
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp

		case key.Matches(msg, keys.Kill):
			if len(m.processes) > 0 && m.table.Cursor() < len(m.processes) {
				proc := m.processes[m.table.Cursor()]
				if err := proc.Kill(); err != nil {
					m.message = fmt.Sprintf("❌ Failed to kill process: %v", err)
				} else {
					m.message = fmt.Sprintf("✅ Killed %s (PID: %d)", proc.Name, proc.PID)
					// Remove from list
					m.processes = append(m.processes[:m.table.Cursor()], m.processes[m.table.Cursor()+1:]...)
					rows := make([]table.Row, len(m.processes))
					for i, p := range m.processes {
						rows[i] = processToRow(p)
					}
					m.table.SetRows(rows)
				}
				m.messageTimer = time.NewTimer(3 * time.Second)
				cmds = append(cmds, waitForTimer(m.messageTimer))
			}

		case key.Matches(msg, keys.Reload):
			m.loading = true
			cmds = append(cmds, reloadProcesses())
		}

	case processesLoadedMsg:
		m.loading = false
		m.processes = msg.processes
		rows := make([]table.Row, len(m.processes))
		for i, p := range m.processes {
			rows[i] = processToRow(p)
		}
		m.table.SetRows(rows)

	case timerExpiredMsg:
		m.message = ""

	case spinner.TickMsg:
		if m.loading {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	m.table, cmd = m.table.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m ProcessListModel) View() string {
	var b strings.Builder

	title := titleStyle.Render("🔍 PortFinder - Active Processes")
	b.WriteString(title + "\n\n")

	if m.loading {
		b.WriteString(m.spinner.View() + " Loading processes...\n")
		return b.String()
	}

	if m.message != "" {
		b.WriteString(m.message + "\n\n")
	}

	count := infoStyle.Render(fmt.Sprintf("Found %d processes using network ports", len(m.processes)))
	b.WriteString(count + "\n\n")

	if len(m.processes) == 0 {
		b.WriteString(dimStyle.Render("No processes are using network ports\n"))
	} else {
		b.WriteString(m.table.View())
	}

	b.WriteString("\n")
	if m.showHelp {
		b.WriteString(m.help.View(keys))
	} else {
		b.WriteString(dimStyle.Render("Press ? for help"))
	}

	return baseStyle.Render(b.String())
}

// PortCheckModel represents the port check view
type PortCheckModel struct {
	ports   map[int]*process.Process
	loading bool
	spinner spinner.Model
	width   int
	height  int
}

// NewPortCheckModel creates a new port check model
func NewPortCheckModel(ports map[int]*process.Process) PortCheckModel {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return PortCheckModel{
		ports:   ports,
		spinner: sp,
	}
}

func (m PortCheckModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m PortCheckModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m PortCheckModel) View() string {
	var b strings.Builder

	title := titleStyle.Render("📊 Common Development Ports")
	b.WriteString(title + "\n\n")

	if m.loading {
		b.WriteString(m.spinner.View() + " Checking ports...\n")
		return b.String()
	}

	// Group ports by category
	categories := []struct {
		Name  string
		Ports []int
	}{
		{"Frontend", []int{3000, 3001, 4200, 5173, 8080}},
		{"Backend", []int{4000, 5000, 8000, 9000}},
		{"Databases", []int{3306, 5432, 6379, 27017}},
		{"Tools", []int{9200, 9090, 3100, 8983}},
	}

	for _, category := range categories {
		b.WriteString(headerStyle.Render(category.Name) + "\n")

		for _, port := range category.Ports {
			proc, exists := m.ports[port]
			if exists && proc != nil {
				status := portUsedStyle.Render(fmt.Sprintf("● %d", port))
				info := fmt.Sprintf("%s (%s)", proc.Name, proc.ProjectPath)
				if proc.IsDocker {
					info = dockerStyle.Render("[Docker] ") + info
				}
				b.WriteString(fmt.Sprintf("  %s %s\n", status, dimStyle.Render(info)))
			} else {
				status := portFreeStyle.Render(fmt.Sprintf("○ %d", port))
				b.WriteString(fmt.Sprintf("  %s %s\n", status, dimStyle.Render("available")))
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("\n" + dimStyle.Render("Press q to quit"))

	return baseStyle.Render(b.String())
}

// Helper functions

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// Messages

type processesLoadedMsg struct {
	processes []*process.Process
}

type timerExpiredMsg struct{}

// Commands

func reloadProcesses() tea.Cmd {
	return func() tea.Msg {
		finder := process.NewFinder()
		processes, _ := finder.ListAll()
		return processesLoadedMsg{processes: processes}
	}
}

func waitForTimer(t *time.Timer) tea.Cmd {
	return func() tea.Msg {
		<-t.C
		return timerExpiredMsg{}
	}
}

// ShowProcessList displays an interactive process list
func ShowProcessList(processes []*process.Process) error {
	p := tea.NewProgram(NewProcessListModel(processes), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// ShowPortCheck displays the port check view
func ShowPortCheck(ports map[int]*process.Process) error {
	p := tea.NewProgram(NewPortCheckModel(ports), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// ShowProcessDetail displays detailed information about a single process
func ShowProcessDetail(proc *process.Process, interactive bool) {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(portUsedStyle.Render(fmt.Sprintf("🔍 Port %d is in use by:", proc.Port)))
	b.WriteString("\n\n")

	// Create a nice box for the process info
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2)

	var content strings.Builder
	content.WriteString(fmt.Sprintf("%s %s\n", headerStyle.Render("Process:"), proc.Name))
	content.WriteString(fmt.Sprintf("%s %d\n", headerStyle.Render("PID:"), proc.PID))
	content.WriteString(fmt.Sprintf("%s %s\n", headerStyle.Render("Command:"), truncate(proc.Command, 50)))
	content.WriteString(fmt.Sprintf("%s %s\n", headerStyle.Render("Project:"), formatProject(proc.ProjectPath)))
	content.WriteString(fmt.Sprintf("%s %s\n", headerStyle.Render("Started:"), formatTime(proc.StartTime)))
	content.WriteString(fmt.Sprintf("%s %s\n", headerStyle.Render("Running For:"), formatDuration(time.Since(proc.StartTime))))

	if proc.IsDocker {
		content.WriteString(fmt.Sprintf("%s %s\n", headerStyle.Render("Docker:"), dockerStyle.Render("Yes (Container: "+proc.DockerID+")")))
	}

	fmt.Print(boxStyle.Render(content.String()))
	fmt.Println()

	if interactive {
		if SimpleConfirm("\nKill this process?") {
			if err := proc.Kill(); err != nil {
				ErrorMsg("Failed to kill process: %v", err)
			} else {
				SuccessMsg("Process killed successfully")
			}
		}
	}
}

func formatProject(path string) string {
	if path == "" || path == "unknown" {
		return dimStyle.Render("unknown")
	}
	return path
}

func formatTime(t time.Time) string {
	return t.Format("Jan 2, 15:04:05")
}
