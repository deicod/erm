package relay

import (
	"encoding/base64"
	"fmt"
	"strings"
)

func ToGlobalID(typ, id string) string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", typ, id)))
}

func FromGlobalID(gid string) (typ, id string, err error) {
	b, err := base64.StdEncoding.DecodeString(gid)
	if err != nil {
		return "", "", err
	}
	parts := strings.SplitN(string(b), ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid global id")
	}
	return parts[0], parts[1], nil
}
