package provider

import (
	"net"
	"sort"

	"github.com/pkg/errors"
)

type (
	IP    = net.IP
	IPNet = net.IPNet
	CIDR  struct {
		IP IP
		*net.IPNet
	}
)

func ParseCIDR(addr string) (*CIDR, error) {
	ip, ipnet, err := net.ParseCIDR(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse cidr %q", addr)
	}
	return &CIDR{IP: ip, IPNet: ipnet}, nil
}

func ParseIP(addr string) IP {
	return net.ParseIP(addr)
}

func ToIPAddrs(in interface{}) []IP {
	var (
		inSlice = in.([]interface{})
		addrs   = make([]IP, len(inSlice))
	)
	for i, addr := range inSlice {
		addrs[i] = ParseIP(addr.(string))
	}
	return addrs
}

func SortIPAddress(priority map[*IPNet]int, in []IP) []IP {
	addrs := make([]IP, len(in))
	copy(addrs, in)
	if len(priority) == 0 {
		return addrs
	}
	sort.SliceStable(addrs, func(i, j int) bool {
		iaddr, jaddr := in[i], in[j]
		iweight, jweight := 0, 0
		for network, weight := range priority {
			if network.Contains(iaddr) {
				if weight > iweight {
					iweight = weight
				}
			}
			if network.Contains(jaddr) {
				if weight > jweight {
					jweight = weight
				}
			}
		}
		return iweight > jweight
	})
	return addrs
}

func FilterIPAddress(filter []*CIDR, in []IP) []IP {
	var (
		addrs = make([]IP, len(in))
		i     int
	)
	if len(filter) == 0 {
		copy(addrs, in)
		return addrs
	}
	for _, addr := range in {
		for _, cidr := range filter {
			if cidr.Contains(addr) {
				addrs[i] = addr
				i++
			}
		}
	}

	return addrs[:i]
}
