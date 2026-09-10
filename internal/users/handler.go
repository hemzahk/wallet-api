package users

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
	return &handler{service: service}
}

type RegisterRequest struct {
	FirstName string `json:"first_name" validate:"required"`
	LastName string `json:"last_name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	PhoneNumber string `json:"phone_number"`
	Password string `json:"password" validate:"required"`
	DOB string `json:"dob"`
	Gender string `json:"gender"`
	State string `json:"state"`
	City string `json:"city"`
	Street string `json:"street"`
	PostCode string `json:"post_code"`
}

// registerCustomer godoc
//
//	@Summary		Registers a user
//	@Description	Registers a user
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		RegisterRequest	true	"User credentials"
//	@Success		201		{object}	string		"User registered"
//	@Failure		400		{object}	error
//	@Failure		500		{object}	error
//	@Router			/authentication/user [post]
func (h *handler) RegisterCustomer(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.ReadJSON(w,r, &req); err != nil {
		json.InternalServerError(w,r,err)
		return
	}

	if err := json.Validate.Struct(req); err != nil {
		json.BadRequestResponse(w,r,err)
		return
	}

	token, err := h.service.RegisterCustomer(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrDuplicateEmail):
			json.ConflictResponse(w, r, err)
		default:
			json.InternalServerError(w, r, err)
		}
		return
	}

	if err := json.JsonResponse(w, http.StatusCreated, token); err != nil {
		json.InternalServerError(w,r,err)
	}
}

// ActivateCustomer godoc
//
//	@Summary		Activates/Register a user
//	@Description	Activates/Register a user by invitation token
//	@Tags			users
//	@Produce		json
//	@Param			token	path		string	true	"Invitation token"
//	@Success		204		{string}	string	"User activated"
//	@Failure		404		{object}	error
//	@Failure		500		{object}	error
//	@Router			/users/activate/{token} [put]
func (h *handler) ActivateCustomer(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	if err := h.service.ActivateCustomer(r.Context(), token); err != nil {
		json.InternalServerError(w,r,err)
		return
	}

	if err := json.JsonResponse(w, http.StatusNoContent, ""); err != nil {
		json.InternalServerError(w,r,err)
	}
}

type RegisterMerchantRequest struct {
	BusinessName string `json:"business_name" validate:"required"`
	FirstName string `json:"first_name" validate:"required"`
	LastName string `json:"last_name" validate:"required"`
	Email string `json:"email" validate:"required,email"`
	PhoneNumber string `json:"phone_number"`
	Password string `json:"password" validate:"required"`
	DOB string `json:"dob"`
	Gender string `json:"gender"`
	State string `json:"state"`
	City string `json:"city"`
	Street string `json:"street"`
	PostCode string `json:"post_code"`	
}

// RegisterMerchant godoc
//
//	@Summary		Registers a merchant
//	@Description	Registers a merchant
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		RegisterMerchantRequest	true	"Merchant informations"
//	@Success		201		{object}	string		"Merchant registered"
//	@Failure		400		{object}	error
//	@Failure		500		{object}	error
//	@Router			/authentication/merchant [post]
func (h *handler) RegisterMerchant(w http.ResponseWriter, r *http.Request)  {
	var req RegisterMerchantRequest
	if err := json.ReadJSON(w,r, &req); err != nil {
		json.InternalServerError(w,r,err)
		return
	}

	if err := json.Validate.Struct(req); err != nil {
		json.BadRequestResponse(w,r,err)
		return
	}

	token, err := h.service.RegisterMerchant(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, store.ErrDuplicateEmail):
			json.ConflictResponse(w, r, err)
		default:
			json.InternalServerError(w, r, err)
		}
		return
	}

	if err := json.JsonResponse(w, http.StatusCreated, token); err != nil {
		json.InternalServerError(w,r,err)
	}
}

// ActivateMerchant godoc
//
//	@Summary		Activates/Register a merchant
//	@Description	Activates/Register a merchant by invitation token
//	@Tags			users
//	@Produce		json
//	@Param			token	path		string	true	"Invitation token"
//	@Success		204		{string}	string	"Merchant activated"
//	@Failure		404		{object}	error
//	@Failure		500		{object}	error
//	@Router			/merchants/activate/{token} [put]
func (h *handler) ActivateMerchant(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	if err := h.service.ActivateMerchant(r.Context(), token); err != nil {
		json.InternalServerError(w,r,err)
		return
	}

	if err := json.JsonResponse(w, http.StatusNoContent, ""); err != nil {
		json.InternalServerError(w,r,err)
	}
}

type CreateUserTokenRequest struct {
	Email string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// CreateToken godoc
//
//	@Summary		Creates a token
//	@Description	Creates a token for a user
//	@Tags			authentication
//	@Accept			json
//	@Produce		json
//	@Param			payload	body		CreateUserTokenRequest	true	"User credentials"
//	@Success		201		{string}	string					"Token"
//	@Failure		400		{object}	error
//	@Failure		401		{object}	error
//	@Failure		500		{object}	error
//	@Router			/authentication/token [post]
func (h *handler) CreateToken(w http.ResponseWriter, r *http.Request) {
	var req CreateUserTokenRequest
	if err := json.ReadJSON(w, r, &req); err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	if err := json.Validate.Struct(req); err != nil {
		json.BadRequestResponse(w, r, err)
		return
	}

	token, err := h.service.CreateToken(r.Context(), req)
	if err != nil {
		switch {
			case errors.Is(err, ErrUnauthorized):
				json.UnauthorizedErrorResponse(w,r,err)
			default:
				json.InternalServerError(w, r, err)
		}		
		return
	}

	if err := json.JsonResponse(w, http.StatusCreated, token); err != nil {
		json.InternalServerError(w, r, err)
	}
}
