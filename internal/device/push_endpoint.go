package device

import (
	"net"
	"net/url"
	"strings"
)

const (
	pushEndpointSchemeHTTPS = "https"
	pushHostFCM             = "fcm.googleapis.com"
	pushHostMozilla         = "updates.push.services.mozilla.com"
	pushHostSuffixWNS       = ".notify.windows.com"
	pushHostSuffixApple     = ".push.apple.com"
)

// Push endpoint allowlist sources:
// - Chromium / FCM examples:
//   https://web.dev/articles/push-notifications-overview
//   https://developer.chrome.com/blog/push-notifications-on-the-open-web/
// - Firefox push service examples:
//   https://blog.mozilla.org/services/2016/04/04/using-vapid-with-webpush/
//   https://mozilla.github.io/ecosystem-platform/relying-parties/how-tos/device-registration
// - WNS host guidance:
//   https://learn.microsoft.com/en-us/windows/apps/develop/notifications/push-notifications/wns-overview
// - Apple Web Push network guidance:
//   https://developer.apple.com/videos/play/wwdc2022/10098/?time=576

// validatePushEndpoint enforces a tight provider allowlist at registration time so
// outbound delivery never has to deal with arbitrary user-controlled hosts later.
func validatePushEndpoint(rawEndpoint string) (string, error) {
	parsed, err := url.Parse(rawEndpoint)
	if err != nil {
		return "", ErrInvalidPushEndpoint
	}
	if parsed.Scheme != pushEndpointSchemeHTTPS {
		return "", ErrInvalidPushEndpoint
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return "", ErrInvalidPushEndpoint
	}
	if isAllowedPushHost(host) {
		return host, nil
	}

	return host, ErrInvalidPushEndpoint
}

func isAllowedPushHost(host string) bool {
	normalizedHost := strings.ToLower(strings.TrimSuffix(host, "."))
	if normalizedHost == "" {
		return false
	}
	if net.ParseIP(normalizedHost) != nil {
		return false
	}
	if normalizedHost == pushHostFCM || normalizedHost == pushHostMozilla {
		return true
	}
	return strings.HasSuffix(normalizedHost, pushHostSuffixWNS) || strings.HasSuffix(normalizedHost, pushHostSuffixApple)
}
