package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"go.mongodb.org/mongo-driver/bson"
)

// --- Local Project Storage (machine-specific, NOT in MongoDB) ---

type LocalProjectData struct {
	LocalPath string `json:"localPath"`
	IsCloned  bool   `json:"isCloned"`
}

type LocalProjectsMap map[string]LocalProjectData // projectID -> data

func localProjectsFilePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".nyx", "local-projects.json")
}

func loadLocalProjects() (LocalProjectsMap, error) {
	path := localProjectsFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return LocalProjectsMap{}, nil
		}
		return nil, err
	}
	var m LocalProjectsMap
	if err := json.Unmarshal(data, &m); err != nil {
		return LocalProjectsMap{}, nil
	}
	return m, nil
}

func saveLocalProjects(m LocalProjectsMap) error {
	path := localProjectsFilePath()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func getLocalProjectData(projectID string) LocalProjectData {
	m, err := loadLocalProjects()
	if err != nil {
		return LocalProjectData{}
	}
	return m[projectID]
}

func setLocalProjectData(projectID string, d LocalProjectData) error {
	m, err := loadLocalProjects()
	if err != nil {
		m = LocalProjectsMap{}
	}
	m[projectID] = d
	return saveLocalProjects(m)
}

// --- Return Types ---

type ProjectLocalStatus struct {
	IsCloned              bool   `json:"isCloned"`
	LocalPath             string `json:"localPath"`
	CurrentBranch         string `json:"currentBranch"`
	HasUncommittedChanges bool   `json:"hasUncommittedChanges"`
	LastCommit            string `json:"lastCommit"`
}

type EnvFileStatus struct {
	Name          string `json:"name"`
	Exists        bool   `json:"exists"`
	ExampleExists bool   `json:"exampleExists"`
	Path          string `json:"path"`
}

type EnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type PortWithProject struct {
	Port        int    `json:"port"`
	Service     string `json:"service"`
	Protocol    string `json:"protocol"`
	ProjectID   string `json:"projectId"`
	ProjectName string `json:"projectName"`
}

// --- Repository Cloning ---

func (a *App) CloneRepository(projectId string) (string, error) {
	project, err := a.GetProject(projectId)
	if err != nil {
		return "", fmt.Errorf("project not found: %v", err)
	}
	if project.RepoURL == "" {
		return "", fmt.Errorf("no repository URL configured")
	}

	// Determine clone path
	home, _ := os.UserHomeDir()
	safeName := sanitizeDirName(project.Name)
	clonePath := filepath.Join(home, "Projects", safeName)

	// Check if already cloned
	local := getLocalProjectData(projectId)
	if local.IsCloned && local.LocalPath != "" {
		if _, err := os.Stat(filepath.Join(local.LocalPath, ".git")); err == nil {
			return local.LocalPath, nil
		}
	}

	// Emit cloning event
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "project:cloning", projectId)
	}

	// Ensure parent dir exists
	if err := os.MkdirAll(filepath.Dir(clonePath), 0755); err != nil {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "project:clone-error", projectId)
		}
		return "", fmt.Errorf("mkdir error: %v", err)
	}

	// Run git clone
	cmd := exec.Command("git", "clone", project.RepoURL, clonePath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "project:clone-error", projectId)
		}
		return "", fmt.Errorf("git clone failed: %s - %v", string(output), err)
	}

	// Save local data
	if err := setLocalProjectData(projectId, LocalProjectData{
		LocalPath: clonePath,
		IsCloned:  true,
	}); err != nil {
		return "", fmt.Errorf("failed to save local data: %v", err)
	}

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "project:cloned", projectId)
	}

	go a.LogActivity("cloned", "project", projectId, "Cloned repository for: "+project.Name, clonePath)
	return clonePath, nil
}

func (a *App) CheckLocalRepo(projectId string) ProjectLocalStatus {
	local := getLocalProjectData(projectId)
	if !local.IsCloned || local.LocalPath == "" {
		return ProjectLocalStatus{}
	}

	gitDir := filepath.Join(local.LocalPath, ".git")
	if _, err := os.Stat(gitDir); err != nil {
		return ProjectLocalStatus{IsCloned: false, LocalPath: local.LocalPath}
	}

	status := ProjectLocalStatus{
		IsCloned:  true,
		LocalPath: local.LocalPath,
	}

	// Get current branch
	if out, err := exec.Command("git", "-C", local.LocalPath, "rev-parse", "--abbrev-ref", "HEAD").Output(); err == nil {
		status.CurrentBranch = strings.TrimSpace(string(out))
	}

	// Check uncommitted changes
	if out, err := exec.Command("git", "-C", local.LocalPath, "status", "--porcelain").Output(); err == nil {
		status.HasUncommittedChanges = len(strings.TrimSpace(string(out))) > 0
	}

	// Get last commit
	if out, err := exec.Command("git", "-C", local.LocalPath, "log", "-1", "--format=%s (%ar)").Output(); err == nil {
		status.LastCommit = strings.TrimSpace(string(out))
	}

	return status
}

func (a *App) PullLatest(projectId string) (string, error) {
	local := getLocalProjectData(projectId)
	if !local.IsCloned || local.LocalPath == "" {
		return "", fmt.Errorf("repository not cloned locally")
	}

	cmd := exec.Command("git", "-C", local.LocalPath, "pull")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git pull failed: %s - %v", string(output), err)
	}

	go a.LogActivity("pulled", "project", projectId, "Pulled latest for project", strings.TrimSpace(string(output)))
	return strings.TrimSpace(string(output)), nil
}

func (a *App) OpenInTerminal(projectId string) error {
	local := getLocalProjectData(projectId)
	if !local.IsCloned || local.LocalPath == "" {
		return fmt.Errorf("repository not cloned locally")
	}

	cmd := exec.Command("open", "-a", "Terminal", local.LocalPath)
	return cmd.Start()
}

func (a *App) SetLocalPath(projectId string, localPath string) error {
	if _, err := os.Stat(localPath); err != nil {
		return fmt.Errorf("path does not exist: %v", err)
	}
	return setLocalProjectData(projectId, LocalProjectData{
		LocalPath: localPath,
		IsCloned:  true,
	})
}

// --- Port Management ---

func (a *App) AddPort(projectId string, port int, service string, protocol string) (Project, error) {
	project, err := a.GetProject(projectId)
	if err != nil {
		return Project{}, err
	}

	if project.Ports == nil {
		project.Ports = []PortEntry{}
	}

	// Check for duplicate port in same project
	for _, p := range project.Ports {
		if p.Port == port {
			return Project{}, fmt.Errorf("port %d already registered", port)
		}
	}

	project.Ports = append(project.Ports, PortEntry{
		Port:     port,
		Service:  service,
		Protocol: protocol,
	})

	updated, err := a.UpdateProject(project)
	if err != nil {
		return Project{}, err
	}

	go a.LogActivity("updated", "project", projectId, fmt.Sprintf("Added port %d (%s) to project: %s", port, service, project.Name), "")
	return updated, nil
}

func (a *App) RemovePort(projectId string, port int) (Project, error) {
	project, err := a.GetProject(projectId)
	if err != nil {
		return Project{}, err
	}

	newPorts := []PortEntry{}
	for _, p := range project.Ports {
		if p.Port != port {
			newPorts = append(newPorts, p)
		}
	}
	project.Ports = newPorts

	updated, err := a.UpdateProject(project)
	if err != nil {
		return Project{}, err
	}

	go a.LogActivity("updated", "project", projectId, fmt.Sprintf("Removed port %d from project: %s", port, project.Name), "")
	return updated, nil
}

func (a *App) GetAllUsedPorts() ([]PortWithProject, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	cursor, err := db.Collection("projects").Find(context.Background(), bson.M{
		"ports": bson.M{"$exists": true, "$ne": nil, "$not": bson.M{"$size": 0}},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var projects []Project
	if err := cursor.All(context.Background(), &projects); err != nil {
		return nil, err
	}

	var result []PortWithProject
	for _, p := range projects {
		for _, port := range p.Ports {
			result = append(result, PortWithProject{
				Port:        port.Port,
				Service:     port.Service,
				Protocol:    port.Protocol,
				ProjectID:   p.ID,
				ProjectName: p.Name,
			})
		}
	}

	if result == nil {
		result = []PortWithProject{}
	}
	return result, nil
}

func (a *App) CheckPortConflicts(port int) ([]string, error) {
	db, err := getDB()
	if err != nil {
		return nil, err
	}

	cursor, err := db.Collection("projects").Find(context.Background(), bson.M{
		"ports.port": port,
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.Background())

	var projects []Project
	if err := cursor.All(context.Background(), &projects); err != nil {
		return nil, err
	}

	var names []string
	for _, p := range projects {
		names = append(names, p.Name)
	}
	if names == nil {
		names = []string{}
	}
	return names, nil
}

func (a *App) CheckPortInUse(port int) bool {
	ln, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 500*time.Millisecond)
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// --- Env File Management ---

func (a *App) ScanEnvFiles(projectId string) ([]EnvFileStatus, error) {
	local := getLocalProjectData(projectId)
	if !local.IsCloned || local.LocalPath == "" {
		return []EnvFileStatus{}, nil
	}

	var results []EnvFileStatus
	tracked := map[string]bool{}

	// Walk root directory for .env* files
	entries, err := os.ReadDir(local.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("read dir error: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, ".env") {
			continue
		}

		if strings.HasSuffix(name, ".example") {
			// This is an example file - find the corresponding actual file
			actualName := strings.TrimSuffix(name, ".example")
			actualPath := filepath.Join(local.LocalPath, actualName)
			_, actualExists := os.Stat(actualPath)

			if !tracked[actualName] {
				results = append(results, EnvFileStatus{
					Name:          actualName,
					Exists:        actualExists == nil,
					ExampleExists: true,
					Path:          actualName,
				})
				tracked[actualName] = true
			}
		} else {
			// Actual env file
			if !tracked[name] {
				examplePath := filepath.Join(local.LocalPath, name+".example")
				_, exampleExists := os.Stat(examplePath)

				results = append(results, EnvFileStatus{
					Name:          name,
					Exists:        true,
					ExampleExists: exampleExists == nil,
					Path:          name,
				})
				tracked[name] = true
			}
		}
	}

	if results == nil {
		results = []EnvFileStatus{}
	}
	return results, nil
}

func (a *App) GetEnvFileContent(projectId string, filename string) (string, error) {
	local := getLocalProjectData(projectId)
	if !local.IsCloned || local.LocalPath == "" {
		return "", fmt.Errorf("repo not cloned")
	}

	// Security: prevent path traversal
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		return "", fmt.Errorf("invalid filename")
	}

	path := filepath.Join(local.LocalPath, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read error: %v", err)
	}
	return string(data), nil
}

func (a *App) SaveEnvFileContent(projectId string, filename string, content string) error {
	local := getLocalProjectData(projectId)
	if !local.IsCloned || local.LocalPath == "" {
		return fmt.Errorf("repo not cloned")
	}

	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		return fmt.Errorf("invalid filename")
	}

	path := filepath.Join(local.LocalPath, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("write error: %v", err)
	}

	go a.LogActivity("updated", "project", projectId, "Updated env file: "+filename, "")
	return nil
}

func (a *App) CreateEnvFromExample(projectId string, exampleFile string) error {
	local := getLocalProjectData(projectId)
	if !local.IsCloned || local.LocalPath == "" {
		return fmt.Errorf("repo not cloned")
	}

	if strings.Contains(exampleFile, "..") || strings.Contains(exampleFile, "/") {
		return fmt.Errorf("invalid filename")
	}

	srcPath := filepath.Join(local.LocalPath, exampleFile)
	targetName := strings.TrimSuffix(exampleFile, ".example")
	dstPath := filepath.Join(local.LocalPath, targetName)

	// Don't overwrite existing
	if _, err := os.Stat(dstPath); err == nil {
		return fmt.Errorf("%s already exists", targetName)
	}

	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read example error: %v", err)
	}

	if err := os.WriteFile(dstPath, data, 0644); err != nil {
		return fmt.Errorf("write error: %v", err)
	}

	go a.LogActivity("created", "project", projectId, "Created "+targetName+" from "+exampleFile, "")
	return nil
}

func (a *App) GetEnvVariables(projectId string, filename string) ([]EnvVar, error) {
	content, err := a.GetEnvFileContent(projectId, filename)
	if err != nil {
		return nil, err
	}

	var vars []EnvVar
	scanner := bufio.NewScanner(strings.NewReader(content))
	envLine := regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)=(.*)$`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if matches := envLine.FindStringSubmatch(line); matches != nil {
			vars = append(vars, EnvVar{Key: matches[1], Value: matches[2]})
		}
	}

	if vars == nil {
		vars = []EnvVar{}
	}
	return vars, nil
}

func (a *App) SetEnvVariable(projectId string, filename string, key string, value string) error {
	content, err := a.GetEnvFileContent(projectId, filename)
	if err != nil {
		return err
	}

	envLine := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `=.*$`)
	newLine := key + "=" + value

	if envLine.MatchString(content) {
		content = envLine.ReplaceAllString(content, newLine)
	} else {
		if !strings.HasSuffix(content, "\n") && content != "" {
			content += "\n"
		}
		content += newLine + "\n"
	}

	return a.SaveEnvFileContent(projectId, filename, content)
}

// --- Helpers ---

func sanitizeDirName(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_\-. ]+`)
	safe := re.ReplaceAllString(name, "")
	safe = strings.TrimSpace(safe)
	if safe == "" {
		safe = "project"
	}
	return safe
}

// GetAllProjects returns all projects (no filter)
func (a *App) GetAllProjects() ([]Project, error) {
	return a.GetProjects("", "")
}
