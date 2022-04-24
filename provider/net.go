package provider

import (
	"net"
)

type cidr struct {
	IP net.IP
	*net.IPNet
}
