package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"io"
	"strconv"
	"time"

	"gopay/pkg/logger"
	"gopay/pkg/response"

	"github.com/gin-gonic/gin"
)

// SignatureAuth 签名验证中间件
//
// 业务方调用 API 时需要在请求头中携带：
//   - X-App-ID:     应用 ID
//   - X-Timestamp:  Unix 时间戳（秒）
//   - X-Nonce:      随机字符串（防重放）
//   - X-Signature:  HMAC-SHA256 签名
//
// 签名算法：
//
//	signature = HMAC-SHA256(app_secret, body + "\n" + timestamp + "\n" + nonce)
func SignatureAuth(db *sql.DB, nonceChecker NonceChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		appID := c.GetHeader("X-App-ID")
		timestampStr := c.GetHeader("X-Timestamp")
		nonce := c.GetHeader("X-Nonce")
		signature := c.GetHeader("X-Signature")

		if appID == "" || timestampStr == "" || nonce == "" || signature == "" {
			response.Unauthorized(c, "缺少签名参数（需要 X-App-ID, X-Timestamp, X-Nonce, X-Signature）")
			c.Abort()
			return
		}

		// 1. 验证时间戳（5 分钟内有效）
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			response.BadRequest(c, "X-Timestamp 格式错误")
			c.Abort()
			return
		}
		now := time.Now().Unix()
		if abs(now-timestamp) > 300 {
			response.Unauthorized(c, "签名已过期")
			c.Abort()
			return
		}

		// 2. 验证 nonce（防重放）
		if nonceChecker != nil && !nonceChecker.Check(nonce) {
			response.Unauthorized(c, "请求已被使用（nonce 重复）")
			c.Abort()
			return
		}

		// 3. 查询 app_secret
		var appSecret string
		var status string
		err = db.QueryRow(
			`SELECT app_secret, status FROM apps WHERE app_id = $1`, appID,
		).Scan(&appSecret, &status)
		if err != nil {
			logger.Error("Signature auth: app not found, app_id=%s", appID)
			response.Unauthorized(c, "应用不存在")
			c.Abort()
			return
		}
		if status != "active" {
			response.Unauthorized(c, "应用已禁用")
			c.Abort()
			return
		}

		// 4. 读取 body 并验证签名
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			response.BadRequest(c, "读取请求体失败")
			c.Abort()
			return
		}
		// 重新设置 body 供后续 handler 使用
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// 5. 计算期望签名: HMAC-SHA256(secret, body + "\n" + timestamp + "\n" + nonce)
		message := string(bodyBytes) + "\n" + timestampStr + "\n" + nonce
		mac := hmac.New(sha256.New, []byte(appSecret))
		mac.Write([]byte(message))
		expectedSig := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
			logger.Error("Signature auth: invalid signature, app_id=%s", appID)
			response.Unauthorized(c, "签名验证失败")
			c.Abort()
			return
		}

		// 6. 将 app_id 写入上下文，后续 handler 可直接使用
		c.Set("verified_app_id", appID)

		c.Next()
	}
}

// NonceChecker nonce 检查接口
type NonceChecker interface {
	Check(nonce string) bool
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
