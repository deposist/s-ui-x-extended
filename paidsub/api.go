package paidsub

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/deposist/s-ui-x/database"
	"github.com/deposist/s-ui-x/database/model"

	"github.com/gin-gonic/gin"
)

// Deps carries the small set of host-app capabilities the module's HTTP
// handlers need (auth identity + audit), injected by the api package so the
// module stays decoupled from api internals.
type Deps struct {
	LoginUser func(*gin.Context) string
	Audit     func(c *gin.Context, actor, event, resource, severity string, details map[string]any)
}

type apiHandlers struct {
	svc      *PaidSubService
	tariffs  *TariffService
	payments *PaymentService
	deps     Deps
}

// RegisterRoutes mounts the module's admin endpoints under /paidsub on an
// ALREADY-authenticated group (session-auth + CSRF for browser routes). The
// module never registers public/unauthenticated routes.
func RegisterRoutes(g *gin.RouterGroup, deps Deps) {
	h := &apiHandlers{svc: NewService(), tariffs: NewTariffService(), payments: NewPaymentService(), deps: deps}
	grp := g.Group("/paidsub")
	grp.GET("/bindings", h.listBindings)
	grp.POST("/bindings", h.setBinding)
	grp.GET("/tariffs", h.listTariffs)
	grp.POST("/tariffs", h.saveTariff)
	grp.GET("/orders", h.listOrders)
	grp.POST("/refund", h.refund)
	grp.GET("/status", h.status)
	grp.POST("/broadcast", h.broadcast)
}

type broadcastRequest struct {
	Text string `json:"text"`
}

// broadcast sends a custom announcement to all bound Telegram users.
func (h *apiHandlers) broadcast(c *gin.Context) {
	var req broadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respFail(c, "invalid request")
		return
	}
	if strings.TrimSpace(req.Text) == "" {
		respFail(c, "message is empty")
		return
	}
	sent, failed, err := Broadcast(c.Request.Context(), req.Text)
	if err != nil {
		respFail(c, err.Error())
		return
	}
	h.audit(c, "paidsub_broadcast", map[string]any{"sent": sent, "failed": failed})
	respOK(c, map[string]any{"sent": sent, "failed": failed})
}

// status reports module health hints for the admin UI (whether the secretbox
// env key is configured — payment tokens are better protected when it is).
func (h *apiHandlers) status(c *gin.Context) {
	respOK(c, map[string]any{
		"secretboxKeySet": strings.TrimSpace(os.Getenv("SUI_SECRETBOX_KEY")) != "",
	})
}

// apiMsg mirrors api.Msg exactly — all three fields MUST be present (no
// omitempty), because the frontend's isMsg() requires the keys success, msg AND
// obj; omitting msg/obj makes the client report "unknown data".
type apiMsg struct {
	Success bool        `json:"success"`
	Msg     string      `json:"msg"`
	Obj     interface{} `json:"obj"`
}

func respOK(c *gin.Context, obj interface{}) {
	c.JSON(http.StatusOK, apiMsg{Success: true, Obj: obj})
}

func respFail(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, apiMsg{Success: false, Msg: msg})
}

func (h *apiHandlers) audit(c *gin.Context, event string, details map[string]any) {
	if h.deps.Audit == nil {
		return
	}
	actor := ""
	if h.deps.LoginUser != nil {
		actor = h.deps.LoginUser(c)
	}
	h.deps.Audit(c, actor, event, "paidsub", "info", details)
}

type bindingRow struct {
	ClientId uint   `json:"clientId"`
	Name     string `json:"name"`
	Enable   bool   `json:"enable"`
	TgUserId int64  `json:"tgUserId"`
	Desc     string `json:"desc"`
	Expiry   int64  `json:"expiry"`
}

// listBindings returns every client with its Telegram binding (tgUserId 0 = not
// bound), so the admin can manage the tg↔client mapping on the feature page.
func (h *apiHandlers) listBindings(c *gin.Context) {
	db := database.GetDB()
	var rows []bindingRow
	err := db.Table("clients c").
		Select("c.id as client_id, c.name as name, c.enable as enable, c.desc as `desc`, c.expiry as expiry, COALESCE(b.tg_user_id, 0) as tg_user_id").
		Joins("LEFT JOIN paidsub_bindings b ON b.client_id = c.id").
		Order("c.name").
		Scan(&rows).Error
	if err != nil {
		respFail(c, err.Error())
		return
	}
	respOK(c, rows)
}

type setBindingRequest struct {
	ClientId uint  `json:"clientId"`
	TgUserId int64 `json:"tgUserId"`
}

// setBinding maps (or, when tgUserId<=0, unmaps) a Telegram user to a client.
func (h *apiHandlers) setBinding(c *gin.Context) {
	var req setBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respFail(c, "invalid request")
		return
	}
	if req.ClientId == 0 {
		respFail(c, "clientId is required")
		return
	}
	db := database.GetDB()
	var count int64
	if err := db.Model(&model.Client{}).Where("id = ?", req.ClientId).Count(&count).Error; err != nil {
		respFail(c, err.Error())
		return
	}
	if count == 0 {
		respFail(c, "client not found")
		return
	}
	if req.TgUserId <= 0 {
		if err := h.svc.UnbindClient(req.ClientId); err != nil {
			respFail(c, err.Error())
			return
		}
		h.audit(c, "paidsub_unbound", map[string]any{"clientId": req.ClientId})
		respOK(c, nil)
		return
	}
	if err := h.svc.SetBinding(req.ClientId, req.TgUserId); err != nil {
		respFail(c, err.Error())
		return
	}
	h.audit(c, "paidsub_bound", map[string]any{"clientId": req.ClientId, "tgUserId": req.TgUserId})
	respOK(c, nil)
}

func (h *apiHandlers) listTariffs(c *gin.Context) {
	rows, err := h.tariffs.GetAll()
	if err != nil {
		respFail(c, err.Error())
		return
	}
	respOK(c, rows)
}

type saveTariffRequest struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

func (h *apiHandlers) saveTariff(c *gin.Context) {
	var req saveTariffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respFail(c, "invalid request")
		return
	}
	switch req.Action {
	case "new", "edit", "del", "delbulk":
	default:
		respFail(c, "invalid action")
		return
	}
	if err := h.tariffs.Save(req.Action, req.Data); err != nil {
		respFail(c, err.Error())
		return
	}
	h.audit(c, "paidsub_tariff_saved", map[string]any{"action": req.Action})
	respOK(c, nil)
}

// orderRow is the read-only order history projection shown in the admin UI. It
// joins the client's name/desc and deliberately selects only display columns —
// provider secrets (idempotency_key, provider_charge_id, provider_payload) are
// never selected, so they cannot leak.
type orderRow struct {
	Id             uint   `json:"id"`
	ClientId       uint   `json:"clientId"`
	Provider       string `json:"provider"`
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
	TelegramUserId int64  `json:"telegramUserId"`
	CreatedAt      int64  `json:"createdAt"`
	ClientName     string `json:"clientName"`
	ClientDesc     string `json:"clientDesc"`
}

// listOrders returns recent payment orders (read-only history) enriched with the
// client's name/desc via a LEFT JOIN (a deleted client yields empty name/desc).
func (h *apiHandlers) listOrders(c *gin.Context) {
	db := database.GetDB()
	var rows []orderRow
	err := db.Table("payment_orders o").
		Select("o.id as id, o.client_id as client_id, o.provider as provider, o.amount as amount, o.currency as currency, o.status as status, o.telegram_user_id as telegram_user_id, o.created_at as created_at, c.name as client_name, c.desc as client_desc").
		Joins("LEFT JOIN clients c ON c.id = o.client_id").
		Order("o.id desc").
		Limit(200).
		Scan(&rows).Error
	if err != nil {
		respFail(c, err.Error())
		return
	}
	respOK(c, rows)
}

type refundRequest struct {
	OrderId uint `json:"orderId"`
	Revoke  bool `json:"revoke"`
}

// refund is the admin-initiated refund: for Telegram Stars it calls
// refundStarPayment; for every other provider it only marks the order refunded
// (the admin refunds the money in the provider's own dashboard). Revoke rolls
// back the granted days/traffic (admin's per-refund choice).
func (h *apiHandlers) refund(c *gin.Context) {
	var req refundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respFail(c, "invalid request")
		return
	}
	if req.OrderId == 0 {
		respFail(c, "orderId is required")
		return
	}
	status, err := h.payments.RefundOrder(c.Request.Context(), req.OrderId, req.Revoke)
	if err != nil {
		respFail(c, err.Error())
		return
	}
	h.audit(c, "paidsub_refunded", map[string]any{"orderId": req.OrderId, "revoke": req.Revoke, "status": status})
	respOK(c, map[string]any{"status": status})
}
