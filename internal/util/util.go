package util

import (
	"github.com/google/uuid"
)

// RandomEmail generates a random email suitable for testing
func RandomEmail() string {
	return uuid.New().String() + "@example.domain"
}
