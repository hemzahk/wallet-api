package payments

import (
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

func (h *handler) Pay(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	user := r.Context().Value("user").(*store.User)

	if err := h.service.Pay(r.Context(), token, user); err != nil {
		switch err{
		case store.ErrCheckoutSessionNotFound:
			json.NotFoundResponse(w,r,err)
		case ErrInsufficientBalance:
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

/*func (h *handler) GetCheckoutSession(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	session, err := h.service.GetCheckoutSession(r.Context(), token)
	if err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	bf := new(big.Float).SetInt(session.Amount)

	// Divide by 100
	divisor := big.NewFloat(100)
	dzdAmount := new(big.Float).Quo(bf, divisor)

	res := struct {
		MerchantID string `json:"merchant_id"`
		Amount string `json:"amount"`
	}{
		MerchantID: session.MerchantID.String(),
		Amount : dzdAmount.String(),
	}

	if err := json.JsonResponse(w, http.StatusOK, res); err != nil {
		json.InternalServerError(w, r, err)
	}
}*/
