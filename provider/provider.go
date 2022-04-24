package provider

import (
	"io"
	"net"
	"path/filepath"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

type Provider struct {
	*ResourceData
	Nix *Nix
	Ssh *Ssh

	addressFilter   []*cidr
	addressPriority map[*net.IPNet]int
}

func (p *Provider) ParseCIDR(addr string) (*cidr, error) {
	ip, ipnet, err := net.ParseCIDR(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse cidr %q", addr)
	}
	return &cidr{IP: ip, IPNet: ipnet}, nil
}

func (p *Provider) ToAddrs(inAddrs interface{}) []net.IP {
	var (
		inAddrSlice = inAddrs.([]interface{})
		addrs       = make([]net.IP, len(inAddrSlice))
	)
	for i, addr := range inAddrSlice {
		addrs[i] = net.ParseIP(addr.(string))
	}
	return addrs
}

func (p *Provider) FilterAddress(inAddrs []net.IP) []net.IP {
	var (
		addrs = make([]net.IP, len(inAddrs))
		i     int
	)
	if len(p.addressFilter) == 0 {
		copy(addrs, inAddrs)
		return addrs
	}
	for _, addr := range inAddrs {
		for _, cidr := range p.addressFilter {
			if cidr.Contains(addr) {
				addrs[i] = addr
				i++
			}
		}
	}

	return addrs[:i]
}

func (p *Provider) SortAddress(inAddrs []net.IP) []net.IP {
	addrs := make([]net.IP, len(inAddrs))
	copy(addrs, inAddrs)
	if len(p.addressPriority) == 0 {
		return addrs
	}
	sort.SliceStable(addrs, func(i, j int) bool {
		iaddr, jaddr := inAddrs[i], inAddrs[j]
		iweight, jweight := 0, 0
		for network, weight := range p.addressPriority {
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

func (p *Provider) Address(rawAddrs interface{}) (net.IP, error) {
	var (
		ip    net.IP
		addrs = p.FilterAddress(p.ToAddrs(rawAddrs))
	)
	if len(addrs) == 0 {
		return ip, errors.Errorf("no address from list %q matched with current address filters", addrs)
	}

	ip = p.SortAddress(addrs)[0]

	return ip, nil
}

//

func (p *Provider) initSsh() error {
	var options []SshOption
	if config, ok := p.ResourceData.Get(KeySsh).(*schema.Set); ok && config.Len() > 0 {
		if config.Len() > 1 {
			return errors.Errorf(
				"expecting single configuration for Ssh, but got a set of %d items",
				config.Len(),
			)
		}
		ssh := config.List()[0].(map[string]interface{})

		if sshConfig, ok := ssh[KeySshConfig].(map[string]interface{}); ok {
			sshConfigStrings := make(map[string]string, len(sshConfig))
			for k, v := range sshConfig {
				sshConfigStrings[k] = v.(string)
			}
			options = append(options, SshOptionConfigMap(sshConfigStrings))
		}
	}
	p.Ssh = NewSsh(options...)
	return nil
}

func (p *Provider) initNix() error {
	var options []NixOption
	if config, ok := p.ResourceData.Get(KeyNix).(*schema.Set); ok && config.Len() > 0 {
		if config.Len() > 1 {
			return errors.Errorf(
				"expecting single configuration for Nix, but got a set of %d items",
				config.Len(),
			)
		}
		nix := config.List()[0].(map[string]interface{})

		if showTrace, ok := nix[KeyNixShowTrace].(bool); ok && showTrace {
			options = append(options, NixOptionShowTrace())
		}
		if num, ok := nix[KeyNixCores].(int); ok && num > 0 {
			options = append(options, NixOptionCores(num))
		}
		if use, ok := nix[KeyNixUseSubstitutes].(bool); ok && use {
			options = append(options, NixOptionUseSubstitutes())
		}
	}

	// NOTE: ssh should be initialized at this moment, see init()
	_, sshOpts, _ := p.Ssh.Command()
	if len(sshOpts) > 0 {
		options = append(options, NixOptionSshOpts(sshOpts...))
	}

	p.Nix = NewNix(options...)
	return nil
}

func (p *Provider) initAddressFilter() error {
	var (
		err     error
		filters = p.Get(KeyAddressFilter).([]interface{})
	)

	p.addressFilter = make([]*cidr, len(filters))
	for i, v := range filters {
		p.addressFilter[i], err = p.ParseCIDR(v.(string))
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) initAddressPriority() error {
	var (
		err        error
		network    *cidr
		priorities = p.Get(KeyAddressPriority).(map[string]interface{})
	)

	p.addressPriority = make(map[*net.IPNet]int, len(priorities))
	for networkCIDR, weight := range priorities {
		network, err = p.ParseCIDR(networkCIDR)
		if err != nil {
			return err
		}
		p.addressPriority[network.IPNet] = weight.(int)
	}
	return nil
}

func (p *Provider) init() error {
	initializers := []func() error{
		p.initSsh,
		p.initNix,
		p.initAddressFilter,
		p.initAddressPriority,
	}
	for _, initializer := range initializers {
		err := initializer()
		if err != nil {
			return err
		}
	}
	return nil
}

//

func (p *Provider) settings(key string, resource *schema.ResourceData) map[string]interface{} {
	providerLevel := p.Get(key).(*schema.Set).List()[0].(map[string]interface{})
	resourceLevel := resource.Get(key).(*schema.Set).List()
	if len(resourceLevel) > 0 {
		providerLevelCopy := make(map[string]interface{}, len(providerLevel))
		for k, v := range providerLevel {
			providerLevelCopy[k] = v
		}
		for k, v := range resourceLevel[0].(map[string]interface{}) {
			providerLevelCopy[k] = v
		}
		providerLevel = providerLevelCopy
	}
	return providerLevel
}

func (p *Provider) NixSettings(resource *schema.ResourceData) map[string]interface{} {
	return p.settings(KeyNix, resource)
}

func (p *Provider) SshSettings(resource *schema.ResourceData) map[string]interface{} {
	return p.settings(KeySsh, resource)
}

//

func (p *Provider) Build(resource *schema.ResourceData) (Derivations, error) {
	buildWrapperPath, _ := p.NixSettings(resource)[KeyNixBuildWrapper].(string)
	buildWrapper, err := NewNixWrapperFile(buildWrapperPath)
	if err != nil {
		return nil, err
	}
	defer buildWrapper.Close()

	configuration := resource.Get(KeyConfiguration).(string)
	configurationAbs, err := filepath.Abs(configuration)
	if err != nil {
		return nil, errors.Wrapf(
			err, "failed to determine absolute path for configuration %q",
			configuration,
		)
	}
	command := p.Nix.Build(
		NixBuildCommandOptionFile(buildWrapper),
		NixBuildCommandOptionArgStr("configuration", configurationAbs),
		NixBuildCommandOptionJSON(),
		NixBuildCommandOptionNoLink(),
	)

	derivations := Derivations{}
	err = command.Execute(&derivations)
	if err != nil {
		return nil, err
	}
	if len(derivations) == 0 {
		return nil, errors.Errorf(
			"no derivations was build for %q configuration",
			configuration,
		)
	}
	if len(derivations) > 1 {
		return nil, errors.Errorf(
			"multiple derivations was build for %q configuration (expecting single derivation)",
			configuration,
		)
	}

	return derivations, nil
}

func (p *Provider) Push(resource *schema.ResourceData, drvs Derivations) error {
	//p.SshSettings(resource)[KeySshConfig]
	return nil
}

func (p *Provider) Close() error {
	var (
		closers = []io.Closer{
			p.Nix,
			p.Ssh,
		}
		err error
	)
	for _, closer := range closers {
		err = closer.Close()
	}
	return err
}

//

func NewProvider(d *ResourceData) (*Provider, error) {
	p := &Provider{ResourceData: d}

	err := p.init()
	if err != nil {
		return nil, err
	}
	return p, nil
}
