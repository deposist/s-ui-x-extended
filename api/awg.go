package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/deposist/s-ui-x-extended/util/redact"

	"github.com/gin-gonic/gin"
)

// registerAWGRoutes mounts admin-side managed AmneziaWG device routes. The
// group is already session-authenticated and CSRF-protected; every handler
// additionally requires an admin/write token scope for mutations.
func (a *APIHandler) registerAWGRoutes(g *gin.RouterGroup) {
	awg := g.Group("/awg")
	awg.GET("/endpoints", a.ApiService.ListAWGEndpoints)
	awg.GET("/obfuscation/random", a.ApiService.GetAWGObfuscationRandom)
	awg.GET("/clients/:clientId/access", a.ApiService.ListClientAWGAccess)
	awg.GET("/clients/:clientId/devices", a.ApiService.ListClientAWGDevices)
	awg.POST("/clients/:clientId/devices", a.ApiService.CreateClientAWGDevice)
	awg.GET("/clients/:clientId/devices/:deviceId/config", a.ApiService.GetClientAWGDeviceConfig)
	awg.GET("/clients/:clientId/devices/:deviceId/qr", a.ApiService.GetClientAWGDeviceQR)
	awg.POST("/clients/:clientId/devices/:deviceId/rotate", a.ApiService.RotateClientAWGDevice)
	awg.POST("/clients/:clientId/devices/:deviceId/revoke", a.ApiService.RevokeClientAWGDevice)
}

func awgPathUint(c *gin.Context, name string) (uint, bool) {
	value, err := strconv.ParseUint(c.Param(name), 10, 32)
	if err != nil || value == 0 {
		jsonMsg(c, "awg", errors.New("invalid "+name))
		return 0, false
	}
	return uint(value), true
}

func awgQueryEndpointID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Query("endpointId"), 10, 32)
	if err != nil || value == 0 {
		jsonMsg(c, "awg", errors.New("invalid endpointId"))
		return 0, false
	}
	return uint(value), true
}

func awgEndpointDevices(a *ApiService) service.AWGEndpointDeviceService {
	runtime := a.Runtime
	if runtime == nil {
		runtime = service.DefaultRuntime()
	}
	return runtime.AWGEndpointDeviceService()
}

// ListAWGEndpoints returns every managed AWG endpoint with its non-secret
// metadata so the client form can offer them for assignment.
func (a *ApiService) ListAWGEndpoints(c *gin.Context) {
	db := database.GetDB()
	endpointIDs, err := service.ListManagedAWGEndpoints(db)
	if err != nil {
		jsonMsg(c, "awg", err)
		return
	}
	type endpointRow struct {
		ID                 uint     `json:"id"`
		Tag                string   `json:"tag"`
		PublicEndpoint     string   `json:"publicEndpoint"`
		Subnet             string   `json:"subnet"`
		DNS                []string `json:"dns"`
		DefaultDeviceLimit int      `json:"defaultDeviceLimit"`
	}
	rows := make([]endpointRow, 0, len(endpointIDs))
	for _, endpointID := range endpointIDs {
		endpoint, settings, err := service.LoadAWGEndpointByID(db, endpointID)
		if err != nil {
			jsonMsg(c, "awg", err)
			return
		}
		dns := make([]string, len(settings.DNS))
		for i := range settings.DNS {
			dns[i] = settings.DNS[i].String()
		}
		rows = append(rows, endpointRow{
			ID: endpoint.Id, Tag: endpoint.Tag, PublicEndpoint: settings.PublicEndpoint,
			Subnet: settings.Subnet.String(), DNS: dns, DefaultDeviceLimit: settings.DefaultDeviceLimit,
		})
	}
	jsonObj(c, rows, nil)
}

// GetAWGObfuscationRandom returns a server-generated Amnezia obfuscation
// parameter set. Generation stays on the server so there is one source of
// truth using crypto/rand instead of the browser's Math.random. H1-H4 are
// always generated; junk parameters (Jc/Jmin/Jmax) only when the caller
// explicitly asks for them (preset=balanced).
func (a *ApiService) GetAWGObfuscationRandom(c *gin.Context) {
	if !a.requireTokenScopeAny(c, "awg", "admin") {
		return
	}
	includeJunk := c.Query("preset") == "balanced"
	params, err := service.GenerateAmneziaParams(includeJunk)
	jsonObj(c, params, err)
}

func (a *ApiService) ListClientAWGAccess(c *gin.Context) {
	clientID, ok := awgPathUint(c, "clientId")
	if !ok {
		return
	}
	accesses, err := service.ListClientAWGEndpointAccess(database.GetDB(), clientID)
	jsonObj(c, accesses, err)
}

func (a *ApiService) ListClientAWGDevices(c *gin.Context) {
	clientID, ok := awgPathUint(c, "clientId")
	if !ok {
		return
	}
	endpointID, ok := awgQueryEndpointID(c)
	if !ok {
		return
	}
	devices := awgEndpointDevices(a)
	if devices == nil {
		jsonMsg(c, "awg", errors.New("AWG device service is unavailable"))
		return
	}
	list, err := devices.ListDevices(clientID, endpointID)
	jsonObj(c, list, err)
}

type createAWGDeviceRequest struct {
	EndpointID uint   `json:"endpointId" form:"endpointId"`
	Name       string `json:"name" form:"name"`
	RequestKey string `json:"requestKey" form:"requestKey"`
	// ExpiresAt is an optional exclusive Unix-seconds expiry (0 = never).
	// Validated server-side: in the future, at most 10 years ahead.
	ExpiresAt int64 `json:"expiresAt" form:"expiresAt"`
}

func (a *ApiService) CreateClientAWGDevice(c *gin.Context) {
	if !a.requireTokenScopeAny(c, "awg", "admin", "write") {
		return
	}
	clientID, ok := awgPathUint(c, "clientId")
	if !ok {
		return
	}
	var req createAWGDeviceRequest
	// ShouldBind selects the binding from the request Content-Type. The panel
	// frontend posts application/x-www-form-urlencoded (the axios default in
	// plugins/api.ts), while API consumers may post JSON; both must bind.
	if err := c.ShouldBind(&req); err != nil || req.EndpointID == 0 {
		jsonMsg(c, "awg", errors.New("invalid request"))
		return
	}
	devices := awgEndpointDevices(a)
	if devices == nil {
		jsonMsg(c, "awg", errors.New("AWG device service is unavailable"))
		return
	}
	requestKey := req.RequestKey
	if requestKey == "" {
		requestKey = fmt.Sprintf("admin-%d-%d-%d", clientID, req.EndpointID, time.Now().UnixNano())
	}
	device, err := devices.CreateDevice(c.Request.Context(), clientID, req.EndpointID, requestKey, req.Name, req.ExpiresAt)
	if err != nil {
		jsonMsg(c, "awg", err)
		return
	}
	a.recordAudit(c, GetLoginUser(c), "awg_device_created", "awg", service.AuditSeverityInfo, map[string]any{
		"clientId": clientID, "endpointId": req.EndpointID, "deviceId": device.ID, "expiresAt": req.ExpiresAt,
	})
	jsonObj(c, device, nil)
}

func (a *ApiService) GetClientAWGDeviceConfig(c *gin.Context) {
	clientID, ok := awgPathUint(c, "clientId")
	if !ok {
		return
	}
	deviceID, ok := awgPathUint(c, "deviceId")
	if !ok {
		return
	}
	endpointID, ok := awgQueryEndpointID(c)
	if !ok {
		return
	}
	devices := awgEndpointDevices(a)
	if devices == nil {
		jsonMsg(c, "awg", errors.New("AWG device service is unavailable"))
		return
	}
	config, err := devices.RenderOwnedConfig(c.Request.Context(), clientID, endpointID, deviceID)
	if err != nil {
		jsonMsg(c, "awg", err)
		return
	}
	a.recordAudit(c, GetLoginUser(c), "awg_config_downloaded", "awg", service.AuditSeverityInfo, map[string]any{
		"clientId": clientID, "endpointId": endpointID, "deviceId": deviceID,
	})
	c.Header("Cache-Control", "no-store")
	c.Header("Content-Disposition", `attachment; filename="awg-device.conf"`)
	c.Data(http.StatusOK, "application/octet-stream", config)
}

func (a *ApiService) GetClientAWGDeviceQR(c *gin.Context) {
	clientID, ok := awgPathUint(c, "clientId")
	if !ok {
		return
	}
	deviceID, ok := awgPathUint(c, "deviceId")
	if !ok {
		return
	}
	endpointID, ok := awgQueryEndpointID(c)
	if !ok {
		return
	}
	devices := awgEndpointDevices(a)
	if devices == nil {
		jsonMsg(c, "awg", errors.New("AWG device service is unavailable"))
		return
	}
	config, err := devices.RenderOwnedConfig(c.Request.Context(), clientID, endpointID, deviceID)
	if err != nil {
		awgQRError(c, err)
		return
	}
	png, err := service.RenderAWGConfigQR(config)
	if err != nil {
		awgQRError(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	c.Data(http.StatusOK, "image/png", png)
}

// awgQRError writes a genuine HTTP status for the binary QR endpoint. Unlike the
// JSON SPA envelope (jsonMsg, which always returns 200), this handler streams an
// image; a 200 body that is actually JSON would be fed straight into an <img>
// tag and render as an empty picture. Oversized configs map to 422 so the
// client can offer the .conf download instead; everything else is 500.
func awgQRError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, service.ErrAWGConfigTooLargeQR) {
		status = http.StatusUnprocessableEntity
	}
	c.JSON(status, Msg{Success: false, Msg: "awg: " + redact.String(err.Error())})
}

func (a *ApiService) RotateClientAWGDevice(c *gin.Context) {
	if !a.requireTokenScopeAny(c, "awg", "admin", "write") {
		return
	}
	clientID, ok := awgPathUint(c, "clientId")
	if !ok {
		return
	}
	deviceID, ok := awgPathUint(c, "deviceId")
	if !ok {
		return
	}
	endpointID, ok := awgQueryEndpointID(c)
	if !ok {
		return
	}
	devices := awgEndpointDevices(a)
	if devices == nil {
		jsonMsg(c, "awg", errors.New("AWG device service is unavailable"))
		return
	}
	requestKey := fmt.Sprintf("admin-rotate-%d-%d-%d", clientID, deviceID, time.Now().UnixNano())
	device, err := devices.RotateOwnedDevice(c.Request.Context(), clientID, endpointID, deviceID, requestKey)
	if err != nil {
		jsonMsg(c, "awg", err)
		return
	}
	a.recordAudit(c, GetLoginUser(c), "awg_device_rotated", "awg", service.AuditSeverityInfo, map[string]any{
		"clientId": clientID, "endpointId": endpointID, "deviceId": deviceID,
	})
	jsonObj(c, device, nil)
}

func (a *ApiService) RevokeClientAWGDevice(c *gin.Context) {
	if !a.requireTokenScopeAny(c, "awg", "admin", "write") {
		return
	}
	clientID, ok := awgPathUint(c, "clientId")
	if !ok {
		return
	}
	deviceID, ok := awgPathUint(c, "deviceId")
	if !ok {
		return
	}
	endpointID, ok := awgQueryEndpointID(c)
	if !ok {
		return
	}
	devices := awgEndpointDevices(a)
	if devices == nil {
		jsonMsg(c, "awg", errors.New("AWG device service is unavailable"))
		return
	}
	if err := devices.RevokeOwnedDevice(c.Request.Context(), clientID, endpointID, deviceID); err != nil {
		jsonMsg(c, "awg", err)
		return
	}
	a.recordAudit(c, GetLoginUser(c), "awg_device_revoked", "awg", service.AuditSeverityWarn, map[string]any{
		"clientId": clientID, "endpointId": endpointID, "deviceId": deviceID,
	})
	jsonObj(c, gin.H{"revoked": true}, nil)
}
