package core

import (
	"context"
	"net"
	"net/http"
	"net/http/pprof"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/byteformats"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/common/json/badjson"

	"github.com/go-chi/chi/v5"
)

// startDebugHTTPServer starts the pprof/debug HTTP server when
// experimental.debug.listen is set. It is opt-in and binds only to the
// operator-configured address; default configs leave it disabled.
func startDebugHTTPServer(options option.DebugOptions) (*http.Server, error) {
	if options.Listen == "" {
		return nil, nil
	}
	r := chi.NewMux()
	r.Route("/debug", func(r chi.Router) {
		r.Get("/gc", func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusNoContent)
			go debug.FreeOSMemory()
		})
		r.Get("/memory", func(writer http.ResponseWriter, request *http.Request) {
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			var memObject badjson.JSONObject
			memObject.Put("heap", byteformats.FormatMemoryBytes(memStats.HeapInuse))
			memObject.Put("stack", byteformats.FormatMemoryBytes(memStats.StackInuse))
			memObject.Put("idle", byteformats.FormatMemoryBytes(memStats.HeapIdle-memStats.HeapReleased))
			memObject.Put("goroutines", runtime.NumGoroutine())
			memObject.Put("rss", rusageMaxRSS())

			encoder := json.NewEncoder(writer)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(&memObject); err != nil {
				log.Debug(E.Cause(err, "encode memory stats"))
			}
		})
		r.Route("/pprof", func(r chi.Router) {
			r.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
				if !strings.HasSuffix(request.URL.Path, "/") {
					// pprof resolves its assets relative to a trailing slash. Append it
					// in place instead of redirecting: a redirect target built from the
					// request path is an open-redirect surface.
					request.URL.Path += "/"
				}
				pprof.Index(writer, request)
			})
			r.HandleFunc("/*", pprof.Index)
			r.HandleFunc("/cmdline", pprof.Cmdline)
			r.HandleFunc("/profile", pprof.Profile)
			r.HandleFunc("/symbol", pprof.Symbol)
			r.HandleFunc("/trace", pprof.Trace)
		})
	})
	server := &http.Server{
		Addr:              options.Listen,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", options.Listen)
	if err != nil {
		return nil, E.Cause(err, "listen debug HTTP server")
	}
	go func() {
		err := server.Serve(listener)
		if err != nil && !E.IsClosed(err) {
			log.Error(E.Cause(err, "serve debug HTTP server"))
		}
	}()
	return server, nil
}
