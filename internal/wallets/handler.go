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

type TopupDTO struct {
	Amount string `json:"amount" validate:"required"`
}

type WithdrawalDTO struct {
	Number     string `json:"number"`
	BankName   string  `json:"bank_name"`
	Amount string `json:"amount" validate:"required"`
	Reference string `json:"reference" validate:"required"`
}

type PayoutWebhookDTO struct {
	EventID  string `json:"event_id"`
	SourceID string `json:"source_id"` // for our PSP 645b44fb-314f-4e34-8106-17b09cc9660a
	GatewayRef string          `json:"gateway_ref"`
	Status     string          `json:"status"`
	Amount     string 		   `json:"amount"`
}

func (h *handler) Topup(w http.ResponseWriter, r *http.Request) {
	var payload TopupDTO
	if err := json.ReadJSON(w, r, &payload); err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	if err := json.Validate.Struct(payload); err != nil {
		json.BadRequestResponse(w, r, err)
		return
	}

	user := r.Context().Value("user").(*store.User)

	session, err := h.service.Topup(r.Context(), payload, user)
	if err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	if err := json.JsonResponse(w, http.StatusOK, session); err != nil {
		json.InternalServerError(w, r, err)
	}
}

type TopupWebhookDTO struct {
	EventID  string `json:"event_id"`
	SourceID string `json:"source_id"` // for our PSP 645b44fb-314f-4e34-8106-17b09cc9660a
	GatewayRef string          `json:"gateway_ref"`
	Status     string          `json:"status"`
	Amount     string 		   `json:"amount"`
}

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

	var payload TopupWebhookDTO
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

	var payload PayoutWebhookDTO
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

func (h *handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	var payload WithdrawalDTO
	if err := json.ReadJSON(w,r,&payload); err != nil {
		json.InternalServerError(w,r,err)
		return
	}

	if err := json.Validate.Struct(payload); err != nil {
		json.BadRequestResponse(w, r, err)
		return
	}

	user := r.Context().Value("user").(*store.User)

	session,  err := h.service.Withdraw(r.Context(), payload, user)
	if err != nil {
		json.InternalServerError(w,r,err)
		return
	}

	if err := json.JsonResponse(w, http.StatusAccepted, session); err != nil {
		json.InternalServerError(w,r,err)
	}
}

type TransferDTO struct {
	Email string `json:"email" validate:"required,email"`
	Amount string `json:"amount" validate:"required"`
	Reference string `json:"reference" validate:"required"`
}

func (h *handler) Transfer(w http.ResponseWriter, r *http.Request) {
	var payload TransferDTO
	if err := json.ReadJSON(w, r, &payload); err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	if err := json.Validate.Struct(payload); err != nil {
		json.BadRequestResponse(w, r, err)
		return
	}

	user := r.Context().Value("user").(*store.User)

	if err := h.service.Transfer(r.Context(), payload, user); err != nil {
		json.InternalServerError(w, r, err)
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
		json.InternalServerError(w, r, err)
		return
	}

	if err := json.JsonResponse(w, http.StatusOK, balance); err != nil {
		json.InternalServerError(w, r, err)
	}
}

type Transaction struct {
	Amount string `json:"amount"`
	Description string `json:"description"`
	CreatedAt string `json:"created_at"`
}

func (h *handler) GetTransactionHistory(w http.ResponseWriter, r *http.Request) {
	user := r.Context().Value("user").(*store.User)

	transactions, err := h.service.GetTransactionHistory(r.Context(), user.IdentityID)
	if err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	var response []Transaction
	for _, val := range transactions {
		amountAsFloat := toFloat(val.PreciseAmount)
		t := Transaction{
			Amount: amountAsFloat.String(),
			CreatedAt: val.CreatedAt.String(),
			Description: val.Description,
		}

		response = append(response, t)
	}

	if err := json.JsonResponse(w, http.StatusOK, response); err != nil {
		json.InternalServerError(w, r, err)
	}
}
