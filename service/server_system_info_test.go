package service

import (
	"reflect"
	"testing"

	gopsnet "github.com/shirou/gopsutil/v4/net"
)

func TestGetSystemInfoHandlesShortInterfaceDataIssue23(t *testing.T) {
	original := systemInfoInterfaces
	systemInfoInterfaces = func() (gopsnet.InterfaceStatList, error) {
		return gopsnet.InterfaceStatList{
			{
				Name:  "empty-flags",
				Flags: []string{},
				Addrs: []gopsnet.InterfaceAddr{
					{Addr: "10.0.0.1/24"},
				},
			},
			{
				Name:  "one-up-flag",
				Flags: []string{"up"},
				Addrs: []gopsnet.InterfaceAddr{
					{Addr: ""},
					{Addr: "x"},
					{Addr: "8.8.8.8/32"},
					{Addr: "192.168.1.8/24"},
					{Addr: "2001:4860:4860::8888/128"},
					{Addr: "fe80::1/64"},
				},
			},
			{
				Name:  "loopback-late",
				Flags: []string{"up", "broadcast", "loopback"},
				Addrs: []gopsnet.InterfaceAddr{
					{Addr: "127.0.0.1/8"},
					{Addr: "2001:db8::2/64"},
				},
			},
		}, nil
	}
	t.Cleanup(func() {
		systemInfoInterfaces = original
	})

	info := (&ServerService{}).GetSystemInfo()

	ipv4Value, ok := info["ipv4"]
	if !ok {
		t.Fatal("expected ipv4 key")
	}
	ipv4, ok := ipv4Value.([]string)
	if !ok {
		t.Fatalf("expected ipv4 []string, got %T", ipv4Value)
	}
	if !containsString(ipv4, "8.8.8.8/32") {
		t.Fatalf("expected usable single-flag IPv4, got %#v", ipv4)
	}
	if containsString(ipv4, "192.168.1.8/24") {
		t.Fatalf("unexpected private IPv4 address: %#v", ipv4)
	}
	if containsString(ipv4, "10.0.0.1/24") {
		t.Fatalf("unexpected IPv4 from interface without up flag: %#v", ipv4)
	}
	if containsString(ipv4, "127.0.0.1/8") {
		t.Fatalf("unexpected IPv4 from loopback interface: %#v", ipv4)
	}

	ipv6Value, ok := info["ipv6"]
	if !ok {
		t.Fatal("expected ipv6 key")
	}
	ipv6, ok := ipv6Value.([]string)
	if !ok {
		t.Fatalf("expected ipv6 []string, got %T", ipv6Value)
	}
	if !containsString(ipv6, "2001:4860:4860::8888/128") {
		t.Fatalf("expected global IPv6 address, got %#v", ipv6)
	}
	if containsString(ipv6, "2001:db8::2/64") {
		t.Fatalf("unexpected IPv6 from loopback interface: %#v", ipv6)
	}
	if containsString(ipv6, "fe80::1/64") {
		t.Fatalf("unexpected link-local IPv6 address: %#v", ipv6)
	}
	if containsString(ipv6, "x") {
		t.Fatalf("unexpected invalid IPv6 address: %#v", ipv6)
	}
}

func TestGetSystemInfoFiltersNonPublicAddressesIssue24(t *testing.T) {
	original := systemInfoInterfaces
	systemInfoInterfaces = func() (gopsnet.InterfaceStatList, error) {
		return gopsnet.InterfaceStatList{
			{
				Name:  "mixed-addresses",
				Flags: []string{"up", "broadcast"},
				Addrs: []gopsnet.InterfaceAddr{
					{Addr: "8.8.8.8/32"},
					{Addr: "10.0.0.5/8"},
					{Addr: "172.16.0.5/12"},
					{Addr: "192.168.1.5/24"},
					{Addr: "169.254.1.5/16"},
					{Addr: "127.0.0.1/8"},
					{Addr: "2001:4860:4860::8888/128"},
					{Addr: "fd00::1/64"},
					{Addr: "fe80::1/64"},
					{Addr: "::1/128"},
					{Addr: "::/128"},
					{Addr: "ff02::1/128"},
					{Addr: ""},
					{Addr: "x"},
				},
			},
		}, nil
	}
	t.Cleanup(func() {
		systemInfoInterfaces = original
	})

	info := (&ServerService{}).GetSystemInfo()

	ipv4, ok := info["ipv4"].([]string)
	if !ok {
		t.Fatalf("expected ipv4 []string, got %T", info["ipv4"])
	}
	if want := []string{"8.8.8.8/32"}; !reflect.DeepEqual(ipv4, want) {
		t.Fatalf("expected only public IPv4 addresses %#v, got %#v", want, ipv4)
	}

	ipv6, ok := info["ipv6"].([]string)
	if !ok {
		t.Fatalf("expected ipv6 []string, got %T", info["ipv6"])
	}
	if want := []string{"2001:4860:4860::8888/128"}; !reflect.DeepEqual(ipv6, want) {
		t.Fatalf("expected only public IPv6 addresses %#v, got %#v", want, ipv6)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
