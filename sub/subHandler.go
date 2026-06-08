package sub

import (
	"time"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/logger"
	"github.com/deposist/s-ui-x/service"
	"github.com/deposist/s-ui-x/util"

	"github.com/gin-gonic/gin"
)

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
