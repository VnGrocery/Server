package dto

import "time"

type UpdateUserRoleRequest struct {
	ExpectedVersion int    `json:"expectedVersion"`
	Role            string `json:"role"`
}

type UpdateUserStatusRequest struct {
	ExpectedVersion int    `json:"expectedVersion"`
	Status          string `json:"status"`
}

type AdminUserResponse struct {
	UserID      string    `json:"userId"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	Role        string    `json:"role"`
	Status      string    `json:"status"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AdminUserListResponse struct {
	Items []AdminUserResponse `json:"items"`
}
