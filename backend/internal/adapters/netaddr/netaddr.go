// Package netaddr works out the addresses other devices on the network use
// to reach this computer.
package netaddr

import (
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"
)

// URLs returns the web addresses other devices can open to reach a server
// listening on listen. A server listening only on this computer has none;
// one listening everywhere is reachable at each private network address
// (home and office networks), not at public or link-local ones.
func URLs(listen net.Addr, interfaceAddrs func() ([]net.Addr, error)) []string {
	tcp, ok := listen.(*net.TCPAddr)
	if !ok || tcp.IP.IsLoopback() {
		return nil
	}
	if !tcp.IP.IsUnspecified() {
		return []string{webAddress(tcp.IP, tcp.Port)}
	}
	addrs, err := interfaceAddrs()
	if err != nil {
		return nil
	}
	var out []string
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if ok && n.IP.To4() != nil && n.IP.IsPrivate() {
			out = append(out, webAddress(n.IP, tcp.Port))
		}
	}
	slices.Sort(out)
	return out
}

func webAddress(ip net.IP, port int) string {
	return fmt.Sprintf("http://%s", net.JoinHostPort(ip.String(), fmt.Sprint(port)))
}

// HostNames are the names other devices may use to reach this computer:
// its network addresses and its name, with ".local" as many networks
// resolve it, lowercased and without ports.
func HostNames(listen net.Addr, interfaceAddrs func() ([]net.Addr, error), hostname string) []string {
	var out []string
	for _, u := range URLs(listen, interfaceAddrs) {
		if parsed, err := url.Parse(u); err == nil {
			out = append(out, parsed.Hostname())
		}
	}
	if name := strings.ToLower(strings.TrimSuffix(hostname, ".local")); name != "" {
		if strings.HasSuffix(strings.ToLower(hostname), ".local") {
			return append(out, name+".local")
		}
		out = append(out, name, name+".local")
	}
	return out
}
