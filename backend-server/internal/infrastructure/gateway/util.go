// util.go - ゲートウェイ共通ユーティリティ関数
package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
