package wechat

import (
	"time"
)

// parseWechatTime 解析微信时间格式
// 微信时间格式：2018-06-08T10:34:56+08:00
func parseWechatTime(timeStr string) (time.Time, error) {
	return time.Parse(time.RFC3339, timeStr)
}
