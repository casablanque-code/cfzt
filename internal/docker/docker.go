package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const socketPath = "/var/run/docker.sock"

type portBinding struct {
	HostPort string `json:"HostPort"`
}

type containerInspect struct {
	State struct {
		Running bool `json:"Running"`
	} `json:"State"`
	NetworkSettings struct {
		Ports map[string][]portBinding `json:"Ports"`
	} `json:"NetworkSettings"`
}

type containerSummary struct {
	Names []string      `json:"Names"`
	State string        `json:"State"`
	Ports []portSummary `json:"Ports"`
}

type portSummary struct {
	PublicPort uint16 `json:"PublicPort"`
	Type       string `json:"Type"`
}

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		Dial: func(_, _ string) (net.Conn, error) {
			return net.DialTimeout("unix", socketPath, 5*time.Second)
		},
	},
}

// apiVersion asks Docker for the minimum supported API version.
func apiVersion() string {
	resp, err := httpClient.Get("http://localhost/version")
	if err != nil {
		return "v1.44" // safe fallback
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var result struct {
		MinAPIVersion string `json:"MinAPIVersion"`
		APIVersion    string `json:"ApiVersion"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "v1.44"
	}
	if result.APIVersion != "" {
		return "v" + result.APIVersion
	}
	return "v1.44"
}

func dockerGet(path string, v any) error {
	ver := apiVersion()
	url := "http://localhost/" + ver + path
	// clean up double slashes
	url = strings.ReplaceAll(url, "//", "/")
	url = strings.Replace(url, "http:/", "http://", 1)

	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("docker socket unavailable: %w\n  hint: is Docker running? check `docker ps`", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == 404 {
		return fmt.Errorf("not found")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("docker API %d: %s", resp.StatusCode, string(body))
	}
	return json.Unmarshal(body, v)
}

// FindContainerPort returns the host-bound TCP port for a container by name.
func FindContainerPort(name string) (string, error) {
	var inspect containerInspect
	if err := dockerGet("/containers/"+name+"/json", &inspect); err != nil {
		return findByList(name)
	}
	if !inspect.State.Running {
		return "", fmt.Errorf("container %q exists but is not running", name)
	}
	return pickPort(name, inspect.NetworkSettings.Ports)
}

func findByList(name string) (string, error) {
	var list []containerSummary
	if err := dockerGet("/containers/json", &list); err != nil {
		return "", err
	}
	for _, c := range list {
		for _, n := range c.Names {
			clean := n
			if len(clean) > 0 && clean[0] == '/' {
				clean = clean[1:]
			}
			if clean != name {
				continue
			}
			if c.State != "running" {
				return "", fmt.Errorf("container %q found but state is %q", name, c.State)
			}
			for _, p := range c.Ports {
				if p.PublicPort > 0 && p.Type == "tcp" {
					return fmt.Sprintf("%d", p.PublicPort), nil
				}
			}
			return "", fmt.Errorf("container %q running but has no published TCP ports\n  hint: start it with -p <host_port>:<container_port>", name)
		}
	}
	return "", fmt.Errorf("container %q not found\n  hint: check `docker ps` for the exact container name", name)
}

func pickPort(name string, ports map[string][]portBinding) (string, error) {
	for _, bindings := range ports {
		for _, b := range bindings {
			if b.HostPort != "" {
				return b.HostPort, nil
			}
		}
	}
	return "", fmt.Errorf("container %q has no published ports\n  hint: start it with -p <host_port>:<container_port>", name)
}
