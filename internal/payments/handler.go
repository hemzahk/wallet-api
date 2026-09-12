package payments

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hemzahk/wallet-api/internal/json"
	"github.com/hemzahk/wallet-api/internal/store"
)

type handler struct {
	service Service
}

func NewHandler(service Service) *handler {
	return &handler{
		service: service,
	}
}

type CheckoutSessionRequest struct {
	Amount string `json:"amount" validate:"required"`
}

// CreateCheckoutSession godoc
//
//	@Summary		Creates a checkout session
//	@Description	Creates a checkout session
//	@Tags			checkout-sessions
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		CheckoutSessionRequest	true	"Checkout session request"
//	@Success		200		{object}	string		"Checkout session created"
//	@Failure		403		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/checkout-sessions [post]
func (h *handler) CreateCheckoutSession(w http.ResponseWriter, r *http.Request) {
	var req CheckoutSessionRequest
	if err := json.ReadJSON(w, r, &req); err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	if err := json.Validate.Struct(req); err != nil {
		json.BadRequestResponse(w, r, err)
		return
	}

	user := r.Context().Value("user").(*store.User)

	token, err := h.service.CreateCheckoutSession(r.Context(), req, user)
	if err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	res := struct {
		Token string `json:"token"`
	} {
		Token: token,
	}

	if err := json.JsonResponse(w, http.StatusOK, res); err != nil {
		json.InternalServerError(w, r, err)
	}
}

//  Pay godoc
//
//	@Summary		Pays 
//	@Description	Pays
//	@Tags			checkout-sessions
//	@Produce		json
//	@Param        	Idempotency-Key		header    string    true   	"Idempotency-Key must be set for valid response"
//	@Param			token	path		string	true	"Checkout session token"
//	@Success		200		{object}	string		"Payment succeeded"
//	@Failure		402		{object}	error
//	@Failure		403		{object}	error
//	@Failure		404		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/checkout-sessions/{token}/pay [post]
func (h *handler) Pay(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	user := r.Context().Value("user").(*store.User)

	if err := h.service.Pay(r.Context(), token, user); err != nil {
		switch {
		case errors.Is(err,store.ErrCheckoutSessionNotFound ):
			json.NotFoundResponse(w,r,err)
		case errors.Is(err,ErrInsufficientBalance):
			json.JsonResponse(w, http.StatusPaymentRequired, err.Error())
		default:
			json.InternalServerError(w, r, err)
		}

		return		
	}

	if err := json.JsonResponse(w, http.StatusOK, "success"); err != nil {
		json.InternalServerError(w, r, err)
	}
}

type RefundRequest struct {
	TransactionRef string `json:"transaction_ref" validate:"required"`
}

//  RequestRefund godoc
//
//	@Summary		Request a refund 
//	@Description	Request a refund
//	@Tags			refunds
//	@Produce		json
//	@Param        	Idempotency-Key		header    string    true   	"Idempotency-Key must be set for valid response"
//	@Param			payload	body		RefundRequest	true	"Refund request"
//	@Success		202		{object}	string		"Refund accepted"
//	@Failure		403		{object}	error
//	@Failure		404		{object}	error
//	@Failure		422		{object}	error
//	@Failure		500		{object}	error
//	@Security		ApiKeyAuth
//	@Router			/refunds [post]
func (h *handler) RequestRefund(w http.ResponseWriter, r *http.Request) {
	var req RefundRequest
	if err := json.ReadJSON(w, r, &req); err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	if err := json.Validate.Struct(req); err != nil {
		json.BadRequestResponse(w, r, err)
		return
	}
	
	if err := h.service.RequestRefund(r.Context(), req); err != nil {
		switch {
		case errors.Is(err, ErrNonRefundable):
			json.JsonResponse(w, http.StatusUnprocessableEntity, err.Error())
		default:
			json.InternalServerError(w, r, err)
		}
		
		return
	}

	if err := json.JsonResponse(w, http.StatusAccepted, map[string]string{"status": "pending"}); err != nil {
		json.InternalServerError(w, r, err)
	}	
}

