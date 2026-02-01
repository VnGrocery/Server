package domain

import "time"

type User struct {
	UserID      string    `firestore:"userId"`
	Email       string    `firestore:"email"`
	DisplayName string    `firestore:"displayName"`
	Role        string    `firestore:"role"`
	CreatedAt   time.Time `firestore:"createdAt"`
	UpdatedAt   time.Time `firestore:"updatedAt"`
}
