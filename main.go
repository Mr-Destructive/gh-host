package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "gh-host",
		Usage: "Manage blog posts from the command line",
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: "Create a new post",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "title",
						Usage:    "The title of the post",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "content",
						Usage:    "The content of the post in Markdown",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "tags",
						Usage: "Comma-separated tags",
					},
					&cli.StringFlag{
						Name:  "date",
						Usage: "The date of the post in YYYY-MM-DD format",
					},
				},
				Action: func(c *cli.Context) error {
					title := c.String("title")
					content := c.String("content")
					tags := c.String("tags")
					date := c.String("date")

					if date == "" {
						date = time.Now().Format("2006-01-02")
					}

					slug := strings.ToLower(strings.ReplaceAll(title, " ", "-"))

					if err := os.MkdirAll("content/posts", 0755); err != nil {
						return err
					}

					fileName := fmt.Sprintf("content/posts/%s.md", slug)
					file, err := os.Create(fileName)
					if err != nil {
						return err
					}
					defer file.Close()

					file.WriteString(fmt.Sprintf(`---
title: %s
date: %s
tags: [%s]
---

%s`, title, date, tags, content))

					fmt.Printf("Created post: %s\n", fileName)

					return nil
				},
			},
			{
				Name:	"delete",
				Usage: "Delete a post",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:	 "slug",
						Usage:	 "The slug of the post to delete",
						Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					slug := c.String("slug")

					fileName := fmt.Sprintf("content/posts/%s.md", slug)
					if err := os.Remove(fileName); err != nil {
						return err
					}

					fmt.Printf("Deleted post: %s\n", fileName)

					return nil
				},
			},
			{
				Name:  "update",
				Usage: "Update a post",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "slug",
						Usage:    "The slug of the post to update",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "title",
						Usage: "The new title of the post",
					},
					&cli.StringFlag{
						Name:  "content",
						Usage: "The new content of the post in Markdown",
					},
					&cli.StringFlag{
						Name:  "tags",
						Usage: "The new comma-separated tags",
					},
				},
				Action: func(c *cli.Context) error {
					slug := c.String("slug")
					title := c.String("title")
					content := c.String("content")
					tags := c.String("tags")

					fileName := fmt.Sprintf("content/posts/%s.md", slug)
					file, err := os.ReadFile(fileName)
					if err != nil {
						return err
					}

					lines := strings.Split(string(file), "\n")
					var newLines []string
					inFrontmatter := false

					for _, line := range lines {
						if strings.HasPrefix(line, "---") {
							inFrontmatter = !inFrontmatter
							newLines = append(newLines, line)
							continue
						}

						if inFrontmatter {
							if strings.HasPrefix(line, "title:") && title != "" {
								line = fmt.Sprintf("title: %s", title)
							} else if strings.HasPrefix(line, "tags:") && tags != "" {
								line = fmt.Sprintf("tags: [%s]", tags)
							}
						} else if content != "" {
							// This will replace the entire content of the file after the frontmatter
							// A better implementation would be to find the content section and replace it
							newLines = append(newLines, content)
							break
						}

						newLines = append(newLines, line)
					}

					output := strings.Join(newLines, "\n")
					err = os.WriteFile(fileName, []byte(output), 0644)
					if err != nil {
						return err
					}

					fmt.Printf("Updated post: %s\n", fileName)

					return nil
				},
			},
		},
	}

	},
		},
		{
			Name:  "serve",
			Usage: "Start the HTTP server to dispatch workflows",
			Action: func(c *cli.Context) error {
				http.HandleFunc("/dispatch-workflow", dispatchWorkflowHandler)
				port := os.Getenv("PORT")
				if port == "" {
					port = "8080"
				}
				log.Printf("Server listening on :%s", port)
				return http.ListenAndServe(":"+port, nil)
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func dispatchWorkflowHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var data struct {
		Title    string `json:"title"`
		Content  string `json:"content"`
		Tags     string `json:"tags"`
		Slug     string `json:"slug"`
		Workflow string `json:"workflow"`
		Secret   string `json:"secret"`
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	// Validate secret
	expectedSecret := os.Getenv("GH_HOST_SECRET")
	if expectedSecret == "" {
		log.Println("GH_HOST_SECRET environment variable not set.")
		http.Error(w, "Server configuration error: GH_HOST_SECRET not set", http.StatusInternalServerError)
		return
	}
	if data.Secret != expectedSecret {
		http.Error(w, "Invalid secret", http.StatusUnauthorized)
		return
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Println("GITHUB_TOKEN environment variable not set.")
		http.Error(w, "Server configuration error: GITHUB_TOKEN not set", http.StatusInternalServerError)
		return
	}

	repo := os.Getenv("GITHUB_REPOSITORY") // e.g., "owner/repo"
	if repo == "" {
		log.Println("GITHUB_REPOSITORY environment variable not set.")
		http.Error(w, "Server configuration error: GITHUB_REPOSITORY not set", http.StatusInternalServerError)
		return
	}

	owner := strings.Split(repo, "/")[0]
	repoName := strings.Split(repo, "/")[1]

	eventType := strings.TrimSuffix(data.Workflow, ".yml") // e.g., "create-post"

	clientPayload := map[string]string{
		"title":   data.Title,
		"content": data.Content,
		"tags":    data.Tags,
		"slug":    data.Slug,
		"secret":  data.Secret, // Pass secret for workflow validation if needed
	}

	payloadBytes, err := json.Marshal(clientPayload)
	if err != nil {
		http.Error(w, "Error marshalling client payload", http.StatusInternalServerError)
		return
	}

	githubAPIURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/dispatches", owner, repoName)
	req, err := http.NewRequest("POST", githubAPIURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		http.Error(w, "Error creating GitHub API request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error sending request to GitHub API", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		respBody, _ := io.ReadAll(resp.Body)
		log.Printf("GitHub API error: %d - %s", resp.StatusCode, string(respBody))
		http.Error(w, fmt.Sprintf("Failed to dispatch workflow: %s", string(respBody)), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Workflow triggered successfully!"))
}
