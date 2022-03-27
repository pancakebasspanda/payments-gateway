package identifier

import (
	"github.com/google/uuid"
	"strings"
)

func NewUUID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}
