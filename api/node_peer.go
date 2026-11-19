package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/deposist/s-ui-x-extended/database"
	"github.com/deposist/s-ui-x-extended/service"
	"github.com/gin-gonic/gin"
)

func RegisterNodePeerRoutes(group *gin.RouterGroup) {
	RegisterNodePeerRoutesWithRuntime(group, nil)
}

func RegisterNodePeerRoutesWithRuntime(group *gin.RouterGroup, runtime *service.Runtime) {
	nodes := &service.NodeService{Runtime: runtime}
	group.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		if !RequestIsHTTPS(c) {
			c.AbortWithStatus(http.StatusForbidden)
			return
		}
		leave, ok := database.TryEnterDBOperation()
		if !ok {
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		defer leave()
		c.Next()
	})
	group.POST("/enroll", func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
		var input struct {
			Code string `json:"code"`
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.Status(http.StatusBadRequest)
			return
		}
		credential, err := nodes.RedeemInvite(input.Code, input.Name)
		if errors.Is(err, service.ErrInvalidNodeInvite) {
			c.Status(http.StatusUnauthorized)
			return
		}
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusCreated, credential)
	})
	group.POST("/sync", func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 256<<10)
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		var input *service.PeerSyncRequest
		if err := decoder.Decode(&input); err != nil || input == nil || decoder.Decode(new(any)) != io.EOF {
			c.Status(http.StatusBadRequest)
			return
		}
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok {
			c.Status(http.StatusUnauthorized)
			return
		}
		input.LeaseReceipt = c.GetHeader(service.PeerLeaseHeader)
		input.LeaseCapable = input.LeaseReceipt != ""
		input.ProbeReceipt = c.GetHeader(service.PeerProbeHeader)
		input.ProbeCapable = input.ProbeReceipt != ""
		input.ApplyFailed = c.GetHeader(service.PeerApplyHeader) == "1"
		output, err := nodes.AcceptPeerSync(c.GetHeader("X-Node-ID"), token, *input)
		if errors.Is(err, service.ErrInvalidNodeCredential) {
			c.Status(http.StatusUnauthorized)
			return
		}
		if errors.Is(err, service.ErrInvalidPeerInventory) {
			c.Status(http.StatusBadRequest)
			return
		}
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.JSON(http.StatusOK, output)
	})
}

func (a *ApiService) getNodePanel(c *gin.Context) {
	panel, err := a.NodeService.PanelSetup()
	jsonObj(c, panel, err)
}

func (a *ApiService) setNodeMaster(c *gin.Context) {
	jsonMsg(c, "", a.NodeService.SetMaster())
}

func (a *ApiService) createNodeInvite(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	if c.Query("details") == "1" {
		invite, err := a.NodeService.InviteDetails()
		jsonObj(c, invite, err)
		return
	}
	code, err := a.NodeService.Invite()
	jsonObj(c, code, err)
}

func (a *ApiService) joinNodeMaster(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	var input struct {
		MasterURL string `json:"masterUrl"`
		Name      string `json:"name"`
		Code      string `json:"code"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	jsonMsg(c, "", a.NodeService.JoinMaster(c.Request.Context(), input.MasterURL, input.Name, strings.TrimSpace(input.Code)))
}

func allowLocalClientWrite(c *gin.Context) bool {
	err := service.RequireLocalClientWrites(database.GetDB())
	if err == nil {
		return true
	}
	status := http.StatusInternalServerError
	message := "could not verify the panel role"
	if errors.Is(err, service.ErrSlaveClientWrite) {
		status = http.StatusForbidden
		message = err.Error()
	}
	c.AbortWithStatusJSON(status, Msg{Success: false, Msg: message})
	return false
}
