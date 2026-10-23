package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"

	"github.com/gin-gonic/gin"
)

// registerNodeRoutes mounts the remote-server endpoints.
//
// The routes live on the already-authenticated browser group, so they inherit
// session + CSRF like the rest of the panel. Mutations record an audit event.
func (a *APIHandler) registerNodeRoutes(g *gin.RouterGroup) {
	nodes := g.Group("/nodes")
	nodes.GET("", a.ApiService.GetNodes)
	nodes.POST("", a.ApiService.CreateNode)
	nodes.GET("/:uuid", a.ApiService.GetNode)
	nodes.PATCH("/:uuid", a.ApiService.UpdateNode)
	nodes.DELETE("/:uuid", a.ApiService.DeleteNode)
	nodes.GET("/:uuid/clients", a.ApiService.GetNodeClients)
	nodes.POST("/:uuid/clients", a.ApiService.AddNodeClients)
	nodes.DELETE("/:uuid/clients", a.ApiService.RemoveNodeClients)

	g.PATCH("/clients/:id/nodes", a.ApiService.SetClientNodes)
}

// nodeClientRequest is the bulk add/remove payload. ids are the selected client
// ids, which the UI keeps across pages, so a bulk call never depends on which
// page the operator happens to be looking at.
type nodeClientRequest struct {
	IDs []uint `json:"ids"`
}

func (a *ApiService) GetNodes(c *gin.Context) {
	nodes, err := a.NodeService.List()
	jsonObj(c, nodes, err)
}

func (a *ApiService) GetNode(c *gin.Context) {
	node, err := a.NodeService.Get(c.Param("uuid"))
	jsonObj(c, node, err)
}

func (a *ApiService) CreateNode(c *gin.Context) {
	var body struct {
		Name     string `json:"name"`
		LinkHost string `json:"linkHost"`
		Enabled  *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		jsonMsg(c, "nodes", err)
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	node, err := a.NodeService.Create(model.Node{
		Name:     body.Name,
		LinkHost: body.LinkHost,
		Enabled:  enabled,
	})
	if err == nil {
		a.recordAudit(c, requestActor(c), "node_created", "node", service.AuditSeverityWarn, map[string]any{
			"uuid": node.UUID,
			"name": node.Name,
		})
	}
	jsonObj(c, node, err)
}

func (a *ApiService) UpdateNode(c *gin.Context) {
	var body struct {
		Name     string `json:"name"`
		LinkHost string `json:"linkHost"`
		Enabled  *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		jsonMsg(c, "nodes", err)
		return
	}
	uuid := c.Param("uuid")
	current, err := a.NodeService.Get(uuid)
	if err != nil {
		jsonMsg(c, "nodes", err)
		return
	}
	enabled := current.Enabled
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	node, err := a.NodeService.Update(uuid, body.Name, body.LinkHost, enabled)
	if err == nil {
		a.recordAudit(c, requestActor(c), "node_updated", "node", service.AuditSeverityWarn, map[string]any{
			"uuid":    uuid,
			"name":    node.Name,
			"enabled": node.Enabled,
		})
	}
	jsonObj(c, node, err)
}

func (a *ApiService) DeleteNode(c *gin.Context) {
	uuid := c.Param("uuid")
	err := a.NodeService.Delete(uuid)
	if err == nil {
		a.recordAudit(c, requestActor(c), "node_deleted", "node", service.AuditSeverityWarn, map[string]any{
			"uuid": uuid,
		})
	}
	jsonMsg(c, "nodes", err)
}

// GetNodeClients serves one page of the clients assigned to a node. Paging is
// server side: limit/offset plus the total under the current filter.
func (a *ApiService) GetNodeClients(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	page, err := a.NodeService.Clients(c.Param("uuid"), limit, offset, c.Query("q"))
	jsonObj(c, page, err)
}

func (a *ApiService) AddNodeClients(c *gin.Context) {
	a.changeNodeClients(c, true)
}

func (a *ApiService) RemoveNodeClients(c *gin.Context) {
	a.changeNodeClients(c, false)
}

func (a *ApiService) changeNodeClients(c *gin.Context, assigned bool) {
	var body nodeClientRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		jsonMsg(c, "nodes", err)
		return
	}
	uuid := c.Param("uuid")
	var (
		changed int
		err     error
	)
	if assigned {
		changed, err = a.NodeService.Assign(uuid, body.IDs)
	} else {
		changed, err = a.NodeService.Unassign(uuid, body.IDs)
	}
	if err == nil {
		event := "node_clients_removed"
		if assigned {
			event = "node_clients_added"
		}
		a.recordAudit(c, requestActor(c), event, "node", service.AuditSeverityWarn, map[string]any{
			"uuid":    uuid,
			"changed": changed,
		})
	}
	jsonMsgObj(c, "nodes", gin.H{"changed": changed}, err)
}

// SetClientNodes replaces the server checkboxes of one client.
func (a *ApiService) SetClientNodes(c *gin.Context) {
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || clientID == 0 {
		jsonMsg(c, "nodes", http.ErrMissingFile)
		return
	}
	var body struct {
		Nodes []string `json:"nodes"`
	}
	if err := json.NewDecoder(c.Request.Body).Decode(&body); err != nil {
		jsonMsg(c, "nodes", err)
		return
	}
	err = a.NodeService.SetClientNodes(uint(clientID), body.Nodes)
	if err == nil {
		a.recordAudit(c, requestActor(c), "client_nodes_changed", "client", service.AuditSeverityWarn, map[string]any{
			"clientId": clientID,
			"nodes":    len(body.Nodes),
		})
	}
	jsonMsg(c, "nodes", err)
}
