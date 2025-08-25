package main

import (
	"fmt"
	"log"
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

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
