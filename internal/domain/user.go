package domain

import "time"

type User struct {
	UserID      string    `firestore:"userId"`
	Email       string    `firestore:"email"`
	DisplayName string    `firestore:"displayName"`
	FirstName   string    `firestore:"firstName"`
	LastName    string    `firestore:"lastName"`
	Role        string    `firestore:"role"`
	Status      string    `firestore:"status"`
	Version     int       `firestore:"version"`
	CreatedAt   time.Time `firestore:"createdAt"`
	UpdatedAt   time.Time `firestore:"updatedAt"`
}
