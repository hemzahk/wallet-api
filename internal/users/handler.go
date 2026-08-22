package users

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hemzahk/wallet-api/internal/json"
)

type handler struct {
	service Service
}

func NewHandler(service Service) *handler {
	return &handler{service: service}
}

type RegisterDTO struct {
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

func (h *handler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var payload RegisterDTO

	if err := json.ReadJSON(w,r, &payload); err != nil {
		json.InternalServerError(w,r,err)
		return
	}

	if err := json.Validate.Struct(payload); err != nil {
		json.BadRequestResponse(w,r,err)
		return
	}

	token, err := h.service.Register(r.Context(), payload)
	if err != nil {
		json.InternalServerError(w,r,err)
		return
	}

	if err := json.JsonResponse(w, http.StatusCreated, token); err != nil {
		json.InternalServerError(w,r,err)
	}
}

func (h *handler) ActivateUser(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")

	if err := h.service.Activate(r.Context(), token); err != nil {
		json.InternalServerError(w,r,err)
		return
	}

	if err := json.JsonResponse(w, http.StatusNoContent, ""); err != nil {
		json.InternalServerError(w,r,err)
	}
}

type RegisterMerchantDTO struct {
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

func (h *handler) RegisterMerchant(w http.ResponseWriter, r *http.Request)  {
	var payload RegisterMerchantDTO
	if err := json.ReadJSON(w,r, &payload); err != nil {
		json.InternalServerError(w,r,err)
		return
	}

	if err := json.Validate.Struct(payload); err != nil {
		json.BadRequestResponse(w,r,err)
		return
	}

	token, err := h.service.RegisterMerchant(r.Context(), payload)
	if err != nil {
		json.InternalServerError(w,r,err)
		return
	}

	if err := json.JsonResponse(w, http.StatusCreated, token); err != nil {
		json.InternalServerError(w,r,err)
	}
}

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

type CreateUserTokenPayload struct {
	Email string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

func (h *handler) CreateToken(w http.ResponseWriter, r *http.Request) {
	var payload CreateUserTokenPayload
	if err := json.ReadJSON(w, r, &payload); err != nil {
		json.InternalServerError(w, r, err)
		return
	}

	if err := json.Validate.Struct(payload); err != nil {
		json.BadRequestResponse(w, r, err)
		return
	}

	token, err := h.service.CreateToken(r.Context(), payload)
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
