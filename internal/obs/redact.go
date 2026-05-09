package obs

import "strings"

var secretKeys = []string{"SID", "HSID", "SSID", "APISID", "SAPISID", "SNlM0e", "FdrFJe", "cookie", "authorization"}

func Redact(s string) string {
	redacted := s
	for _, key := range secretKeys {
		redacted = strings.ReplaceAll(redacted, key+"=", key+"=<redacted>")
	}
	return redacted
}
