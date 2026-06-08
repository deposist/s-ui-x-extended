package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/deposist/s-ui-x/api"
	"github.com/deposist/s-ui-x/config"
	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/middleware"
	"github.com/deposist/s-ui-x/service"
	"github.com/deposist/s-ui-x/web"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func main() {
	log.SetOutput(os.Stdout)
	fmt.Println("phase6 e2e panel-server: init logger")
	logger.Init(logger.LevelWarning)

	fmt.Println("phase6 e2e panel-server: init database")
	if err := database.InitDB(config.GetDBPath()); err != nil {
		log.Fatal(err)
	}
	fmt.Println("phase6 e2e panel-server: load settings")
	if _, err := (&service.SettingService{}).GetAllSetting(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("phase6 e2e panel-server: init api-only web server")
	runtime := service.NewRuntime(nil)
	service.SetDefaultRuntime(runtime)
	settingService := &service.SettingService{}
	baseURL, err := settingService.GetWebPath()
	if err != nil {
		log.Fatal(err)
	}
	cookieKeys, err := settingService.GetCookieKeys()
	if err != nil {
		log.Fatal(err)
	}
	store, err := web.NewSQLiteSessionStore(database.GetDB(), cookieKeys...)
	if err != nil {
		log.Fatal(err)
	}
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())
	engine.Use(middleware.AdminSecurityHeaders(api.RequestIsHTTPS))
	engine.Use(sessions.Sessions("s-ui", store))
	groupAPIV2 := engine.Group(baseURL + "apiv2")
	apiv2 := api.NewAPIv2Handler(groupAPIV2, api.WithRuntime(runtime))
	groupAPI := engine.Group(baseURL + "api")
	api.NewAPIHandler(groupAPI, apiv2, api.WithRuntime(runtime))
	engine.GET(baseURL, func(c *gin.Context) {
		c.String(http.StatusOK, "phase6 e2e panel")
	})
	engine.GET(baseURL+"login", func(c *gin.Context) {
		c.String(http.StatusOK, "phase6 e2e panel")
	})

	port, err := settingService.GetPort()
	if err != nil {
		log.Fatal(err)
	}
	listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		fmt.Printf("phase6 e2e panel-server: listen error: %#v\n", err)
		os.Exit(1)
	}
	server := &http.Server{
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Warning("phase6 e2e api-only server stopped:", err)
		}
	}()
	fmt.Println("phase6 e2e panel-server: ready")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		_ = server.Shutdown(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
	}
}
