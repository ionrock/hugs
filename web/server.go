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
	// Set up routes with method-specific patterns
	mux := http.NewServeMux()

	// Register routes with HTTP method patterns
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /edit/", s.handleEdit)
	mux.HandleFunc("GET /new", s.handleNew)
	mux.HandleFunc("POST /new", s.handleNew)
	mux.HandleFunc("POST /save", s.handleSave)

	log.Info().Str("content_dir", s.ContentDir).Msg("Using content directory")
	log.Info().Str("address", "http://localhost"+s.Port).Msg("Starting server")
	return http.ListenAndServe(s.Port, mux)
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
	if r.Method == http.MethodPost {
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

	// Render the new post form template
	component := templates.New()
	err := component.Render(r.Context(), w)
	if err != nil {
		log.Error().Err(err).Msg("Error rendering new post template")
		http.Error(w, "Error rendering template: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleSave(w http.ResponseWriter, r *http.Request) {

	// Get form values
	filename := r.FormValue("filename")
	content := r.FormValue("content")

	if filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	log.Debug().
		Str("filename", filename).
		Str("dir", s.ContentDir).
		Msg("Saving post")

	path := filepath.Join(s.ContentDir, filename)
	file, err := os.Create(path)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("Failed to create post file")
		http.Error(w, "Error saving post: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Write the front matter
	_, err = fmt.Fprintf(file, content)

	if err != nil {
		log.Error().Err(err).Str("filename", filename).Msg("Error saving post")
		http.Error(w, "Error saving post: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Info().Str("filename", filename).Msg("Post saved")

	// When we save, we should also commit the changes to the repo. Let's create a new function to commit and use it here. The commit function can use the title of the post to describe the change. For example, "updated post 'My Title'". AI!

	// Redirect back to the post list
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
