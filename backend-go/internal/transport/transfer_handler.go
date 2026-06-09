package transport

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/aspira/aspira-pay/internal/domain/transfer"
	"github.com/aspira/aspira-pay/internal/service"
)

type TransferHandler struct {
	svc *service.TransferService
}

func NewTransferHandler(svc *service.TransferService) *TransferHandler {
	return &TransferHandler{svc: svc}
}

// ResolveRecipient looks up a recipient (§5.3.1).
func (h *TransferHandler) ResolveRecipient(c *gin.Context) {
	var req transfer.ResolveRecipientRequest
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	resp, err := h.svc.ResolveRecipient(req)
	if err != nil { c.JSON(404, gin.H{"error": err.Error()}); return }
	c.JSON(200, resp)
}

// CreateQuote generates a transfer quote (§5.3.2).
func (h *TransferHandler) CreateQuote(c *gin.Context) {
	var req transfer.TransferQuoteRequest
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	resp, err := h.svc.CreateQuote(req)
	if err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	c.JSON(200, resp)
}

// ConfirmTransfer executes a transfer (§5.3.3).
func (h *TransferHandler) ConfirmTransfer(c *gin.Context) {
	var req transfer.TransferConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	payerID := c.GetString("user_id")
	order, err := h.svc.ConfirmTransfer(payerID, req)
	if err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	c.JSON(200, order)
}

// ListTransfers returns transfer history.
func (h *TransferHandler) ListTransfers(c *gin.Context) {
	userID := c.GetString("user_id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	orders, total, err := h.svc.ListTransfers(userID, page, pageSize)
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"transfers": orders, "total": total})
}

// ListContacts returns transfer contact history.
func (h *TransferHandler) ListContacts(c *gin.Context) {
	contacts, err := h.svc.ListContacts(c.GetString("user_id"))
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"contacts": contacts})
}

// ── Payment Link Handlers (§6.7) ────────────────

// CreatePaymentLink creates a new payment link (§6.7.1).
func (h *TransferHandler) CreatePaymentLink(c *gin.Context) {
	var req transfer.CreatePaymentLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	link, err := h.svc.CreatePaymentLink(c.GetString("user_id"), req)
	if err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	c.JSON(201, link)
}

// GetPaymentLinkPublic returns payment link info by token (§6.7.2 — public).
func (h *TransferHandler) GetPaymentLinkPublic(c *gin.Context) {
	token := c.Param("token")
	link, err := h.svc.GetPaymentLinkByToken(token)
	if err != nil { c.JSON(404, gin.H{"error": "payment link not found or expired"}); return }
	c.JSON(200, gin.H{
		"payment_link_id": link.PaymentLinkID, "status": link.Status,
		"amount": link.Amount, "currency": link.Currency,
		"title": link.Title, "description": link.Description,
		"expire_at": link.ExpireAt,
	})
}

// PayPaymentLink processes payment for a link (§6.7.4).
func (h *TransferHandler) PayPaymentLink(c *gin.Context) {
	linkID := c.Param("payment_link_id")
	var req struct { SourceAccountID string `json:"source_account_id"` }
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	order, err := h.svc.PayPaymentLink(c.GetString("user_id"), linkID, req.SourceAccountID)
	if err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	c.JSON(200, order)
}

// CancelPaymentLink cancels a payment link (§6.7.5).
func (h *TransferHandler) CancelPaymentLink(c *gin.Context) {
	if err := h.svc.CancelPaymentLink(c.GetString("user_id"), c.Param("payment_link_id")); err != nil {
		c.JSON(400, gin.H{"error": err.Error()}); return
	}
	c.JSON(200, gin.H{"status": "cancelled"})
}

// ListPaymentLinks returns links created by the user.
func (h *TransferHandler) ListPaymentLinks(c *gin.Context) {
	links, _ := h.svc.ListPaymentLinks(c.GetString("user_id"))
	c.JSON(200, gin.H{"links": links})
}
