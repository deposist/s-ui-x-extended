package api

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/deposist/s-ui-x-extended/database/model"
	"github.com/deposist/s-ui-x-extended/service"

	"github.com/gin-gonic/gin"
)

// registerNodeRoutes mounts the remote-server endpoints.
//
// The routes live on the already-authenticated browser group, so they inherit
// session + CSRF like the rest of the panel. Bodies are form-encoded to match
// every other handler here; the two list fields carry a JSON array.
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

	g.GET("/clients/:id/nodes", a.ApiService.GetClientNodes)
	g.PATCH("/clients/:id/nodes", a.ApiService.SetClientNodes)
	g.GET("/clients/:id/limits", a.ApiService.GetClientLimits)
	g.PATCH("/clients/:id/limits", a.ApiService.SetClientLimits)
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
	// A node that arrives enabled by default keeps creation to two fields; the
	// switch only has to be sent when it is being turned off.
	enabled := true
	if raw := c.Request.FormValue("enabled"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			jsonMsg(c, "nodes", err)
			return
		}
		enabled = parsed
	}
	node, err := a.NodeService.Create(model.Node{
		Name:     c.Request.FormValue("name"),
		LinkHost: c.Request.FormValue("linkHost"),
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
	uuid := c.Param("uuid")
	current, err := a.NodeService.Get(uuid)
	if err != nil {
		jsonMsg(c, "nodes", err)
		return
	}
	name := c.Request.FormValue("name")
	if name == "" {
		name = current.Name
	}
	linkHost := c.Request.FormValue("linkHost")
	if linkHost == "" {
		linkHost = current.LinkHost
	}
	enabled := current.Enabled
	if raw := c.Request.FormValue("enabled"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			jsonMsg(c, "nodes", err)
			return
		}
		enabled = parsed
	}
	node, err := a.NodeService.Update(uuid, name, linkHost, enabled)
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
	// assigned=1 limits the result to the server's own clients; without it the
	// endpoint is the picker, listing every client with an assigned flag.
	onlyAssigned := c.Query("assigned") == "1" || c.Query("assigned") == "true"
	page, err := a.NodeService.Clients(c.Param("uuid"), limit, offset, c.Query("q"), onlyAssigned)
	jsonObj(c, page, err)
}

func (a *ApiService) AddNodeClients(c *gin.Context) {
	a.changeNodeClients(c, true)
}

func (a *ApiService) RemoveNodeClients(c *gin.Context) {
	a.changeNodeClients(c, false)
}

func (a *ApiService) changeNodeClients(c *gin.Context, assigned bool) {
	ids, err := formUintList(c.Request.FormValue("ids"))
	if err != nil {
		jsonMsg(c, "nodes", err)
		return
	}
	uuid := c.Param("uuid")
	var changed int
	if assigned {
		changed, err = a.NodeService.Assign(uuid, ids)
	} else {
		changed, err = a.NodeService.Unassign(uuid, ids)
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

// GetClientNodes returns the servers one client is assigned to, for the
// checkboxes in the client card.
func (a *ApiService) GetClientNodes(c *gin.Context) {
	clientID, ok := clientIDParam(c)
	if !ok {
		return
	}
	uuids, err := a.NodeService.ClientNodeUUIDs(clientID)
	jsonObj(c, uuids, err)
}

// SetClientNodes replaces the server checkboxes of one client.
func (a *ApiService) SetClientNodes(c *gin.Context) {
	clientID, ok := clientIDParam(c)
	if !ok {
		return
	}
	var nodeUUIDs []string
	if raw := c.Request.FormValue("nodes"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &nodeUUIDs); err != nil {
			jsonMsg(c, "nodes", err)
			return
		}
	}
	err := a.NodeService.SetClientNodes(clientID, nodeUUIDs)
	if err == nil {
		a.recordAudit(c, requestActor(c), "client_nodes_changed", "client", service.AuditSeverityWarn, map[string]any{
			"clientId": clientID,
			"nodes":    len(nodeUUIDs),
		})
	}
	jsonMsg(c, "nodes", err)
}

// GetClientLimits returns one client's per-user limits.
func (a *ApiService) GetClientLimits(c *gin.Context) {
	clientID, ok := clientIDParam(c)
	if !ok {
		return
	}
	limits, err := a.NodeService.GetClientLimits(clientID)
	jsonObj(c, limits, err)
}

// SetClientLimits stores one client's per-user limits. They are per client and
// enforced by every server that client is assigned to.
func (a *ApiService) SetClientLimits(c *gin.Context) {
	clientID, ok := clientIDParam(c)
	if !ok {
		return
	}
	limits := model.ClientLimits{SpeedShared: true}
	if raw := c.Request.FormValue("speedShared"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			jsonMsg(c, "nodes", err)
			return
		}
		limits.SpeedShared = parsed
	}
	for _, field := range []struct {
		name   string
		target *uint64
	}{
		{"speed", &limits.Speed},
		{"downloadSpeed", &limits.DownloadSpeed},
		{"uploadSpeed", &limits.UploadSpeed},
	} {
		raw := c.Request.FormValue(field.name)
		if raw == "" {
			continue
		}
		value, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			jsonMsg(c, "nodes", err)
			return
		}
		*field.target = value
	}
	if raw := c.Request.FormValue("connections"); raw != "" {
		value, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			jsonMsg(c, "nodes", err)
			return
		}
		limits.Connections = uint32(value)
	}
	saved, err := a.NodeService.SetClientLimits(clientID, limits)
	if err == nil {
		a.recordAudit(c, requestActor(c), "client_limits_changed", "client", service.AuditSeverityWarn, map[string]any{
			"clientId":    clientID,
			"speedShared": saved.SpeedShared,
			"connections": saved.Connections,
		})
	}
	jsonObj(c, saved, err)
}

// clientIDParam reads the numeric client id from the path.
func clientIDParam(c *gin.Context) (uint, bool) {
	clientID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || clientID == 0 {
		jsonMsg(c, "nodes", fmt.Errorf("client id is not a number: %q", c.Param("id")))
		return 0, false
	}
	return uint(clientID), true
}

// formUintList reads a JSON array of ids from one form field. The UI keeps its
// selection across pages, so a bulk call always sends the whole selection.
func formUintList(raw string) ([]uint, error) {
	if raw == "" {
		return nil, nil
	}
	var ids []uint
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, err
	}
	return ids, nil
}
