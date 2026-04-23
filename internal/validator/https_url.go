package validator

import (
	"fmt"
	"net"
	"net/url"
)

// HTTPSURL returns an error if value is not a valid HTTPS URL without private/internal IPs.
func HTTPSURL(field, value string) error {
	if value == "" {
		return nil // empty is handled by "required" validator
	}

	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("%s: invalid URL", field)
	}

	if u.Scheme != "https" {
		return fmt.Errorf("%s: must be HTTPS", field)
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("%s: invalid URL host", field)
	}

	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() {
			return fmt.Errorf("%s: cannot use private or loopback IP", field)
		}
	}

	return nil
}
