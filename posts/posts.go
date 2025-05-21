package posts

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Post struct {
	Title    string
	Date     time.Time
	Content  string
	IsDraft  bool
	Tags     []string
	Filename string
}

// ListPosts returns all posts in the content/post directory
func ListPosts(contentDir string) ([]Post, error) {
	var posts []Post

	files, err := os.ReadDir(contentDir)
	if err != nil {
		return nil, fmt.Errorf("reading posts directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			post, err := ReadPost(filepath.Join(contentDir, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("reading post %s: %w", file.Name(), err)
			}
			posts = append(posts, post)
		}
	}

	return posts, nil
}

// ReadPost reads a post file and parses its front matter
func ReadPost(path string) (Post, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Post{}, fmt.Errorf("reading post: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	inFrontMatter := false
	post := Post{
		Filename: filepath.Base(path),
		Content:  string(content),
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			if !inFrontMatter {
				inFrontMatter = true
				continue
			} else {
				break
			}
		}
		if inFrontMatter {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "title":
				post.Title = value
			case "date":
				dateformats := []string{
					"2006-01-02",
					"2006-01-02 15:04:05",
					"2006-01-02 15:04",
					"2006-01-02T15:04:05Z",
				}
				for _, format := range dateformats {
					date, err := time.Parse(format, value)
					if err != nil {
						continue
					}
					post.Date = date
					break
				}
				if post.Date.IsZero() {
					return Post{}, fmt.Errorf("invalid date format: %v", err)
				}
			case "draft":
				post.IsDraft = value == "true"
			case "tags":
				// Remove brackets if present
				value = strings.Trim(value, "[]")
				if value != "" {
					// Split on commas and trim spaces
					tags := strings.Split(value, ",")
					for i, tag := range tags {
						tags[i] = strings.TrimSpace(tag)
					}
					post.Tags = tags
				}
			}
		}
	}

	if post.Title == "" {
		return Post{}, fmt.Errorf("title not found in content")
	}

	// Create filename from title if not set
	if post.Filename == "" {
		fmt.Println("Creating filename from title")
		slug := strings.ToLower(post.Title)
		slug = strings.ReplaceAll(slug, " ", "-")
		slug = strings.ReplaceAll(slug, "'", "")
		slug = strings.ReplaceAll(slug, "\"", "")
		post.Filename = fmt.Sprintf("%s.md", slug)
	} else {
		fmt.Println("Loaded", post.Filename)
	}

	return post, nil
}

// CreateNewPost creates a new post with the given title
func CreateNewPost(contentDir, title string) (Post, error) {
	now := time.Now()
	post := Post{
		Title:   title,
		Date:    now,
		IsDraft: true,
	}

	// Create the filename from the title
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "'", "")
	slug = strings.ReplaceAll(slug, "\"", "")
	post.Filename = fmt.Sprintf("%s.md", slug)

	// Create the file
	path := filepath.Join(contentDir, post.Filename)
	file, err := os.Create(path)
	if err != nil {
		return Post{}, err
	}
	defer file.Close()

	// Write the front matter
	fmt.Fprintf(file, "---\ntitle: %s\ndate: %s\ndraft: true\n---\n\n", title, now.Format("2006-01-02"))

	return post, nil
}

// SavePost saves a post to disk
func SavePost(contentDir string, post Post) error {
	path := filepath.Join(contentDir, post.Filename)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the front matter
	fmt.Fprintf(file, "---\ntitle: %s\ndate: %s\ndraft: %v\n---\n\n%s",
		post.Title,
		post.Date.Format("2006-01-02"),
		post.IsDraft,
		post.Content)

	return nil
}

// Slug returns the base filename without the .md extension
func (p Post) Slug() string {
	return strings.TrimSuffix(p.Filename, ".md")
}

func NewPostFromMarkdown(content string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	inFrontMatter := false
	var title string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "---" {
			if !inFrontMatter {
				inFrontMatter = true
				continue
			} else {
				break
			}
		}
		if inFrontMatter && strings.HasPrefix(line, "title:") {
			title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			return title, nil
		}
	}

	return "", fmt.Errorf("title not found in content")
}
