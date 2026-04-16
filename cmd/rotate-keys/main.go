package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func main() {
	keyType := flag.String("type", "", "Key type to rotate (admin-api-key, alipay, wechat, db, all)")
	verify := flag.Bool("verify", false, "Verify key after rotation")
	emergency := flag.Bool("emergency", false, "Emergency rotation (disable old keys immediately)")
	checkAge := flag.Bool("check-age", false, "Check age of all keys")

	flag.Parse()

	if *checkAge {
		checkKeyAge()
		return
	}

	if *keyType == "" {
		log.Fatal("Please specify --type (admin-api-key, alipay, wechat, db, all)")
	}

	switch *keyType {
	case "admin-api-key":
		rotateAdminAPIKey(*emergency)
	case "alipay":
		rotateAlipayKey(*verify)
	case "wechat":
		rotateWechatKey(*verify)
	case "db":
		rotateDBPassword(*verify)
	case "all":
		rotateAllKeys(*emergency)
	default:
		log.Fatalf("Unknown key type: %s", *keyType)
	}
}

func rotateAdminAPIKey(emergency bool) {
	log.Println("Rotating Admin API Key...")

	// 生成新的 API Key (32 字节 = 256 位)
	newKey := generateSecureKey(32)

	if emergency {
		log.Println("⚠️  EMERGENCY MODE: Old key will be disabled immediately")
	}

	fmt.Println("\n=== New Admin API Key ===")
	fmt.Printf("ADMIN_API_KEY=%s\n", newKey)
	fmt.Println("\nPlease update your environment variables and restart the service.")
	fmt.Println("Keep the old key for 7 days as backup, then delete it.")

	// 记录轮换日志
	logRotation("admin-api-key", emergency)
}

func rotateAlipayKey(verify bool) {
	log.Println("Rotating Alipay Key...")
	fmt.Println("\n=== Alipay Key Rotation Steps ===")
	fmt.Println("1. Generate new RSA key pair:")
	fmt.Println("   openssl genrsa -out alipay_private_key.pem 2048")
	fmt.Println("   openssl rsa -in alipay_private_key.pem -pubout -out alipay_public_key.pem")
	fmt.Println("\n2. Upload alipay_public_key.pem to Alipay Open Platform")
	fmt.Println("3. Update ALIPAY_APP_PRIVATE_KEY environment variable")
	fmt.Println("4. Restart the service")

	if verify {
		fmt.Println("\n5. Verify by making a test payment")
	}

	logRotation("alipay", false)
}

func rotateWechatKey(verify bool) {
	log.Println("Rotating WeChat Pay Key...")
	fmt.Println("\n=== WeChat Pay Key Rotation Steps ===")
	fmt.Println("1. Log in to WeChat Pay Merchant Platform")
	fmt.Println("2. Navigate to Account Center > API Security")
	fmt.Println("3. Generate new API v3 key (32 characters)")
	fmt.Println("4. Download new merchant certificate")
	fmt.Println("5. Update environment variables:")
	fmt.Println("   - WECHAT_API_V3_KEY")
	fmt.Println("   - WECHAT_CERT_PATH")
	fmt.Println("   - WECHAT_KEY_PATH")
	fmt.Println("6. Restart the service")

	if verify {
		fmt.Println("\n7. Verify by making a test payment")
	}

	logRotation("wechat", false)
}

func rotateDBPassword(verify bool) {
	log.Println("Rotating Database Password...")

	// 生成新的数据库密码 (24 字节)
	newPassword := generateSecureKey(24)

	fmt.Println("\n=== New Database Password ===")
	fmt.Printf("DB_PASSWORD=%s\n", newPassword)
	fmt.Println("\n=== Database Password Rotation Steps ===")
	fmt.Println("1. Connect to PostgreSQL:")
	fmt.Println("   psql -U postgres")
	fmt.Println("\n2. Update password:")
	fmt.Printf("   ALTER USER gopay WITH PASSWORD '%s';\n", newPassword)
	fmt.Println("\n3. Update DB_PASSWORD environment variable")
	fmt.Println("4. Restart the service")

	if verify {
		fmt.Println("\n5. Verify database connection")
	}

	logRotation("db", false)
}

func rotateAllKeys(emergency bool) {
	log.Println("Rotating ALL keys...")
	if emergency {
		log.Println("⚠️  EMERGENCY MODE ACTIVATED")
	}

	rotateAdminAPIKey(emergency)
	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	rotateAlipayKey(false)
	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	rotateWechatKey(false)
	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	rotateDBPassword(false)
}

func checkKeyAge() {
	log.Println("Checking key age...")

	// 这里需要从密钥管理系统或日志中读取密钥创建时间
	// 简化示例：从环境变量的修改时间推断
	fmt.Println("\n=== Key Age Report ===")
	fmt.Println("Note: This is a simplified check. In production, use a proper key management system.")
	fmt.Println("\nRecommended rotation intervals:")
	fmt.Println("- Admin API Key: 90 days")
	fmt.Println("- Payment channel keys: 90 days")
	fmt.Println("- Database password: 180 days")
	fmt.Println("\nPlease check your key management system for actual key ages.")
}

func generateSecureKey(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		log.Fatalf("Failed to generate secure key: %v", err)
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}

func logRotation(keyType string, emergency bool) {
	logFile := "key-rotation.log"
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("Warning: Failed to write rotation log: %v", err)
		return
	}
	defer f.Close()

	timestamp := time.Now().Format(time.RFC3339)
	mode := "normal"
	if emergency {
		mode = "EMERGENCY"
	}

	logEntry := fmt.Sprintf("[%s] Rotated %s (mode: %s)\n", timestamp, keyType, mode)
	if _, err := f.WriteString(logEntry); err != nil {
		log.Printf("Warning: Failed to write rotation log: %v", err)
	}
}
