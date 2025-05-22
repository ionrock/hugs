package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ionrock/hugs/posts"
	"github.com/ionrock/hugs/templates"
	"github.com/urfave/cli/v2"
)

var contentDir string

func main() {
	app := &cli.App{
		Name:  "hugs",
		Usage: "Hugo blog editor server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "port",
				Aliases: []string{"p"},
				Value:   "8080",
				Usage:   "Port to run the server on",
			},
			&cli.StringFlag{
				Name:    "content-dir",
				Aliases: []string{"d"},
				Usage:   "Path to the content/post directory (defaults to ./content/post)",
			},
		},
		Action: runServer,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runServer(c *cli.Context) error {
	var contentDir string

	// Get the content directory
	if c.String("content-dir") != "" {
		// Get absolute path for the content directory
		absPath, err := filepath.Abs(c.String("content-dir"))
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		contentDir = absPath
	}

	if contentDir == "" {
		// Get the current working directory
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Use default Hugo blog directory
		contentDir = filepath.Join(wd, "content", "post")
	}

	// Ensure the content directory exists
	if _, err := os.Stat(contentDir); os.IsNotExist(err) {
		return fmt.Errorf("content directory not found: %s", contentDir)
	}

	// Set up routes
	http.HandleFunc("/", handleIndex)
	http.HandleFunc("/edit/", handleEdit)
	http.HandleFunc("/new", handleNew)
	http.HandleFunc("/save", handleSave)

	// Start the server
	port := c.String("port")
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	log.Printf("Starting server on http://localhost%s", port)
	return http.ListenAndServe(port, nil)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Get all posts
	postList, err := posts.ListPosts(contentDir)
	if err != nil {
		http.Error(w, "Error reading posts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Render the template
	component := templates.Index(postList)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get the filename from the URL
	filename := strings.TrimPrefix(r.URL.Path, "/edit/")
	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	// Read the post
	post, err := posts.ReadPost(filepath.Join(contentDir, filename))
	if err != nil {
		http.Error(w, "Error reading post: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Render the template
	component := templates.Edit(post)
	err = component.Render(r.Context(), w)
	if err != nil {
		http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleNew(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		title := r.FormValue("title")
		if title == "" {
			http.Error(w, "Title is required", http.StatusBadRequest)
			return
		}

		post, err := posts.CreateNewPost(contentDir, title)
		if err != nil {
			http.Error(w, "Error creating post: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Redirect to edit the new post
		http.Redirect(w, r, "/edit/"+post.Filename, http.StatusSeeOther)
		return
	}

	// Show the new post form
	w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>New Post</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            max-width: 800px;
            margin: 0 auto;
            padding: 20px;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 5px;
        }
        input[type="text"] {
            width: 100%;
            padding: 8px;
            font-size: 16px;
        }
        button {
            background: #0070f3;
            color: white;
            border: none;
            padding: 10px 20px;
            border-radius: 4px;
            cursor: pointer;
        }
        .back-link {
            display: inline-block;
            margin-bottom: 20px;
            color: #0070f3;
            text-decoration: none;
        }
    </style>
</head>
<body>
    <a href="/" class="back-link">← Back to posts</a>
    <h1>New Post</h1>
    <form method="POST">
        <div class="form-group">
            <label for="title">Title:</label>
            <input type="text" id="title" name="title" required>
        </div>
        <button type="submit">Create Post</button>
    </form>
</body>
</html>
`))
}

func handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get form values
	filename := r.FormValue("filename")
	content := r.FormValue("content")
	isDraft := r.FormValue("draft") == "on"

	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	// Extract title from the markdown content
	title, err := posts.NewPostFromMarkdown(content)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create post object
	post := posts.Post{
		Title:    title,
		Content:  content,
		IsDraft:  isDraft,
		Filename: filename,
	}

	// Save the post
	err = posts.SavePost(contentDir, post)
	if err != nil {
		http.Error(w, "Error saving post: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Redirect back to the post list
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
