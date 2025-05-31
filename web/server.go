package web

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
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

// commitChanges commits the saved post to the git repository
func (s *Server) commitChanges(filename, title string) error {
	log.Debug().Str("filename", filename).Str("title", title).Msg("Committing changes to git")

	// Get the repository root directory (parent of content directory)
	repoDir := filepath.Dir(filepath.Dir(s.ContentDir))
	
	// Stage the file
	gitAdd := exec.Command("git", "add", filepath.Join("content", "post", filename))
	gitAdd.Dir = repoDir
	if err := gitAdd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Create commit message
	commitMsg := fmt.Sprintf("Updated post '%s'", title)
	
	// Commit the changes
	gitCommit := exec.Command("git", "commit", "-m", commitMsg)
	gitCommit.Dir = repoDir
	if err := gitCommit.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	log.Info().Str("filename", filename).Str("title", title).Msg("Changes committed to git")
	return nil
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
	mux.HandleFunc("GET /push", s.handlePush)

	log.Info().Str("content_dir", s.ContentDir).Msg("Using content directory")
	log.Info().Str("address", "http://localhost"+s.Port).Msg("Starting server")
	return http.ListenAndServe(s.Port, mux)
}

// hasUnpushedChanges checks if there are commits that haven't been pushed to the remote
func (s *Server) hasUnpushedChanges() bool {
	// Get the repository root directory (parent of content directory)
	repoDir := filepath.Dir(filepath.Dir(s.ContentDir))
	
	// Check if there are unpushed commits
	// git log @{u}..HEAD will list commits that are in HEAD but not in the upstream branch
	cmd := exec.Command("git", "log", "@{u}..HEAD", "--oneline")
	cmd.Dir = repoDir
	
	var out bytes.Buffer
	cmd.Stdout = &out
	
	// If there's an error, it could be because there's no upstream branch
	// In that case, we'll assume there are changes to push
	if err := cmd.Run(); err != nil {
		log.Debug().Err(err).Msg("Error checking for unpushed changes, assuming changes exist")
		return true
	}
	
	// If the output is not empty, there are unpushed commits
	return out.String() != ""
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
	
	// Check if there are unpushed changes
	hasChanges := s.hasUnpushedChanges()
	log.Debug().Bool("has_unpushed_changes", hasChanges).Msg("Checked for unpushed changes")

	// Render the template
	component := templates.Index(postList, hasChanges)
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

func (s *Server) handlePush(w http.ResponseWriter, r *http.Request) {
	log.Info().Msg("Pushing changes to remote repository")

	// Get the repository root directory (parent of content directory)
	repoDir := filepath.Dir(filepath.Dir(s.ContentDir))
	
	// Execute git push
	gitPush := exec.Command("git", "push")
	gitPush.Dir = repoDir
	
	if err := gitPush.Run(); err != nil {
		log.Error().Err(err).Msg("Failed to push changes to remote repository")
		http.Error(w, "Error pushing changes: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	log.Info().Msg("Successfully pushed changes to remote repository")
	
	// Redirect back to the post list
	http.Redirect(w, r, "/", http.StatusSeeOther)
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

	// Extract post title from content for commit message
	title, err := posts.NewPostFromMarkdown(content)
	if err != nil {
		log.Warn().Err(err).Msg("Could not extract title for commit message")
		title = filename // Fallback to using filename if title extraction fails
	}

	// Commit the changes to git
	if err := s.commitChanges(filename, title); err != nil {
		log.Warn().Err(err).Msg("Failed to commit changes to git")
	}

	// Redirect back to the post list
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
