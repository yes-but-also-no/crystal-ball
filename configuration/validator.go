package configuration

import (
	"fmt"
	"net"
	"net/url"
	"strings"
)

var privateIPBlocks []*net.IPNet

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"169.254.0.0/16", // RFC3927 link-local
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
		"fc00::/7",       // IPv6 unique local addr
	} {
		_, block, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(fmt.Errorf("parse error on %q: %v", cidr, err))
		}
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	for _, block := range privateIPBlocks {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

func checkContains(hosts []string, host string) bool {
	for _, h := range hosts {
		if h == host {
			return true
		}
	}
	return false
}

func (f *Filter) ValidateURL(path string) (bool, error) {
	u, err := url.Parse(path)
	if err != nil {
		return false, err
	}
	if u.Scheme != "https" {
		return false, nil
	}
	requireMatch := f.Mode == "whitelist"
	// Trim port from host
	host := strings.SplitN(u.Host, ":", 2)[0]
	hasHost := checkContains(f.Domains, host)
	if hasHost != requireMatch {
		return false, nil
	}
	ip := net.ParseIP(host)
	if ip != nil {
		return !isPrivateIP(ip), nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return false, err
	}
	for _, ip = range ips {
		if isPrivateIP(ip) {
			return false, nil
		}
	}
	return true, nil
}
