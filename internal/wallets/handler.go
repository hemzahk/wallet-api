package wallets

import (
	jsonutils "encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/hemzahk/wallet-api/internal/gateway"
	"github.com/hemzahk/wallet-api/internal/json"
	"github.com/hemzahk/wallet-api/internal/store"
)

type handler struct {
	service Service
	gateway gateway.PaymentGateway
}

func NewHandler(service Service, gateway gateway.PaymentGateway) *handler {
	return &handler{
		service: service,
		gateway: gateway,
	}
}

type TopupRequest struct {
	Amount string `json:"amount" validate:"required"`
}

type WithdrawalRequest struct {
	Number     string `json:"number"`
	BankName   string  `json:"bank_name"`
	Amount string `json:"amount" validate:"required"`
	Reference string `json:"reference" validate:"required"`
}

type PayoutWebhookPayload struct {
	EventID  string `json:"event_id"`
	SourceID string `json:"source_id"` // for our PSP 645b44fb-314f-4e34-8106-17b09cc9660a
	GatewayRef string `json:"gateway_ref"`
	Status     string `json:"status"`
	Amount     string `json:"amount"`
}

type TransferRequest struct {
	Email string `json:"email" validate:"required,email"`
	Amount string `json:"amount" validate:"required"`
	Reference string `json:"reference" validate:"required"`
}

// Topup godoc
//
//	@Summary		Initiates a topup
//	@Description	Initiates a topup
//	@Tags			wallet
//	@Accept			json
//	@Produce		json
//	@Param        	Idempotency-Key		header    string    true   	"Idempotency-Key must be set for valid response"
//	@Param			payload	body		TopupRequest	true	"Topup amount"
//	@Success		202		{object}	string		"Topup initiated"
//	@Failure		401		{object}	string "unauthorized"
//	@Failure		404		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/wallet/topup [post]
func (h *handler) Topup(w http.ResponseWriter, r *http.Request) {
	var req TopupRequest
	if err := json.ReadJSON(w, r, &req); err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	if err := json.Validate.Struct(req); err != nil {
		json.BadRequestResponse(w, r, err)
		return
	}

	user := r.Context().Value("user").(*store.User)

	session, err := h.service.Topup(r.Context(), req, user)
	if err != nil {
		switch {
		case errors.Is(err, ErrMaxTopupAmountExceeded):
			json.JsonResponse(w, http.StatusUnprocessableEntity, err)
		case errors.Is(err, store.ErrBalanceNotFound):
			json.NotFoundResponse(w, r, err)
		default: 
			json.InternalServerError(w, r, err)
		}
		return
	}

	if err := json.JsonResponse(w, http.StatusAccepted, session); err != nil {
		json.InternalServerError(w, r, err)
	}
}

type TopupWebhookPayload struct {
	EventID  string `json:"event_id"`
	SourceID string `json:"source_id"` // for our PSP 645b44fb-314f-4e34-8106-17b09cc9660a
	GatewayRef string `json:"gateway_ref"`
	Status     string `json:"status"`
	Amount     string `json:"amount"`
}

// TopupWebhook godoc
//
//	@Summary		Response from the PSP
//	@Description	Response from the PSP
//	@Tags			webhooks
//	@Accept			json
//	@Produce		json
//	@Param        	X-Webhook-Signature		header    string    true   	"X-Webhook-Signature must be set for valid response"
//	@Param			payload	body		TopupWebhookPayload	true	"Webhook payload"
//	@Success		200		{object}	string		"success"
//	@Failure		500		{object}	error
//	@Router			/webhooks/topup [post]
func (h *handler) TopupWebhook(w http.ResponseWriter, r *http.Request) {
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		json.WriteJSONError(w, http.StatusBadRequest, "INVALID_BODY: failed to read request body")
		return
	}
	defer r.Body.Close()

	signature := r.Header.Get("X-Webhook-Signature")
	if err := h.gateway.VerifyWebhookSignature(rawBody, signature); err != nil {
		if errors.Is(err, gateway.ErrInvalidWebhookSignature) {
			json.WriteJSONError(w, http.StatusUnauthorized, "INVALID_SIGNATURE: webhook signature verification failed")
			return
		}
		json.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR: signature verification error")
		return
	}

	var payload TopupWebhookPayload
	if err := jsonutils.Unmarshal(rawBody, &payload); err != nil {
		json.WriteJSONError(w, http.StatusBadRequest, "INVALID_PAYLOAD: malformed webhook payload")
		return
	}
	
	if err := h.service.TopupWebhook(r.Context(), payload); err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	if err := json.JsonResponse(w, http.StatusOK, "success"); err != nil {
		json.InternalServerError(w, r, err)
	}
}

// 	PayoutWebhook godoc
//
//	@Summary		Response from the PSP
//	@Description	Response from the PSP
//	@Tags			webhooks
//	@Accept			json
//	@Produce		json
//	@Param        	X-Webhook-Signature		header    string    true   	"X-Webhook-Signature must be set for valid response"
//	@Param			payload	body		PayoutWebhookPayload	true	"Webhook payload"
//	@Success		200		{object}	string		"success"
//	@Failure		500		{object}	error
//	@Router			/webhooks/payout [post]
func (h *handler) PayoutWebhook(w http.ResponseWriter, r *http.Request) {
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		json.WriteJSONError(w, http.StatusBadRequest, "INVALID_BODY: failed to read request body")
		return
	}
	defer r.Body.Close()

	signature := r.Header.Get("X-Webhook-Signature")
	if err := h.gateway.VerifyWebhookSignature(rawBody, signature); err != nil {
		if errors.Is(err, gateway.ErrInvalidWebhookSignature) {
			json.WriteJSONError(w, http.StatusUnauthorized, "INVALID_SIGNATURE: webhook signature verification failed")
			return
		}
		json.WriteJSONError(w, http.StatusInternalServerError, "INTERNAL_ERROR: signature verification error")
		return
	}

	var payload PayoutWebhookPayload
	if err := jsonutils.Unmarshal(rawBody, &payload); err != nil {
		json.WriteJSONError(w, http.StatusBadRequest, "INVALID_PAYLOAD: malformed webhook payload")
		return
	}
	
	if err := h.service.PayoutWebhook(r.Context(), payload); err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	if err := json.JsonResponse(w, http.StatusOK, "success"); err != nil {
		json.InternalServerError(w, r, err)
	}
}

// Withdraw godoc
//
//	@Summary		Initiates a withdrawal
//	@Description	Initiates a withdrawal
//	@Tags			wallet
//	@Accept			json
//	@Produce		json
//	@Param        	Idempotency-Key		header    string    true   	"Idempotency-Key must be set for valid response"
//	@Param			payload	body		WithdrawalRequest	true	"Withdrawal amount"
//	@Success		202		{object}	string		"Withdrawal initiated"
//	@Failure		401		{object}	string "unauthorized"
//	@Failure		404		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/wallet/withdraw [post]
func (h *handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	var req WithdrawalRequest
	if err := json.ReadJSON(w,r,&req); err != nil {
		json.InternalServerError(w,r,err)
		return
	}

	if err := json.Validate.Struct(req); err != nil {
		json.BadRequestResponse(w, r, err)
		return
	}

	user := r.Context().Value("user").(*store.User)

	session,  err := h.service.Withdraw(r.Context(), req, user)
	if err != nil {
		switch {
		case errors.Is(err, ErrMaxWithdrawalAmountExceeded):
			json.JsonResponse(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, store.ErrBalanceNotFound):
			json.NotFoundResponse(w, r, err)
		case errors.Is(err, ErrInsufficientBalance):
			json.JsonResponse(w, http.StatusPaymentRequired, err.Error())
		default:
			json.InternalServerError(w,r,err)
		}
		
		return
	}

	if err := json.JsonResponse(w, http.StatusAccepted, session); err != nil {
		json.InternalServerError(w,r,err)
	}
}

// Transfer godoc
//
//	@Summary		P2P money transfer
//	@Description	P2P money transfer
//	@Tags			wallet
//	@Accept			json
//	@Produce		json
//	@Param        	Idempotency-Key		header    string    true   	"Idempotency-Key must be set for valid response"
//	@Param			payload	body		TransferRequest	true	"Transfer request"
//	@Success		202		{object}	string		"Transfer succeeded"
//	@Failure		401		{object}	string "unauthorized"
//	@Failure		402		{object}	error
//	@Failure		404		{object}	error
//	@Failure		422		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/wallet/transfer [post]
func (h *handler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest
	if err := json.ReadJSON(w, r, &req); err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	if err := json.Validate.Struct(req); err != nil {
		json.BadRequestResponse(w, r, err)
		return
	}

	user := r.Context().Value("user").(*store.User)

	if err := h.service.Transfer(r.Context(), req, user); err != nil {
		switch {
		case errors.Is(err, ErrMaxTransferAmountExceeded):
			json.JsonResponse(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, ErrInsufficientBalance): 
			json.JsonResponse(w, http.StatusPaymentRequired, err.Error())
		case errors.Is(err, ErrSelfTransfer):
			json.JsonResponse(w, http.StatusUnprocessableEntity, err.Error())
		case errors.Is(err, store.ErrBalanceNotFound):
			json.NotFoundResponse(w, r, err)
		default:
			json.InternalServerError(w, r, err)
		}
		
		return
	}

	if err := json.JsonResponse(w, http.StatusOK, "success"); err != nil {
		json.InternalServerError(w, r, err)
	}
}

func (h *handler) GetWallet(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*store.User)

	balance, err := h.service.GetWallet(r.Context(), user.ID)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrBalanceNotFound):
			json.NotFoundResponse(w, r, err)
		default:
			json.InternalServerError(w, r, err)
		}
		
		return
	}

	if err := json.JsonResponse(w, http.StatusOK, balance); err != nil {
		json.InternalServerError(w, r, err)
	}
}

// 	GetTransactionHistory godoc
//
//	@Summary		Get user's transaction history
//	@Description	Get user's transaction history
//	@Tags			wallet
//	@Produce		json
//	@Success		200		{object}	string		"transactions"
//	@Failure		401		{object}	string      "unauthorized"
//	@Failure		404		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/wallet/transactions [get]
func (h *handler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*store.User)

	transactions, err := h.service.GetTransactionHistory(r.Context(), user.IdentityID)
	if err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	if err := json.JsonResponse(w, http.StatusOK, transactions); err != nil {
		json.InternalServerError(w, r, err)
	}
}
