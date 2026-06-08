package sub

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/deposist/s-ui-x/config"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/middleware"
	"github.com/deposist/s-ui-x/network"
	"github.com/deposist/s-ui-x/service"
	"github.com/deposist/s-ui-x/util/common"

	"github.com/gin-gonic/gin"
)

type Server struct {
	httpServer *http.Server
	listener   net.Listener
	ctx        context.Context
	cancel     context.CancelFunc

	service.SettingService
}

func NewServer() *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		ctx:    ctx,
		cancel: cancel,
	}
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

	subPath, err := s.SettingService.GetSubPath()
	if err != nil {
		return nil, err
	}

	subDomain, err := s.SettingService.GetSubDomain()
	if err != nil {
		return nil, err
	}

	if subDomain != "" {
		engine.Use(middleware.DomainValidator(subDomain))
	}
	engine.Use(middleware.SubSecurityHeaders())

	registeredFormats := map[string]string{}
	if err := rememberSubscriptionPath(registeredFormats, subPath, "link"); err != nil {
		return nil, err
	}
	if err := rememberSubscriptionPath(registeredFormats, joinSubscriptionPath(subPath, "json"), "json"); err != nil {
		return nil, err
	}
	if err := rememberSubscriptionPath(registeredFormats, joinSubscriptionPath(subPath, "clash"), "clash"); err != nil {
		return nil, err
	}

	g := engine.Group(subPath)
	NewSubHandler(g)
	if subPath != "/" {
		if err := registerSubscriptionFormatRoute(engine, registeredFormats, "/json/", "json"); err != nil {
			return nil, err
		}
		if err := registerSubscriptionFormatRoute(engine, registeredFormats, "/clash/", "clash"); err != nil {
			return nil, err
		}
	}
	if err := s.registerCustomFormatRoutes(engine, registeredFormats); err != nil {
		return nil, err
	}

	return engine, nil
}

func (s *Server) registerCustomFormatRoutes(engine *gin.Engine, registered map[string]string) error {
	jsonPath, err := s.SettingService.GetSubJsonPath()
	if err != nil {
		return err
	}
	clashPath, err := s.SettingService.GetSubClashPath()
	if err != nil {
		return err
	}
	if err := registerSubscriptionFormatRoute(engine, registered, jsonPath, "json"); err != nil {
		return err
	}
	return registerSubscriptionFormatRoute(engine, registered, clashPath, "clash")
}

func registerSubscriptionFormatRoute(engine *gin.Engine, registered map[string]string, path string, format string) error {
	path = normalizeSubscriptionRoutePath(path)
	if path == "/" {
		return common.NewError("subscription format path cannot be root")
	}
	if existing, ok := registered[path]; ok {
		if existing == format {
			return nil
		}
		return common.NewError("subscription path conflict: ", path)
	}
	registered[path] = format

	handler := &SubHandler{}
	group := engine.Group(path)
	group.Use(rateLimitMiddleware())
	switch format {
	case "json":
		group.GET("/:subid", handler.json)
		group.HEAD("/:subid", handler.subHeaders)
	case "clash":
		group.GET("/:subid", handler.clash)
		group.HEAD("/:subid", handler.subHeaders)
	default:
		return common.NewError("unknown subscription format: ", format)
	}
	return nil
}

func rememberSubscriptionPath(registered map[string]string, path string, format string) error {
	path = normalizeSubscriptionRoutePath(path)
	if existing, ok := registered[path]; ok && existing != format {
		return common.NewError("subscription path conflict: ", path)
	}
	registered[path] = format
	return nil
}

func joinSubscriptionPath(base string, child string) string {
	if base == "/" {
		return normalizeSubscriptionRoutePath(child)
	}
	return normalizeSubscriptionRoutePath(strings.TrimRight(base, "/") + "/" + strings.Trim(child, "/"))
}

func normalizeSubscriptionRoutePath(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
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

	certFile, err := s.SettingService.GetSubCertFile()
	if err != nil {
		return err
	}
	keyFile, err := s.SettingService.GetSubKeyFile()
	if err != nil {
		return err
	}
	listen, err := s.SettingService.GetSubListen()
	if err != nil {
		return err
	}
	port, err := s.SettingService.GetSubPort()
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
		_ = service.RecordListenFallbackAudit("sub", listenResult.RequestedAddr, listenResult.FallbackAddr, listenResult.BindError)
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
		logger.Info("Sub server run https on", listener.Addr())
	} else {
		logger.Info("Sub server run http on", listener.Addr())
	}
	s.listener = listener

	s.httpServer = &http.Server{
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		if serveErr := s.httpServer.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			logger.Warning("Sub server stopped unexpectedly:", serveErr)
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
