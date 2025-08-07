package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func run() error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "demo-github-mcp",
		Title:   "A demo github mcp server",
		Version: "0.0.1",
	}, nil)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list-repositories",
		Description: "A tool to list all repositories in a Github org",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"name": {
					Type:        "string",
					Description: "GitHub organization name (e.g., kubernetes)",
				},
				"url": {
					Type:        "string",
					Description: "GitHub organization URL (e.g., https://github.com/kubernetes)",
				},
			},
		},
	}, ListRepositories)
	t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), os.Stderr)
	log.Println("🚀 MCP server starting up...")
	if err := server.Run(context.Background(), t); err != nil {
		log.Printf("Server failed: %v", err)
	}
	log.Println("🚀 MCP server shutting down...")
	return nil
}

// User can pass in either the name of the org (example: kubernetes), or its URL (example: https://github.com/kubernetes)
type GithubOrgArgs struct {
	Name string
	URL  string
}

func ListRepositories(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[GithubOrgArgs]) (*mcp.CallToolResultFor[struct{}], error) {
	if params == nil {
		return nil, fmt.Errorf("empty params")
	}

	args := params.Arguments
	if args.Name == "" && args.URL == "" {
		return nil, fmt.Errorf("empty args")
	}
	var apiURL string
	var organization string
	if args.URL != "" {
		// If URL is provided, extract org name and build API URL
		url := strings.TrimPrefix(args.URL, "https://")
		url = strings.TrimPrefix(url, "http://")
		url = strings.TrimPrefix(url, "github.com/")
		url = strings.TrimSuffix(url, "/")

		orgName := strings.Split(url, "/")[0]
		apiURL = fmt.Sprintf("https://api.github.com/orgs/%s/repos", orgName)
		organization = orgName
	} else {
		// Use the provided organization name
		apiURL = fmt.Sprintf("https://api.github.com/orgs/%s/repos", args.Name)
		organization = args.Name
	}
	// apiURL = fmt.Sprintf("%s%s", apiURL, "?per_page=100")
	apiURL = apiURL + "?per_page=100"
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(body))
	}
	type repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
		Private  bool   `json:"private"`
	}

	// Parse the JSON response
	var repositories []repository
	if err := json.NewDecoder(resp.Body).Decode(&repositories); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Repositories for organization %s:", organization))
	for _, repo := range repositories {
		result.WriteString(fmt.Sprintf("Name: %s, URL: %s", repo.Name, repo.HTMLURL))
	}

	return &mcp.CallToolResultFor[struct{}]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result.String()},
		},
	}, nil
}

func main() {
	log.Fatal(run())
}
