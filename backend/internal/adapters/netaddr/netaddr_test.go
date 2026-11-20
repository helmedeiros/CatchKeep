package netaddr_test

import (
	"errors"
	"net"
	"slices"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/netaddr"
)

func ifaces(addrs ...string) func() ([]net.Addr, error) {
	return func() ([]net.Addr, error) {
		var out []net.Addr
		for _, a := range addrs {
			_, n, _ := net.ParseCIDR(a)
			ip, _, _ := net.ParseCIDR(a)
			out = append(out, &net.IPNet{IP: ip, Mask: n.Mask})
		}
		return out, nil
	}
}

func listen(addr string) net.Addr {
	a, _ := net.ResolveTCPAddr("tcp", addr)
	return a
}

func TestURLs(t *testing.T) {
	network := ifaces("127.0.0.1/8", "192.168.1.20/24", "10.0.0.5/8", "169.254.3.4/16", "fe80::1/64", "203.0.113.9/24")
	for name, tc := range map[string]struct {
		listen string
		want   []string
	}{
		"only this computer": {"127.0.0.1:8080", nil},
		"every network":      {"0.0.0.0:8080", []string{"http://10.0.0.5:8080", "http://192.168.1.20:8080"}},
		"one address":        {"192.168.1.20:9000", []string{"http://192.168.1.20:9000"}},
	} {
		if got := netaddr.URLs(listen(tc.listen), network); !slices.Equal(got, tc.want) {
			t.Errorf("%s: URLs = %v, want %v", name, got, tc.want)
		}
	}
	broken := func() ([]net.Addr, error) { return nil, errors.New("no network") }
	if got := netaddr.URLs(listen("0.0.0.0:8080"), broken); got != nil {
		t.Errorf("URLs = %v without a network", got)
	}
}

func TestHostNames(t *testing.T) {
	network := ifaces("127.0.0.1/8", "192.168.1.20/24")
	got := netaddr.HostNames(listen("0.0.0.0:8080"), network, "Ashs-MacBook")
	want := []string{"192.168.1.20", "ashs-macbook", "ashs-macbook.local"}
	if !slices.Equal(got, want) {
		t.Fatalf("HostNames = %v, want %v", got, want)
	}
	if got := netaddr.HostNames(listen("127.0.0.1:8080"), network, "Ashs-MacBook.local"); !slices.Equal(got, []string{"ashs-macbook.local"}) {
		t.Fatalf("HostNames = %v; listening only on this computer keeps its name", got)
	}
}
