package utilsx

import (
	"strings"

	"github.com/gofrs/uuid"
)

func GenUuid() string {
	uuidStr := uuid.Must(uuid.NewV4()).String()
	return strings.ReplaceAll(uuidStr, "-", "")
}
