package web

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

// serveAssetsFS returns a Gin handler that serves files from assetsFS while
// responding 404 for missing files. This is intentionally
// different from gin.StaticFS, which falls through to NoRoute when a file
// is missing — and our NoRoute serves index.html. After a panel upgrade,
// any browser tab still holding the previous index.html fetches old chunk
// names like /assets/_5nyNEw12.js, those names no longer exist in the
// embedded FS, and a NoRoute fallback served HTML for a JS module request,
// triggering the "Failed to load module script" / "Failed to fetch
// dynamically imported module" errors that broke the Clients tab.
//
// Returning a real 404 lets Vite's preload-error handler in the frontend
// detect the stale chunk and reload index.html cleanly.
func serveAssetsFS(assetsFS fs.FS) gin.HandlerFunc {
	fileServer := http.FileServer(http.FS(assetsFS))
	return func(c *gin.Context) {
		// Trim the configured base path + "assets/" prefix to leave only
		// the in-FS file path. gin's *filepath param already starts with
		// '/', strip it so fs.Stat receives a clean relative path.
		rel := strings.TrimPrefix(c.Param("filepath"), "/")
		clean := path.Clean(rel)
		if clean == "." || clean == "/" || strings.HasPrefix(clean, "..") {
			c.Status(http.StatusNotFound)
			return
		}
		// fs.FS rooted at html/assets does not include the assets/ prefix.
		if _, err := fs.Stat(assetsFS, clean); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		// Rewrite the request path so net/http.FileServer sees only the
		// path under the embedded FS (it does not know about base_url).
		c.Request.URL.Path = "/" + clean
		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}
