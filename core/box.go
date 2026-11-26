package core

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/util/common"

	"github.com/sagernet/sing-box/adapter"
	boxCertificate "github.com/sagernet/sing-box/adapter/certificate"
	"github.com/sagernet/sing-box/adapter/endpoint"
	"github.com/sagernet/sing-box/adapter/inbound"
	"github.com/sagernet/sing-box/adapter/outbound"
	"github.com/sagernet/sing-box/adapter/provider"
	boxService "github.com/sagernet/sing-box/adapter/service"
	"github.com/sagernet/sing-box/common/certificate"
	"github.com/sagernet/sing-box/common/dialer"
	"github.com/sagernet/sing-box/common/httpclient"
	"github.com/sagernet/sing-box/common/netns"
	"github.com/sagernet/sing-box/common/taskmonitor"
	"github.com/sagernet/sing-box/common/tls"
	"github.com/sagernet/sing-box/common/trafficcontrol"
	"github.com/sagernet/sing-box/common/urltest"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/dns"
	"github.com/sagernet/sing-box/experimental"
	"github.com/sagernet/sing-box/experimental/cachefile"
	"github.com/sagernet/sing-box/experimental/clashmode"
	"github.com/sagernet/sing-box/experimental/deprecated"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-box/protocol/direct"
	"github.com/sagernet/sing-box/route"
	sbCommon "github.com/sagernet/sing/common"
	F "github.com/sagernet/sing/common/format"
	"github.com/sagernet/sing/common/ntp"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/pause"
)

var _ adapter.SimpleLifecycle = (*Box)(nil)

type Box struct {
	ctx                 context.Context
	createdAt           time.Time
	debugOptions        option.DebugOptions
	logFactory          log.Factory
	logger              log.ContextLogger
	network             *route.NetworkManager
	endpoint            *endpoint.Manager
	inbound             *inbound.Manager
	outbound            *outbound.Manager
	provider            *provider.Manager
	service             *boxService.Manager
	certificateProvider *boxCertificate.Manager
	dnsTransport        *dns.TransportManager
	dnsRouter           *dns.Router
	connection          *route.ConnectionManager
	router              *route.Router
	httpClientService   adapter.LifecycleService
	internalService     []adapter.LifecycleService
	referenceManager    *route.ReferenceManager
	ntpService          *ntp.Service
	scope               *adapter.Scope
	logFactoryInScope   bool
	statsTracker        *StatsTracker
	connTracker         *ConnTracker
	done                chan struct{}
	closeOnce           sync.Once
	closeErr            error
}

type Options struct {
	option.Options
	Context context.Context
}

func Context(
	ctx context.Context,
	inboundRegistry adapter.InboundRegistry,
	outboundRegistry adapter.OutboundRegistry,
	endpointRegistry adapter.EndpointRegistry,
	providerRegistry adapter.ProviderRegistry,
	dnsTransportRegistry adapter.DNSTransportRegistry,
	serviceRegistry adapter.ServiceRegistry,
	certificateProviderRegistry adapter.CertificateProviderRegistry,
) context.Context {
	if service.FromContext[option.InboundOptionsRegistry](ctx) == nil ||
		service.FromContext[adapter.InboundRegistry](ctx) == nil {
		ctx = service.ContextWith[option.InboundOptionsRegistry](ctx, inboundRegistry)
		ctx = service.ContextWith[adapter.InboundRegistry](ctx, inboundRegistry)
	}
	if service.FromContext[option.OutboundOptionsRegistry](ctx) == nil ||
		service.FromContext[adapter.OutboundRegistry](ctx) == nil {
		ctx = service.ContextWith[option.OutboundOptionsRegistry](ctx, outboundRegistry)
		ctx = service.ContextWith[adapter.OutboundRegistry](ctx, outboundRegistry)
	}
	if service.FromContext[option.EndpointOptionsRegistry](ctx) == nil ||
		service.FromContext[adapter.EndpointRegistry](ctx) == nil {
		ctx = service.ContextWith[option.EndpointOptionsRegistry](ctx, endpointRegistry)
		ctx = service.ContextWith[adapter.EndpointRegistry](ctx, endpointRegistry)
	}
	if service.FromContext[option.ProviderOptionsRegistry](ctx) == nil ||
		service.FromContext[adapter.ProviderRegistry](ctx) == nil {
		ctx = service.ContextWith[option.ProviderOptionsRegistry](ctx, providerRegistry)
		ctx = service.ContextWith[adapter.ProviderRegistry](ctx, providerRegistry)
	}
	if service.FromContext[adapter.DNSTransportRegistry](ctx) == nil {
		ctx = service.ContextWith[option.DNSTransportOptionsRegistry](ctx, dnsTransportRegistry)
		ctx = service.ContextWith[adapter.DNSTransportRegistry](ctx, dnsTransportRegistry)
	}
	if service.FromContext[adapter.ServiceRegistry](ctx) == nil {
		ctx = service.ContextWith[option.ServiceOptionsRegistry](ctx, serviceRegistry)
		ctx = service.ContextWith[adapter.ServiceRegistry](ctx, serviceRegistry)
	}
	if service.FromContext[adapter.CertificateProviderRegistry](ctx) == nil {
		ctx = service.ContextWith[option.CertificateProviderOptionsRegistry](ctx, certificateProviderRegistry)
		ctx = service.ContextWith[adapter.CertificateProviderRegistry](ctx, certificateProviderRegistry)
	}
	return ctx
}

func NewBox(options Options) (*Box, error) {
	var err error
	createdAt := time.Now()
	ctx := options.Context
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = service.ContextWithDefaultRegistry(ctx)

	endpointRegistry := service.FromContext[adapter.EndpointRegistry](ctx)
	inboundRegistry := service.FromContext[adapter.InboundRegistry](ctx)
	outboundRegistry := service.FromContext[adapter.OutboundRegistry](ctx)
	dnsTransportRegistry := service.FromContext[adapter.DNSTransportRegistry](ctx)
	providerRegistry := service.FromContext[adapter.ProviderRegistry](ctx)
	serviceRegistry := service.FromContext[adapter.ServiceRegistry](ctx)
	certificateProviderRegistry := service.FromContext[adapter.CertificateProviderRegistry](ctx)

	if endpointRegistry == nil {
		return nil, common.NewError("missing endpoint registry in context")
	}
	if inboundRegistry == nil {
		return nil, common.NewError("missing inbound registry in context")
	}
	if outboundRegistry == nil {
		return nil, common.NewError("missing outbound registry in context")
	}
	if dnsTransportRegistry == nil {
		return nil, common.NewError("missing DNS transport registry in context")
	}
	if providerRegistry == nil {
		return nil, common.NewError("missing provider registry in context")
	}
	if serviceRegistry == nil {
		return nil, common.NewError("missing service registry in context")
	}
	if certificateProviderRegistry == nil {
		return nil, common.NewError("missing certificate provider registry in context")
	}

	ctx = pause.WithDefaultManager(ctx)
	experimentalOptions := sbCommon.PtrValueOrDefault(options.Experimental)
	debugOptions := sbCommon.PtrValueOrDefault(experimentalOptions.Debug)
	err = checkDebugOptions(debugOptions)
	if err != nil {
		return nil, err
	}
	var needCacheFile bool
	var needClashAPI bool
	var needV2RayAPI bool
	if experimentalOptions.CacheFile != nil && experimentalOptions.CacheFile.Enabled {
		needCacheFile = true
	}
	if experimentalOptions.ClashAPI != nil {
		needClashAPI = true
	}
	if experimentalOptions.V2RayAPI != nil && experimentalOptions.V2RayAPI.Listen != "" {
		needV2RayAPI = true
	}
	if experimentalOptions.UnifiedDelay != nil && experimentalOptions.UnifiedDelay.Enabled {
		ctx = urltest.ContextWithIsUnifiedDelay(ctx)
	}
	needAPIService := sbCommon.Any(options.Services, func(it option.Service) bool {
		return it.Type == C.TypeAPI
	})
	if service.PtrFromContext[urltest.HistoryStorage](ctx) == nil {
		ctx = service.ContextWithPtr(ctx, urltest.NewHistoryStorage())
	}
	platformInterface := service.FromContext[adapter.PlatformInterface](ctx)
	var defaultLogWriter io.Writer
	if platformInterface != nil {
		defaultLogWriter = io.Discard
	}
	var logFactory log.Factory
	logFactory, err = NewFactory(log.Options{
		Context:       ctx,
		Options:       sbCommon.PtrValueOrDefault(options.Log),
		Observable:    needClashAPI || needAPIService,
		DefaultWriter: defaultLogWriter,
		BaseTime:      createdAt,
	})
	if err != nil {
		return nil, common.NewError("create log factory", err)
	}
	service.MustRegister[log.Factory](ctx, logFactory)

	var internalServices []adapter.LifecycleService
	routeOptions := sbCommon.PtrValueOrDefault(options.Route)
	certificateOptions := sbCommon.PtrValueOrDefault(options.Certificate)
	if C.IsAndroid || certificateOptions.Store != "" && certificateOptions.Store != C.CertificateStoreSystem ||
		len(certificateOptions.Certificate) > 0 ||
		len(certificateOptions.CertificatePath) > 0 ||
		len(certificateOptions.CertificateDirectoryPath) > 0 {
		certificateStore, err := certificate.NewStore(ctx, logFactory.NewLogger("certificate"), certificateOptions)
		if err != nil {
			return nil, err
		}
		service.MustRegister[adapter.CertificateStore](ctx, certificateStore)
		internalServices = append(internalServices, certificateStore)
	}
	netnsManager, err := netns.NewManager(logFactory.NewLogger("netns"), options.NetworkNamespaces, nil)
	if err != nil {
		return nil, err
	}
	service.MustRegister[adapter.NetworkNamespaceManager](ctx, netnsManager)
	internalServices = append(internalServices, netnsManager)

	dnsOptions := sbCommon.PtrValueOrDefault(options.DNS)
	endpointManager := endpoint.NewManager(endpointRegistry)
	inboundManager := inbound.NewManager(inboundRegistry, endpointManager)
	outboundManager := outbound.NewManager(outboundRegistry, endpointManager, routeOptions.Final)
	providerManager := provider.NewManager(providerRegistry)
	dnsTransportManager := dns.NewTransportManager(dnsTransportRegistry, outboundManager, dnsOptions.Final)
	serviceManager := boxService.NewManager(serviceRegistry)
	certificateProviderManager := boxCertificate.NewManager(certificateProviderRegistry)
	service.MustRegister[adapter.EndpointManager](ctx, endpointManager)
	service.MustRegister[adapter.InboundManager](ctx, inboundManager)
	service.MustRegister[adapter.OutboundManager](ctx, outboundManager)
	service.MustRegister[adapter.ProviderManager](ctx, providerManager)
	service.MustRegister[adapter.DNSTransportManager](ctx, dnsTransportManager)
	service.MustRegister[adapter.ServiceManager](ctx, serviceManager)
	service.MustRegister[adapter.CertificateProviderManager](ctx, certificateProviderManager)

	dnsRouter, err := dns.NewRouter(ctx, logFactory, dnsOptions)
	if err != nil {
		return nil, common.NewError("initialize DNS router", err)
	}
	service.MustRegister[adapter.DNSRouter](ctx, dnsRouter)
	service.MustRegister[adapter.DNSRuleSetUpdateValidator](ctx, dnsRouter)
	connectionManager := route.NewConnectionManager(ctx, logFactory.NewLogger("connection"))
	service.MustRegister[adapter.ConnectionManager](ctx, connectionManager)
	networkManager, err := route.NewNetworkManager(ctx, logFactory.NewLogger("network"), routeOptions, dnsOptions)
	if err != nil {
		return nil, common.NewError("initialize network manager", err)
	}
	service.MustRegister[adapter.NetworkManager](ctx, networkManager)
	// Must register after ConnectionManager: the default HTTP client resolution
	// reads it from the context.
	httpClientManager := httpclient.NewManager(ctx, logFactory.NewLogger("httpclient"), options.HTTPClients, routeOptions.DefaultHTTPClient)
	service.MustRegister[adapter.HTTPClientManager](ctx, httpClientManager)
	httpClientService := adapter.LifecycleService(httpClientManager)

	router := route.NewRouter(ctx, logFactory, "router", routeOptions, dnsOptions)
	service.MustRegister[adapter.Router](ctx, router)
	err = router.Initialize(routeOptions.Rules, routeOptions.RuleSet)
	if err != nil {
		return nil, common.NewError("initialize router", err)
	}
	if needClashAPI || needAPIService {
		trafficManager := trafficcontrol.NewManager()
		service.MustRegisterPtr(ctx, trafficManager)
		router.AppendTracker(trafficManager)
		internalServices = append(internalServices, trafficManager)
		var clashDefaultMode string
		if experimentalOptions.ClashAPI != nil {
			clashDefaultMode = experimentalOptions.ClashAPI.DefaultMode
		}
		clashMode := clashmode.NewManager(ctx, logFactory.NewLogger("clash-mode"), clashDefaultMode, clashmode.CalculateModeList(options.Options))
		service.MustRegisterPtr(ctx, clashMode)
		internalServices = append(internalServices, clashMode)
	}
	referenceManager := route.NewReferenceManager(ctx, logFactory.NewLogger("reference"), options.Options)
	internalServices = append(internalServices, referenceManager)

	ntpOptions := sbCommon.PtrValueOrDefault(options.NTP)
	var timeService *tls.TimeServiceWrapper
	if ntpOptions.Enabled {
		timeService = new(tls.TimeServiceWrapper)
		service.MustRegister[ntp.TimeService](ctx, timeService)
	}
	for i, transportOptions := range dnsOptions.Servers {
		var tag string
		if transportOptions.Tag != "" {
			tag = transportOptions.Tag
		} else {
			tag = F.ToString(i)
		}
		err = dnsTransportManager.Create(
			ctx,
			logFactory.NewLogger(F.ToString("dns/", transportOptions.Type, "[", tag, "]")),
			tag,
			transportOptions.Type,
			transportOptions.Options,
		)
		if err != nil {
			return nil, common.NewError("initialize DNS server[", i, "]", err)
		}
	}
	err = dnsRouter.Initialize(dnsOptions.Rules)
	if err != nil {
		return nil, common.NewError("initialize dns router", err)
	}
	for i, endpointOptions := range options.Endpoints {
		var tag string
		if endpointOptions.Tag != "" {
			tag = endpointOptions.Tag
		} else {
			tag = F.ToString(i)
		}
		endpointCtx := ctx
		if tag != "" {
			// Контекст с тегом для endpoint
			endpointCtx = adapter.WithContext(endpointCtx, &adapter.InboundContext{
				Outbound: tag,
			})
		}
		err = endpointManager.Create(
			endpointCtx,
			router,
			logFactory.NewLogger(F.ToString("endpoint/", endpointOptions.Type, "[", tag, "]")),
			tag,
			endpointOptions.Type,
			endpointOptions.Options,
		)
		if err != nil {
			return nil, common.NewError("initialize endpoint["+F.ToString(i)+"] "+tag, err)
		}
	}
	for i, inboundOptions := range options.Inbounds {
		var tag string
		if inboundOptions.Tag != "" {
			tag = inboundOptions.Tag
		} else {
			tag = F.ToString(i)
		}
		err = inboundManager.Create(
			ctx,
			router,
			logFactory.NewLogger(F.ToString("inbound/", inboundOptions.Type, "[", tag, "]")),
			tag,
			inboundOptions.Type,
			inboundOptions.Options,
		)
		if err != nil {
			return nil, common.NewError("initialize inbound[", i, "] ", tag, err)
		}
	}
	options.Outbounds = append(options.Outbounds, option.Outbound{
		Tag:  "Compatible",
		Type: C.TypeDirect,
	})
	for i, serviceOptions := range options.Services {
		var tag string
		if serviceOptions.Tag != "" {
			tag = serviceOptions.Tag
		} else {
			tag = F.ToString(i)
		}
		err = serviceManager.Create(
			ctx,
			logFactory.NewLogger(F.ToString("service/", serviceOptions.Type, "[", tag, "]")),
			tag,
			serviceOptions.Type,
			serviceOptions.Options,
		)
		if err != nil {
			return nil, common.NewError("initialize service["+F.ToString(i)+"]"+tag, err)
		}
	}
	for i, outboundOptions := range options.Outbounds {
		var tag string
		if outboundOptions.Tag != "" {
			tag = outboundOptions.Tag
		} else {
			tag = F.ToString(i)
		}
		outboundCtx := ctx
		if tag != "" {
			// Контекст с тегом для outbound
			outboundCtx = adapter.WithContext(outboundCtx, &adapter.InboundContext{
				Outbound: tag,
			})
		}
		err = outboundManager.Create(
			outboundCtx,
			router,
			logFactory.NewLogger(F.ToString("outbound/", outboundOptions.Type, "[", tag, "]")),
			tag,
			outboundOptions.Type,
			outboundOptions.Options,
		)
		if err != nil {
			return nil, common.NewError("initialize outbound["+F.ToString(i)+"] "+tag, err)
		}
	}
	for i, providerOptions := range options.Providers {
		var tag string
		if providerOptions.Tag != "" {
			tag = providerOptions.Tag
		} else {
			tag = F.ToString(i)
		}
		err = providerManager.Create(
			ctx,
			router,
			logFactory,
			tag,
			providerOptions.Type,
			providerOptions.Options,
		)
		if err != nil {
			return nil, common.NewError("initialize provider["+F.ToString(i)+"] "+tag, err)
		}
	}
	for i, certificateProviderOptions := range options.CertificateProviders {
		var tag string
		if certificateProviderOptions.Tag != "" {
			tag = certificateProviderOptions.Tag
		} else {
			tag = F.ToString(i)
		}
		err = certificateProviderManager.Create(
			ctx,
			logFactory.NewLogger(F.ToString("certificate-provider/", certificateProviderOptions.Type, "[", tag, "]")),
			tag,
			certificateProviderOptions.Type,
			certificateProviderOptions.Options,
		)
		if err != nil {
			return nil, common.NewError("initialize certificate provider[", i, "]", err)
		}
	}
	outboundManager.Initialize(func() (adapter.Outbound, error) {
		return direct.NewOutbound(
			ctx,
			router,
			logFactory.NewLogger("outbound/direct"),
			"direct",
			option.DirectOutboundOptions{},
		)
	})
	dnsTransportManager.Initialize(func() (adapter.DNSTransport, error) {
		return dnsTransportRegistry.CreateDNSTransport(
			ctx,
			logFactory.NewLogger("dns/local"),
			"local",
			C.DNSTypeLocal,
			&option.LocalDNSServerOptions{},
		)
	})
	httpClientManager.Initialize(func() (*httpclient.ManagedTransport, error) {
		deprecated.Report(ctx, deprecated.OptionImplicitDefaultHTTPClient)
		var httpClientOptions option.HTTPClientOptions
		httpClientOptions.DefaultOutbound = true
		return httpclient.NewTransport(ctx, logFactory.NewLogger("httpclient"), "", httpClientOptions)
	})
	if platformInterface != nil {
		err = platformInterface.Initialize(networkManager)
		if err != nil {
			return nil, common.NewError("initialize platform interface", err)
		}
	}
	statsTracker := NewStatsTracker()
	connTracker := NewConnTracker()
	router.AppendTracker(statsTracker)
	router.AppendTracker(connTracker)

	if needCacheFile {
		cacheFile := cachefile.New(ctx, logFactory.NewLogger("cache-file"), sbCommon.PtrValueOrDefault(experimentalOptions.CacheFile))
		service.MustRegister[adapter.CacheFile](ctx, cacheFile)
		internalServices = append(internalServices, cacheFile)
	}
	if needClashAPI {
		clashAPIOptions := sbCommon.PtrValueOrDefault(experimentalOptions.ClashAPI)
		clashServer, err := experimental.NewClashServer(ctx, logFactory.(log.ObservableFactory), clashAPIOptions)
		if err != nil {
			return nil, common.NewError(err, "create clash-server")
		}
		internalServices = append(internalServices, clashServer)
	}
	if needV2RayAPI {
		v2rayServer, err := experimental.NewV2RayServer(logFactory.NewLogger("v2ray-api"), sbCommon.PtrValueOrDefault(experimentalOptions.V2RayAPI))
		if err != nil {
			return nil, common.NewError(err, "create v2ray-server")
		}
		if v2rayServer.StatsService() != nil {
			router.AppendTracker(v2rayServer.StatsService())
			internalServices = append(internalServices, v2rayServer)
			service.MustRegister[adapter.V2RayServer](ctx, v2rayServer)
		}
	}
	var ntpService *ntp.Service
	if ntpOptions.Enabled {
		if ntpOptions.WriteToSystem {
			err = adapter.CheckSecurityFeature(ctx, "NTP `write_to_system`")
			if err != nil {
				return nil, err
			}
		}
		ntpDialer, err := dialer.New(ctx, ntpOptions.DialerOptions, ntpOptions.ServerIsDomain())
		if err != nil {
			return nil, common.NewError(err, "create NTP service")
		}
		ntpService = ntp.NewService(ntp.Options{
			Context:       ctx,
			Dialer:        ntpDialer,
			Logger:        logFactory.NewLogger("ntp"),
			Server:        ntpOptions.ServerOptions.Build(),
			Interval:      time.Duration(ntpOptions.Interval),
			WriteToSystem: ntpOptions.WriteToSystem,
		})
		timeService.TimeService = ntpService
	}
	return &Box{
		ctx:                 ctx,
		network:             networkManager,
		endpoint:            endpointManager,
		inbound:             inboundManager,
		outbound:            outboundManager,
		provider:            providerManager,
		service:             serviceManager,
		certificateProvider: certificateProviderManager,
		dnsTransport:        dnsTransportManager,
		dnsRouter:           dnsRouter,
		connection:          connectionManager,
		router:              router,
		httpClientService:   httpClientService,
		createdAt:           createdAt,
		debugOptions:        debugOptions,
		logFactory:          logFactory,
		logger:              logFactory.Logger(),
		internalService:     internalServices,
		referenceManager:    referenceManager,
		ntpService:          ntpService,
		scope:               adapter.NewScope(ctx, logFactory.Logger()),
		statsTracker:        statsTracker,
		connTracker:         connTracker,
		done:                make(chan struct{}),
	}, nil
}

func (s *Box) PreStart() error {
	err := s.preStart()
	if err != nil {
		_ = s.Close()
		return err
	}
	s.logger.Info("sing-box pre-started (", F.Seconds(time.Since(s.createdAt).Seconds()), "s)")
	return nil
}

func (s *Box) Start() error {
	err := s.start()
	if err != nil {
		return errors.Join(err, s.Close())
	}
	s.logger.Info("sing-box started (", F.Seconds(time.Since(s.createdAt).Seconds()), "s)")
	return nil
}

type boxComponent struct {
	name      string
	lifecycle adapter.Lifecycle
}

func (s *Box) startComponents(stage adapter.StartStage, components ...boxComponent) error {
	for _, component := range components {
		if err := s.scope.Start(component.name, component.lifecycle, stage); err != nil {
			return err
		}
	}
	return nil
}

func (s *Box) startInternalServices(stage adapter.StartStage) error {
	for _, lifecycleService := range s.internalService {
		if err := s.scope.Start(lifecycleService.Name(), lifecycleService, stage); err != nil {
			return err
		}
	}
	return nil
}

func (s *Box) preStart() error {
	monitor := taskmonitor.New(s.logger, C.StartTimeout)
	monitor.Start("start logger")
	s.scope.Add(s.logFactory.Close)
	s.logFactoryInScope = true
	err := s.logFactory.Start()
	monitor.Finish()
	if err != nil {
		return common.NewError(err, "start logger")
	}
	applyDebugOptions(s.debugOptions)
	debugHTTPServer, err := startDebugHTTPServer(s.debugOptions)
	if err != nil {
		return err
	}
	if debugHTTPServer != nil {
		s.scope.Add(func() error {
			return common.NewError(debugHTTPServer.Close(), "close debug HTTP server")
		})
	}
	err = s.startInternalServices(adapter.StartStateInitialize) // cache-file clash-api v2ray-api
	if err != nil {
		return err
	}
	err = s.startComponents(adapter.StartStateInitialize,
		boxComponent{s.httpClientService.Name(), s.httpClientService},
		boxComponent{"network", s.network},
		boxComponent{"dns-transport", s.dnsTransport},
		boxComponent{"dns-router", s.dnsRouter},
		boxComponent{"connection", s.connection},
		boxComponent{"router", s.router},
		boxComponent{"outbound", s.outbound},
		boxComponent{"provider", s.provider},
		boxComponent{"endpoint", s.endpoint},
		boxComponent{"certificate-provider", s.certificateProvider},
		boxComponent{"inbound", s.inbound},
		boxComponent{"service", s.service},
	)
	if err != nil {
		return err
	}
	return s.startComponents(adapter.StartStateStart,
		boxComponent{"outbound", s.outbound},
		boxComponent{"provider", s.provider},
		boxComponent{"dns-transport", s.dnsTransport},
		boxComponent{"network", s.network},
		boxComponent{"connection", s.connection},
		boxComponent{s.httpClientService.Name(), s.httpClientService},
		boxComponent{"router", s.router},
		boxComponent{"dns-router", s.dnsRouter},
	)
}

func (s *Box) start() error {
	err := s.preStart()
	if err != nil {
		return err
	}
	err = s.startInternalServices(adapter.StartStateStart)
	if err != nil {
		return err
	}
	if s.ntpService != nil {
		done := adapter.LogElapsed(s.logger, "start ntp service")
		err = s.ntpService.Start()
		done()
		if err != nil {
			return common.NewError(err, "start ntp service")
		}
		s.scope.Add(s.ntpService.Close)
	}
	err = s.startComponents(adapter.StartStateStart,
		boxComponent{"endpoint", s.endpoint},
		boxComponent{"certificate-provider", s.certificateProvider},
		boxComponent{"inbound", s.inbound},
		boxComponent{"service", s.service},
	)
	if err != nil {
		return err
	}
	err = s.startComponents(adapter.StartStatePostStart,
		boxComponent{"outbound", s.outbound},
		boxComponent{"provider", s.provider},
		boxComponent{"network", s.network},
		boxComponent{"dns-transport", s.dnsTransport},
		boxComponent{"dns-router", s.dnsRouter},
		boxComponent{"connection", s.connection},
		boxComponent{"router", s.router},
		boxComponent{"endpoint", s.endpoint},
		boxComponent{"certificate-provider", s.certificateProvider},
		boxComponent{"inbound", s.inbound},
		boxComponent{"service", s.service},
	)
	if err != nil {
		return err
	}
	err = s.startInternalServices(adapter.StartStatePostStart)
	if err != nil {
		return err
	}
	err = s.startComponents(adapter.StartStateStarted,
		boxComponent{"network", s.network},
		boxComponent{"dns-transport", s.dnsTransport},
		boxComponent{"dns-router", s.dnsRouter},
		boxComponent{"connection", s.connection},
		boxComponent{"router", s.router},
		boxComponent{"outbound", s.outbound},
		boxComponent{"provider", s.provider},
		boxComponent{"endpoint", s.endpoint},
		boxComponent{"certificate-provider", s.certificateProvider},
		boxComponent{"inbound", s.inbound},
		boxComponent{"service", s.service},
	)
	if err != nil {
		return err
	}
	return s.startInternalServices(adapter.StartStateStarted)
}

func (s *Box) Close() error {
	s.closeOnce.Do(func() {
		s.closeErr = s.close()
	})
	return s.closeErr
}

func (s *Box) close() error {
	if s.done != nil {
		close(s.done)
	}
	s.logger.Info("closing sing-box")
	var err error
	if s.scope != nil {
		err = s.scope.Close()
	}
	if !s.logFactoryInScope && s.logFactory != nil {
		err = errors.Join(err, s.logFactory.Close())
	}
	s.logger.Info("sing-box closed (live time: ", F.Seconds(time.Since(s.createdAt).Seconds()), "s)")
	if s.statsTracker != nil {
		s.statsTracker.Reset()
	}
	if s.connTracker != nil {
		s.connTracker.Reset()
	}
	return err
}

func (s *Box) Uptime() uint32 {
	return uint32(time.Since(s.createdAt).Seconds())
}

func (s *Box) Network() adapter.NetworkManager {
	return s.network
}

func (s *Box) Router() adapter.Router {
	return s.router
}

func (s *Box) Inbound() adapter.InboundManager {
	return s.inbound
}

func (s *Box) Outbound() adapter.OutboundManager {
	return s.outbound
}

func (s *Box) Service() adapter.ServiceManager {
	return s.service
}

func (s *Box) Endpoint() adapter.EndpointManager {
	return s.endpoint
}

func (s *Box) LogFactory() log.Factory {
	return s.logFactory
}

func (s *Box) StatsTracker() *StatsTracker {
	return s.statsTracker
}

func (s *Box) ConnTracker() *ConnTracker {
	return s.connTracker
}
