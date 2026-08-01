package sub

import (
	"sync"
	"time"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/logger"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/deposist/s-ui-x-extended/util"

	"github.com/gin-gonic/gin"
)

// --- /sub enumeration detection (D-1, MITRE T1190) ---
// The /sub surface is semi-public and previously had zero audit coverage, so
// subscription-token scraping / sub-id guessing left no trace. Invalid sub-id
// lookups are counted per source IP; crossing a threshold within a window emits
// a single throttled warn audit event (debounced so a scanner cannot flood the
// audit log itself).
const (
	subEnumWindow    = 15 * time.Minute
	subEnumThreshold = 10
	subEnumMaxKeys   = 4096
)

type subEnumState struct {
	count     int
	windowAt  time.Time
	alertedAt time.Time
}

var (
	subEnumMu   sync.Mutex
	subEnumByIP = map[string]subEnumState{}
)

func noteSubNotFound(ip string) {
	if ip == "" {
		return
	}
	subEnumMu.Lock()
	now := time.Now()
	st := subEnumByIP[ip]
	if st.windowAt.IsZero() || now.Sub(st.windowAt) > subEnumWindow {
		st = subEnumState{windowAt: now}
	}
	st.count++
	alert := st.count >= subEnumThreshold && now.Sub(st.alertedAt) > subEnumWindow
	if alert {
		st.alertedAt = now
	}
	if _, exists := subEnumByIP[ip]; !exists && len(subEnumByIP) >= subEnumMaxKeys {
		for candidate, state := range subEnumByIP {
			if now.Sub(state.windowAt) > subEnumWindow {
				delete(subEnumByIP, candidate)
			}
		}
		if len(subEnumByIP) >= subEnumMaxKeys {
			subEnumMu.Unlock()
			return
		}
	}
	subEnumByIP[ip] = st
	count := st.count
	subEnumMu.Unlock()

	if alert {
		_ = (&service.AuditService{}).Record(service.AuditEvent{
			Actor:    "anonymous",
			Event:    "sub_enumeration",
			Resource: "sub",
			Severity: service.AuditSeverityWarn,
			IP:       ip,
			Details:  map[string]any{"invalidLookups": count, "windowMinutes": int(subEnumWindow.Minutes())},
		})
	}
}

type SubHandler struct {
	service.SettingService
	SubService
	JsonService
	ClashService
}

const maxSubscriptionHeaderBytes = 512

func NewSubHandler(g *gin.RouterGroup) {
	a := &SubHandler{}
	a.initRouter(g)
}

func (s *SubHandler) initRouter(g *gin.RouterGroup) {
	g.Use(rateLimitMiddleware())
	g.GET("/:subid", s.subs)
	g.HEAD("/:subid", s.subHeaders)
	g.GET("/json/:subid", s.json)
	g.HEAD("/json/:subid", s.subHeaders)
	g.GET("/clash/:subid", s.clash)
	g.HEAD("/clash/:subid", s.subHeaders)
}

func (s *SubHandler) subs(c *gin.Context) {
	format, isFormat := c.GetQuery("format")
	if isFormat {
		switch format {
		case "json":
			s.json(c)
		case "clash":
			s.clash(c)
		default:
			c.String(400, "Error!")
		}
		return
	}
	if !s.subLinkEnabled(c) {
		return
	}

	var headers []string
	var result *string
	var err error
	subId := c.Param("subid")
	result, headers, err = s.SubService.GetSubs(subId)
	if err != nil || result == nil {
		logger.Error(err)
		s.writeError(c, err)
		return
	}

	s.writeResult(c, result, headers)
}

func (s *SubHandler) json(c *gin.Context) {
	result, headers, err := s.JsonService.GetJson(c.Param("subid"), "json")
	if err != nil || result == nil {
		logger.Error(err)
		s.writeError(c, err)
		return
	}
	s.writeResult(c, result, headers)
}

func (s *SubHandler) clash(c *gin.Context) {
	result, headers, err := s.ClashService.GetClash(c.Param("subid"))
	if err != nil || result == nil {
		logger.Error(err)
		s.writeError(c, err)
		return
	}
	s.writeResult(c, result, headers)
}

func (s *SubHandler) subHeaders(c *gin.Context) {
	if !s.subLinkEnabled(c) {
		return
	}
	subId := c.Param("subid")
	client, err := s.SubService.getClientBySubId(subId)
	if err != nil {
		logger.Error(err)
		s.writeError(c, err)
		return
	}

	headers := buildClientHeaders(client, cachedSubDisplaySettings(&s.SettingService, time.Now()))
	s.addHeaders(c, headers)

	c.Status(200)
}

func (s *SubHandler) subLinkEnabled(c *gin.Context) bool {
	enabled, err := s.SettingService.GetSubLinkEnable()
	if err != nil {
		logger.Error(err)
		s.writeError(c, err)
		return false
	}
	if !enabled {
		c.String(404, "Not Found")
		return false
	}
	return true
}

func (s *SubHandler) addHeaders(c *gin.Context, headers []string) {
	if len(headers) < 3 {
		return
	}
	headers = safeSubscriptionHeaders(headers)
	c.Writer.Header().Set("Subscription-Userinfo", headers[0])
	c.Writer.Header().Set("Profile-Update-Interval", headers[1])
	c.Writer.Header().Set("Profile-Title", headers[2])
	if len(headers) > 3 && headers[3] != "" {
		c.Writer.Header().Set("Support-Url", headers[3])
	}
	if len(headers) > 4 && headers[4] != "" {
		c.Writer.Header().Set("Profile-Web-Page-Url", headers[4])
	}
	if len(headers) > 5 && headers[5] != "" {
		c.Writer.Header().Set("Profile-Announcement", headers[5])
	}
}

func (s *SubHandler) writeResult(c *gin.Context, result *string, headers []string) {
	s.addHeaders(c, headers)
	c.String(200, *result)
}

func (s *SubHandler) writeError(c *gin.Context, err error) {
	if database.IsNotFound(err) {
		noteSubNotFound(c.ClientIP())
		c.String(404, "Not Found")
		return
	}
	c.String(400, "Error!")
}

func safeSubscriptionHeaders(headers []string) []string {
	safe := make([]string, len(headers))
	for i, header := range headers {
		safe[i] = util.SafeHeader(header, maxSubscriptionHeaderBytes)
	}
	return safe
}
