

package main

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"

	"github.com/gomarkdown/markdown"
	"gopkg.in/yaml.v2"
)

type Post struct {
	Title   string   `yaml:"title"`
	Date    string   `yaml:"date"`
	Tags    []string `yaml:"tags"`
	Content string
	Slug    string
	BaseURL string
}

func main() {
	baseURL := os.Getenv("BASE_URL")

	// Create output directory
	if err := os.MkdirAll("output", 0755); err != nil {
		log.Fatal(err)
	}

	// Create content directory if it doesn't exist
	if err := os.MkdirAll("content/posts", 0755); err != nil {
		log.Fatal(err)
	}

	// Read all posts
	posts, err := readPosts("content/posts", baseURL)
	if err != nil {
		log.Fatal(err)
	}

	// Generate individual post pages
	if err := generatePosts(posts, baseURL); err != nil {
		log.Fatal(err)
	}

	// Generate index page
	if err := generateIndex(posts, baseURL); err != nil {
		log.Fatal(err)
	}

	// Generate tag pages
	if err := generateTags(posts, baseURL); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Site generated successfully!")
}

func readPosts(dir string, baseURL string) ([]Post, error) {
	var posts []Post

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			post, err := readPost(fmt.Sprintf("%s/%s", dir, file.Name()), baseURL)
			if err != nil {
				return nil, err
			}
			posts = append(posts, post)
		}
	}

	return posts, nil
}

func readPost(fileName string, baseURL string) (Post, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return Post{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var frontmatter []string
	var content []string
	inFrontmatter := false
	count := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "---") {
			count++
			if count == 1 {
				inFrontmatter = true
				continue
			}
			if count == 2 {
				inFrontmatter = false
				continue
			}
		}

		if inFrontmatter {
			frontmatter = append(frontmatter, line)
		} else {
			content = append(content, line)
		}
	}

	var post Post
	err = yaml.Unmarshal([]byte(strings.Join(frontmatter, "\n")), &post)
	if err != nil {
		return Post{}, err
	}

	post.Content = string(markdown.ToHTML([]byte(strings.Join(content, "\n")), nil, nil))
	post.Slug = strings.TrimSuffix(fileName, ".md")
	post.Slug = strings.TrimPrefix(post.Slug, "content/posts/")
	post.BaseURL = baseURL

	return post, nil
}

func generatePosts(posts []Post, baseURL string) error {
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/post.html")
	if err != nil {
		return err
	}

	for _, post := range posts {
		file, err := os.Create(fmt.Sprintf("output/%s.html", post.Slug))
		if err != nil {
			return err
		}
		defer file.Close()

		err = tmpl.Execute(file, post)
		if err != nil {
			return err
		}
	}

	return nil
}

func generateIndex(posts []Post, baseURL string) error {
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/index.html")
	if err != nil {
		return err
	}

	file, err := os.Create("output/index.html")
	if err != nil {
		return err
	}
	defer file.Close()

	data := struct {
		Posts   []Post
		BaseURL string
	}{
		Posts:   posts,
		BaseURL: baseURL,
	}

	err = tmpl.Execute(file, data)
	if err != nil {
		return err
	}

	return nil
}

func generateTags(posts []Post, baseURL string) error {
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/tag.html")
	if err != nil {
		return err
	}

	tags := make(map[string][]Post)
	for _, post := range posts {
		for _, tag := range post.Tags {
			tags[tag] = append(tags[tag], post)
		}
	}

	for tag, posts := range tags {
		file, err := os.Create(fmt.Sprintf("output/tag-%s.html", tag))
		if err != nil {
			return err
		}
		defer file.Close()

		data := struct {
			Tag     string
			Posts   []Post
			BaseURL string
		}{
			Tag:     tag,
			Posts:   posts,
			BaseURL: baseURL,
		}

		err = tmpl.Execute(file, data)
		if err != nil {
			return err
		}
	}

	return nil
}

