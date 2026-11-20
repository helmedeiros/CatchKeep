// Package netaddr works out the addresses other devices on the network use
// to reach this computer.
package netaddr

import (
	"fmt"
	"net"
	"slices"
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
		return []string{url(tcp.IP, tcp.Port)}
	}
	addrs, err := interfaceAddrs()
	if err != nil {
		return nil
	}
	var out []string
	for _, a := range addrs {
		n, ok := a.(*net.IPNet)
		if ok && n.IP.To4() != nil && n.IP.IsPrivate() {
			out = append(out, url(n.IP, tcp.Port))
		}
	}
	slices.Sort(out)
	return out
}

func url(ip net.IP, port int) string {
	return fmt.Sprintf("http://%s", net.JoinHostPort(ip.String(), fmt.Sprint(port)))
}
