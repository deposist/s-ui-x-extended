package web

import (
	"context"
	"crypto/tls"
	"embed"
	"html/template"
	"io"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/api"
	"github.com/deposist/s-ui-x/config"
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/middleware"
	"github.com/deposist/s-ui-x/network"
	"github.com/deposist/s-ui-x/service"

	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

//go:embed *
var content embed.FS

type Server struct {
	httpServer     *http.Server
	listener       net.Listener
	ctx            context.Context
	cancel         context.CancelFunc
	settingService service.SettingService
	runtime        *service.Runtime
	assetsFS       fs.FS
}

type Option func(*Server)

func WithRuntime(runtime *service.Runtime) Option {
	return func(s *Server) {
		s.runtime = runtime
	}
}

func NewServer(options ...Option) (*Server, error) {
	assetsFS, err := fs.Sub(content, "html/assets")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	server := &Server{
		ctx:      ctx,
		cancel:   cancel,
		assetsFS: assetsFS,
	}
	for _, option := range options {
		if option != nil {
			option(server)
		}
	}
	return server, nil
}

func (s *Server) initRouter() (*gin.Engine, error) {
	if config.IsDebug() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.DefaultWriter = io.Discard
		gin.DefaultErrorWriter = io.Discard
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.Default()

	// Load the HTML template
	t := template.New("").Funcs(engine.FuncMap)
	template, err := t.ParseFS(content, "html/index.html")
	if err != nil {
		return nil, err
	}
	engine.SetHTMLTemplate(template)

	base_url, err := s.settingService.GetWebPath()
	if err != nil {
		return nil, err
	}

	webDomain, err := s.settingService.GetWebDomain()
	if err != nil {
		return nil, err
	}

	if webDomain != "" {
		engine.Use(middleware.DomainValidator(webDomain))
	}
	engine.Use(middleware.AdminSecurityHeaders(api.RequestIsHTTPS))

	cookieKeys, err := s.settingService.GetCookieKeys()
	if err != nil {
		return nil, err
	}

	engine.Use(gzip.Gzip(gzip.DefaultCompression))
	assetsBasePath := base_url + "assets/"

	store, err := NewSQLiteSessionStore(database.GetDB(), cookieKeys...)
	if err != nil {
		return nil, err
	}
	engine.Use(sessions.Sessions("s-ui", store))

	engine.Use(func(c *gin.Context) {
		uri := c.Request.RequestURI
		if strings.HasPrefix(uri, assetsBasePath) {
			// Hashed assets are immutable: file name changes whenever
			// content changes.
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
	})

	// Serve the assets folder. We use a custom handler instead of
	// engine.StaticFS so that a missing file responds with 404 directly
	// instead of falling through to NoRoute -> index.html, which made the
	// browser receive HTML for a JS module request after an upgrade and
	// fail with "Failed to load module script: Expected a JavaScript-or-
	// Wasm module script but the server responded with a MIME type of
	// text/html". This was the root cause of the broken Clients tab in
	// upgraded panels: the cached index.html still referenced an old
	// chunk hash that no longer existed in the embedded FS.
	assetsHandler := serveAssetsFS(s.assetsFS)
	engine.GET(assetsBasePath+"*filepath", assetsHandler)
	engine.HEAD(assetsBasePath+"*filepath", assetsHandler)

	group_apiv2 := engine.Group(base_url + "apiv2")
	apiv2 := api.NewAPIv2Handler(group_apiv2, api.WithRuntime(s.runtime))

	group_api := engine.Group(base_url + "api")
	api.NewAPIHandler(group_api, apiv2, api.WithRuntime(s.runtime))

	// Serve index.html as the entry point
	// Handle all other routes by serving index.html
	engine.NoRoute(func(c *gin.Context) {
		if c.Request.URL.Path == strings.TrimSuffix(base_url, "/") {
			c.Redirect(http.StatusTemporaryRedirect, base_url)
			return
		}
		if !strings.HasPrefix(c.Request.URL.Path, base_url) {
			c.String(404, "")
			return
		}
		if c.Request.URL.Path != base_url+"login" && !api.IsLogin(c) {
			c.Redirect(http.StatusTemporaryRedirect, base_url+"login")
			return
		}
		if c.Request.URL.Path == base_url+"login" && api.IsLogin(c) {
			c.Redirect(http.StatusTemporaryRedirect, base_url)
			return
		}
		// index.html must not be cached: it embeds hashed asset URLs
		// and an upgrade rewrites those hashes. A cached index.html
		// after an upgrade would point at chunks that were removed
		// from the embed FS, which is exactly the failure mode that
		// broke the Clients tab in 1.5.x ("Failed to fetch dynamically
		// imported module"). The hashed assets themselves stay
		// immutable; only this entry document needs revalidation.
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.HTML(http.StatusOK, "index.html", gin.H{"BASE_URL": base_url})
	})

	return engine, nil
}

func (s *Server) Start() (err error) {
	//This is an anonymous function, no function name
	defer func() {
		if err != nil {
			_ = s.Stop()
		}
	}()

	engine, err := s.initRouter()
	if err != nil {
		return err
	}

	certFile, err := s.settingService.GetCertFile()
	if err != nil {
		return err
	}
	keyFile, err := s.settingService.GetKeyFile()
	if err != nil {
		return err
	}
	listen, err := s.settingService.GetListen()
	if err != nil {
		return err
	}
	port, err := s.settingService.GetPort()
	if err != nil {
		return err
	}
	listenAddr := net.JoinHostPort(listen, strconv.Itoa(port))
	listenResult, err := network.ListenWithFallbackResult(listenAddr, listen, strconv.Itoa(port))
	if err != nil {
		return err
	}
	listener := listenResult.Listener
	if listenResult.Fallback {
		_ = service.RecordListenFallbackAudit("web", listenResult.RequestedAddr, listenResult.FallbackAddr, listenResult.BindError)
	}
	if certFile != "" || keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			_ = listener.Close()
			return err
		}
		c := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		listener = network.NewAutoHttpsListener(listener)
		listener = tls.NewListener(listener, c)
	}

	if certFile != "" || keyFile != "" {
		logger.Info("web server run https on", listener.Addr())
	} else {
		logger.Info("web server run http on", listener.Addr())
	}
	s.listener = listener

	s.httpServer = &http.Server{
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		// Expose the raw connection so the long-running 3x-ui import handlers
		// can lift the 30s Read/Write timeouts. The gzip middleware wraps
		// c.Writer such that http.NewResponseController can no longer reach the
		// connection, so the deadline must be set on the conn directly.
		ConnContext: api.SaveConnContext,
	}

	go func() {
		if serveErr := s.httpServer.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Warning("web server stopped unexpectedly:", serveErr)
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	var err error
	if s.httpServer != nil {
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
		err = s.httpServer.Shutdown(shutdownCtx)
		cancelShutdown()
		if err != nil {
			s.cancel()
			if s.listener != nil {
				_ = s.listener.Close()
			}
			return err
		}
	} else if s.listener != nil {
		err = s.listener.Close()
		if err != nil {
			s.cancel()
			return err
		}
	}
	s.cancel()
	return nil
}

func (s *Server) GetCtx() context.Context {
	return s.ctx
}
