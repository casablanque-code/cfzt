package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"sort"
	"strconv"
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
	PrivatePort uint16 `json:"PrivatePort"`
	PublicPort  uint16 `json:"PublicPort"`
	Type        string `json:"Type"`
}

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, "unix", socketPath)
		},
	},
}

func apiVersion() string {
	resp, err := httpClient.Get("http://localhost/version")
	if err != nil {
		return "v1.44"
	}
	defer func() { _ = resp.Body.Close() }()

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
	url = strings.ReplaceAll(url, "//", "/")
	url = strings.Replace(url, "http:/", "http://", 1)

	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("docker socket unavailable: %w\n  hint: is Docker running? check `docker ps`", err)
	}
	defer func() { _ = resp.Body.Close() }()

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

func FindContainerPort(name, containerPort string) (string, error) {
	var inspect containerInspect
	if err := dockerGet("/containers/"+name+"/json", &inspect); err != nil {
		return findByList(name, containerPort)
	}
	if !inspect.State.Running {
		return "", fmt.Errorf("container %q exists but is not running", name)
	}
	return pickPort(name, containerPort, inspect.NetworkSettings.Ports)
}

func findByList(name, containerPort string) (string, error) {
	var list []containerSummary
	if err := dockerGet("/containers/json", &list); err != nil {
		return "", err
	}

	for _, c := range list {
		for _, n := range c.Names {
			clean := strings.TrimPrefix(n, "/")
			if clean != name {
				continue
			}
			if c.State != "running" {
				return "", fmt.Errorf("container %q found but state is %q", name, c.State)
			}

			if containerPort != "" {
				for _, p := range c.Ports {
					if p.Type == "tcp" &&
						p.PublicPort > 0 &&
						fmt.Sprintf("%d", p.PrivatePort) == containerPort {
						return fmt.Sprintf("%d", p.PublicPort), nil
					}
				}
				return "", fmt.Errorf(
					"container %q has no published TCP binding for container port %s\n  hint: check `docker ps` for the container's actual published ports",
					name,
					containerPort,
				)
			}

			for _, p := range c.Ports {
				if p.PublicPort > 0 && p.Type == "tcp" {
					return fmt.Sprintf("%d", p.PublicPort), nil
				}
			}

			return "", fmt.Errorf(
				"container %q running but has no published TCP ports\n  hint: start it with -p <host_port>:<container_port>",
				name,
			)
		}
	}

	return "", fmt.Errorf(
		"container %q not found\n  hint: check `docker ps` for the exact container name",
		name,
	)
}

func pickPort(name, containerPort string, ports map[string][]portBinding) (string, error) {
	if containerPort != "" {
		key := containerPort + "/tcp"
		for _, b := range ports[key] {
			if b.HostPort != "" {
				return b.HostPort, nil
			}
		}
		return "", fmt.Errorf(
			"container %q has no published TCP binding for container port %s\n  hint: check `docker inspect %s` for its actual published ports",
			name,
			containerPort,
			name,
		)
	}

	var tcpKeys []string
	for k, bindings := range ports {
		if len(bindings) == 0 || bindings[0].HostPort == "" {
			continue
		}
		if !strings.HasSuffix(k, "/tcp") {
			continue
		}
		tcpKeys = append(tcpKeys, k)
	}

	if len(tcpKeys) == 0 {
		return "", fmt.Errorf(
			"container %q has no published TCP ports\n  hint: start it with -p <host_port>:<container_port>",
			name,
		)
	}

	sort.Slice(tcpKeys, func(i, j int) bool {
		return containerPortNum(tcpKeys[i]) < containerPortNum(tcpKeys[j])
	})

	best := tcpKeys[0]
	for _, b := range ports[best] {
		if b.HostPort != "" {
			return b.HostPort, nil
		}
	}

	return "", fmt.Errorf(
		"container %q has no published ports\n  hint: start it with -p <host_port>:<container_port>",
		name,
	)
}

func containerPortNum(key string) int {
	numPart, _, found := strings.Cut(key, "/")
	if !found {
		return math.MaxInt
	}

	n, err := strconv.Atoi(numPart)
	if err != nil {
		return math.MaxInt
	}
	return n
}
