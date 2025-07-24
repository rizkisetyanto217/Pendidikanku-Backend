package constants

import "github.com/google/uuid"

var (
	// UUID untuk user non-login (dummy user)
	DummyUserID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
)