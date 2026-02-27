package process

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo represents a rich process trace
type ProcessInfo struct {
	PID           int32
	User          string
	CPU           float64
	RAM           float32
	Command       string
	ContainerID   string
	ContainerName string
}

// dockerClient provides a Unix socket HTTP client
func dockerClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
		Timeout: 5 * time.Second,
	}
}

// GetContainersMap connects to docker socket and returns map[ID]Name
func GetContainersMap() map[string]string {
	cmap := make(map[string]string)
	client := dockerClient()

	resp, err := client.Get("http://localhost/containers/json")
	if err != nil {
		// Socket not mounted or Docker dead
		log.Printf("Docker Socket Error: Permission denied or unavailable. Ensure /var/run/docker.sock is mounted with appropriate permissions: %v", err)
		return cmap
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return cmap
	}

	body, _ := io.ReadAll(resp.Body)
	var containers []struct {
		Id    string   `json:"Id"`
		Names []string `json:"Names"`
	}
	if err := json.Unmarshal(body, &containers); err != nil {
		return cmap
	}

	for _, c := range containers {
		if len(c.Names) > 0 {
			name := strings.TrimPrefix(c.Names[0], "/")
			// Truncate ID to 12 chars
			shortID := c.Id
			if len(shortID) > 12 {
				shortID = shortID[:12]
			}
			cmap[shortID] = name
			cmap[c.Id] = name
		}
	}
	return cmap
}

// Updated regex to catch Docker Desktop / WSL runtimes and standard Docker daemon IDs
// Fallback regex to capture any 64 character hex string in cgroups
var cgroupRegex = regexp.MustCompile(`([a-f0-9]{64})`)

func getContainerIDFromCgroup(pid int32) string {
	paths := []string{
		fmt.Sprintf("/host/proc/%d/cgroup", pid),
		fmt.Sprintf("/proc/%d/cgroup", pid),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err == nil {
			content := string(data)
			
			// Priority: Match explicit Docker / K8s formats
			explicitRegex := regexp.MustCompile(`(?:/docker/|/kubepods[^/]*/|/containers/|[a-f0-9]{8}-|docker-|containerd-)([a-f0-9]{64})`)
			match := explicitRegex.FindStringSubmatch(content)
			if len(match) > 1 {
				return truncateID(match[1])
			}
			
			// Fallback: match any 64 char hex string (e.g for WSL or alternative CRI)
			hexMatch := cgroupRegex.FindString(content)
			if hexMatch != "" {
				return truncateID(hexMatch)
			}
		}
	}
	return ""
}

func truncateID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// GetProcesses fetches active processes and filters via query
func GetProcesses(query string, sortBy string, sortDir string) []ProcessInfo {
	query = strings.ToLower(query)
	cmap := GetContainersMap()

	var results []ProcessInfo
	procs, err := process.Processes()
	if err != nil {
		log.Printf("Error getting processes: %v", err)
		return results
	}

	for _, p := range procs {
		cmd, err := p.Cmdline()
		if err != nil || cmd == "" {
			name, err := p.Name()
			if err != nil || name == "" {
				continue
			}
			cmd = "[" + name + "]"
		}

		pidStr := fmt.Sprintf("%d", p.Pid)

		if query != "" {
			// Check if query matches part of the command OR exactly matches the PID
			if !strings.Contains(strings.ToLower(cmd), query) && pidStr != query {
				continue
			}
		}

		cpu, _ := p.CPUPercent()
		ram, _ := p.MemoryPercent()
		user, _ := p.Username()
		if user == "" {
			user = "root"
		}

		containerID := getContainerIDFromCgroup(p.Pid)
		containerName := ""
		if containerID != "" {
			if name, ok := cmap[containerID]; ok {
				containerName = name
			} else if name, ok := cmap[containerID[:12]]; ok {
				containerName = name
			}
		}

		results = append(results, ProcessInfo{
			PID:           p.Pid,
			User:          user,
			CPU:           cpu,
			RAM:           ram,
			Command:       cmd,
			ContainerID:   containerID,
			ContainerName: containerName,
		})
	}

	// Dynamic Sorting
	sort.Slice(results, func(i, j int) bool {
		asc := sortDir == "asc"
		switch sortBy {
		case "pid":
			if asc {
				return results[i].PID < results[j].PID
			}
			return results[i].PID > results[j].PID
		case "user":
			if asc {
				return results[i].User < results[j].User
			}
			return results[i].User > results[j].User
		case "container":
			// Special logic for container
			hasContainerI := results[i].ContainerName != "" || results[i].ContainerID != ""
			hasContainerJ := results[j].ContainerName != "" || results[j].ContainerID != ""
			
			if hasContainerI != hasContainerJ {
				if asc {
					return hasContainerI // containers first
				}
				return hasContainerJ // non-containers first
			}
			// Both have containers or both don't, sort alphabetically
			if asc {
				return results[i].ContainerName < results[j].ContainerName
			}
			return results[i].ContainerName > results[j].ContainerName
		case "command":
			if asc {
				return results[i].Command < results[j].Command
			}
			return results[i].Command > results[j].Command
		case "ram":
			if asc {
				return results[i].RAM < results[j].RAM
			}
			return results[i].RAM > results[j].RAM
		case "cpu":
			fallthrough
		default:
			if results[i].CPU == results[j].CPU {
				if asc {
					return results[i].RAM < results[j].RAM
				}
				return results[i].RAM > results[j].RAM
			}
			if asc {
				return results[i].CPU < results[j].CPU
			}
			return results[i].CPU > results[j].CPU
		}
	})

	// Limit to top 150 for ui rendering perf
	if len(results) > 150 {
		results = results[:150]
	}

	return results
}

func KillProcess(pid int32) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}

func StopContainer(id string) error {
	client := dockerClient()
	req, err := http.NewRequest("POST", fmt.Sprintf("http://localhost/containers/%s/stop?t=5", id), nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("docker stop failed: status %d - %s", resp.StatusCode, string(body))
	}
	return nil
}
