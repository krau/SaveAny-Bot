package types

import (
	"crypto/md5"
	"encoding/hex"
)

func hashStr(s string) string {
	hash := md5.New()
	hash.Write([]byte(s))
	return hex.EncodeToString(hash.Sum(nil))
}
