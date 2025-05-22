package web

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ionrock/hugs/posts"
	"github.com/ionrock/hugs/templates"
	"github.com/rs/zerolog/log"
)

// Server represents the web server for the Hugo blog editor
type Server struct {
	ContentDir string
	Port       string
}

// New creates a new server instance
func New(contentDir, port string) (*Server, error) {
	// Get absolute path for the content directory
	if contentDir != "" {
		absPath, err := filepath.Abs(contentDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path: %w", err)
		}
		contentDir = filepath.Join(absPath, "content", "post")
	} else {
		// Get the current working directory
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}

		// Use default Hugo blog directory
		contentDir = filepath.Join(wd, "content", "post")

	}

	// Ensure the content directory exists
	if _, err := os.Stat(contentDir); os.IsNotExist(err) {
		log.Error().Str("dir", contentDir).Msg("Content directory not found")
		return nil, err
	}

	// Format port
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	return &Server{
		ContentDir: contentDir,
		Port:       port,
	}, nil
}

// Start starts the web server
func (s *Server) Start() error {
	// Set up routes
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/edit/", s.handleEdit)
	http.HandleFunc("/new", s.handleNew)
	http.HandleFunc("/save", s.handleSave)

	log.Info().Str("content_dir", s.ContentDir).Msg("Using content directory")
	log.Info().Str("address", "http://localhost"+s.Port).Msg("Starting server")
	return http.ListenAndServe(s.Port, nil)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Get all posts
	postList, err := posts.ListPosts(s.ContentDir)
	if err != nil {
		log.Error().Err(err).Msg("Error reading posts")
		http.Error(w, "Error reading posts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Render the template
	component := templates.Index(postList)
	err = component.Render(r.Context(), w)
	if err != nil {
		log.Error().Err(err).Msg("Error rendering index template")
		http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleEdit(w http.ResponseWriter, r *http.Request) {
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
	post, err := posts.ReadPost(filepath.Join(s.ContentDir, filename))
	if err != nil {
		log.Error().Err(err).Str("filename", filename).Msg("Error reading post")
		http.Error(w, "Error reading post: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Render the template
	component := templates.Edit(post)
	err = component.Render(r.Context(), w)
	if err != nil {
		log.Error().Err(err).Str("filename", filename).Msg("Error rendering edit template")
		http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleNew(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		title := r.FormValue("title")
		if title == "" {
			http.Error(w, "Title is required", http.StatusBadRequest)
			return
		}

		post, err := posts.CreateNewPost(s.ContentDir, title)
		if err != nil {
			log.Error().Err(err).Str("title", title).Msg("Error creating new post")
			http.Error(w, "Error creating post: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Info().Str("filename", post.Filename).Msg("Created new post")

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

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {
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
		log.Error().Err(err).Str("filename", filename).Msg("Error extracting title from markdown")
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
	err = posts.SavePost(s.ContentDir, post)
	if err != nil {
		log.Error().Err(err).Str("filename", filename).Msg("Error saving post")
		http.Error(w, "Error saving post: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info().Str("filename", filename).Bool("draft", isDraft).Msg("Post saved")

	// Redirect back to the post list
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
