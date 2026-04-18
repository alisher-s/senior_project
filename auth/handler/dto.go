package handler

import (
	"github.com/google/uuid"

	"github.com/nu/student-event-ticketing-platform/auth/model"
)

type UserDTO struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Role         string    `json:"role"`
	Roles        []string  `json:"roles"`
	PendingRoles []string  `json:"pending_roles,omitempty"`
}

type PatchMeRolesRequestDTO struct {
	Roles []string `json:"roles" validate:"required,min=1,max=8,dive,oneof=organizer"`
}

type MeRolesResponseDTO struct {
	User UserDTO `json:"user"`
}

func userToDTO(u model.User) UserDTO {
	d := UserDTO{
		ID:    u.ID,
		Email: u.Email,
		Role:  string(u.Role),
	}
	d.Roles = make([]string, len(u.ActiveRoles))
	for i, r := range u.ActiveRoles {
		d.Roles[i] = string(r)
	}
	if len(u.PendingRoles) > 0 {
		d.PendingRoles = make([]string, len(u.PendingRoles))
		for i, r := range u.PendingRoles {
			d.PendingRoles[i] = string(r)
		}
	}
	return d
}

type AuthResponseDTO struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	User         UserDTO  `json:"user"`
}

type RegisterRequestDTO struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type LoginRequestDTO struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type RefreshRequestDTO struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

