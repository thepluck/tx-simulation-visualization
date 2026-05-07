package runid

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"
)

func New() string {
	var raw [4]byte
	if _, err := rand.Read(raw[:]); err == nil {
		return time.Now().UTC().Format("20060102T150405.000000000") + "-" + hex.EncodeToString(raw[:])
	}
	return time.Now().UTC().Format("20060102T150405.000000000") + "-" + strconv.FormatInt(time.Now().UnixNano(), 36)
}
