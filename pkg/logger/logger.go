package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	logLevel    string
)

// Init 初始化日志
func Init(level, logFile string) {
	logLevel = level

	var output *os.File
	var err error

	if logFile != "" {
		// 确保日志目录存在
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Fatalf("Failed to create log directory: %v", err)
		}

		// 打开日志文件
		output, err = os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalf("Failed to open log file: %v", err)
		}
	} else {
		output = os.Stdout
	}

	infoLogger = log.New(output, "[INFO] ", log.LstdFlags)
	errorLogger = log.New(output, "[ERROR] ", log.LstdFlags)
	debugLogger = log.New(output, "[DEBUG] ", log.LstdFlags)
}

// Info 输出信息日志
func Info(format string, v ...interface{}) {
	if infoLogger == nil {
		Init("info", "")
	}
	msg := fmt.Sprintf(format, v...)
	infoLogger.Print(Sanitize(msg))
}

// Error 输出错误日志
func Error(format string, v ...interface{}) {
	if errorLogger == nil {
		Init("info", "")
	}
	msg := fmt.Sprintf(format, v...)
	errorLogger.Print(Sanitize(msg))
}

// Debug 输出调试日志
func Debug(format string, v ...interface{}) {
	if logLevel == "debug" {
		if debugLogger == nil {
			Init("debug", "")
		}
		msg := fmt.Sprintf(format, v...)
		debugLogger.Print(Sanitize(msg))
	}
}

// Fatal 输出致命错误并退出
func Fatal(format string, v ...interface{}) {
	if errorLogger == nil {
		Init("info", "")
	}
	errorLogger.Fatalf(format, v...)
}

// LogRequest 记录 HTTP 请求
func LogRequest(method, path string, statusCode int, duration time.Duration) {
	Info("%s %s - %d - %v", method, path, statusCode, duration)
}

// LogPayment 记录支付相关日志
func LogPayment(orderID, provider, action string, details interface{}) {
	Info("[Payment] OrderID=%s Provider=%s Action=%s Details=%v", orderID, provider, action, details)
}

// LogWebhook 记录 Webhook 日志
func LogWebhook(provider, event string, success bool, details interface{}) {
	status := "SUCCESS"
	if !success {
		status = "FAILED"
	}
	Info("[Webhook] Provider=%s Event=%s Status=%s Details=%v", provider, event, status, details)
}

// LogNotify 记录业务通知日志
func LogNotify(orderID, url string, retry int, success bool, err error) {
	status := "SUCCESS"
	errMsg := ""
	if !success {
		status = "FAILED"
		if err != nil {
			errMsg = fmt.Sprintf(" Error=%v", err)
		}
	}
	Info("[Notify] OrderID=%s URL=%s Retry=%d Status=%s%s", orderID, url, retry, status, errMsg)
}
