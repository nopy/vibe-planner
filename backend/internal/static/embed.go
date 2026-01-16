package static

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var embeddedDist embed.FS

// ServeEmbeddedSPA sets up routes to serve the embedded React SPA
func ServeEmbeddedSPA(router *gin.Engine) error {
	// Extract the dist subdirectory from the embedded FS
	distFS, err := fs.Sub(embeddedDist, "dist")
	if err != nil {
		return err
	}

	// Serve static files (assets with file extensions)
	router.Use(staticFileMiddleware(distFS))

	// NoRoute handler for SPA routing (serves index.html for non-API routes)
	router.NoRoute(spaFallbackHandler(distFS))

	return nil
}

// staticFileMiddleware attempts to serve static files before other handlers
func staticFileMiddleware(distFS fs.FS) gin.HandlerFunc {
	fileServer := http.FileServer(http.FS(distFS))

	return func(c *gin.Context) {
		// Skip API routes
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Next()
			return
		}

		// Skip health endpoints
		if c.Request.URL.Path == "/healthz" || c.Request.URL.Path == "/ready" {
			c.Next()
			return
		}

		// Only handle GET and HEAD requests for static files
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Next()
			return
		}

		// Check if the path has a file extension (likely a static asset)
		ext := path.Ext(c.Request.URL.Path)
		if ext != "" {
			// Try to serve the file
			if fileExists(distFS, strings.TrimPrefix(c.Request.URL.Path, "/")) {
				// Set cache headers for static assets
				setCacheHeaders(c, ext)
				fileServer.ServeHTTP(c.Writer, c.Request)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// spaFallbackHandler serves index.html for all non-API routes (SPA client-side routing)
func spaFallbackHandler(distFS fs.FS) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If we got here and it's an API route, return 404
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		// Serve index.html with no-cache headers (so updates are picked up immediately)
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")

		// Open and serve index.html
		indexFile, err := distFS.Open("index.html")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load application"})
			return
		}
		defer indexFile.Close()

		stat, err := indexFile.Stat()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load application"})
			return
		}

		c.DataFromReader(http.StatusOK, stat.Size(), "text/html; charset=utf-8", indexFile, nil)
	}
}

// fileExists checks if a file exists in the embedded FS
func fileExists(distFS fs.FS, filePath string) bool {
	f, err := distFS.Open(filePath)
	if err != nil {
		return false
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return false
	}

	return !stat.IsDir()
}

// setCacheHeaders sets appropriate cache headers based on file extension
func setCacheHeaders(c *gin.Context, ext string) {
	// Immutable assets (fingerprinted by build tools like Vite)
	immutableExtensions := map[string]bool{
		".js":    true,
		".css":   true,
		".woff":  true,
		".woff2": true,
		".ttf":   true,
		".eot":   true,
		".otf":   true,
	}

	// Long cache for images
	imageExtensions := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".gif":  true,
		".svg":  true,
		".ico":  true,
		".webp": true,
	}

	if immutableExtensions[ext] {
		// Vite adds hashes to JS/CSS filenames, so these are immutable
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else if imageExtensions[ext] {
		// Images get a long cache but not immutable (might be updated)
		c.Header("Cache-Control", "public, max-age=86400") // 1 day
	} else {
		// Other files get shorter cache
		c.Header("Cache-Control", "public, max-age=3600") // 1 hour
	}
}
