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
	Title     string
	Date      time.Time
	Content   string
	IsDraft   bool
	Filename  string
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
	file, err := os.Open(path)
	if err != nil {
		return Post{}, err
	}
	defer file.Close()

	post := Post{
		Filename: filepath.Base(path),
	}

	scanner := bufio.NewScanner(file)
	inFrontMatter := false
	var content strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		
		if line == "---" {
			if !inFrontMatter {
				inFrontMatter = true
				continue
			} else {
				inFrontMatter = false
				continue
			}
		}

		if inFrontMatter {
			if strings.HasPrefix(line, "title:") {
				post.Title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			} else if strings.HasPrefix(line, "date:") {
				dateStr := strings.TrimSpace(strings.TrimPrefix(line, "date:"))
				post.Date, _ = time.Parse("2006-01-02", dateStr)
			} else if strings.HasPrefix(line, "draft:") {
				post.IsDraft = strings.TrimSpace(strings.TrimPrefix(line, "draft:")) == "true"
			}
		} else {
			content.WriteString(line)
			content.WriteString("\n")
		}
	}

	post.Content = content.String()
	return post, scanner.Err()
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