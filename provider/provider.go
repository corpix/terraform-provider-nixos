package provider

import (
	"context"
	"net"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
)

type Provider struct {
	*ResourceData

	addressFilter   []*CIDR
	addressPriority map[*IPNet]int
}

func (p *Provider) Address(rawAddrs interface{}) (IP, error) {
	var (
		ip    IP
		addrs = FilterIPAddress(p.addressFilter, ToIPAddrs(rawAddrs))
	)
	if len(addrs) == 0 {
		return ip, errors.Errorf("no address from list %q matched with current address filters", addrs)
	}

	ip = SortIPAddress(p.addressPriority, addrs)[0]

	return ip, nil
}

//

func (p *Provider) initAddressFilter() error {
	var (
		err     error
		filters = p.Get(KeyAddressFilter).([]interface{})
	)

	p.addressFilter = make([]*CIDR, len(filters))
	for i, v := range filters {
		p.addressFilter[i], err = ParseCIDR(v.(string))
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) initAddressPriority() error {
	var (
		err        error
		network    *CIDR
		priorities = p.Get(KeyAddressPriority).(map[string]interface{})
	)

	p.addressPriority = make(map[*net.IPNet]int, len(priorities))
	for networkCIDR, weight := range priorities {
		network, err = ParseCIDR(networkCIDR)
		if err != nil {
			return err
		}
		p.addressPriority[network.IPNet] = weight.(int)
	}
	return nil
}

func (p *Provider) init() error {
	initializers := []func() error{
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
	var err error

	providerLevel := p.Get(key).(*schema.Set).List()[0].(map[string]interface{})
	if resource == nil {
		return providerLevel
	}

	resourceLevelList := resource.Get(key).(*schema.Set).List()
	if len(resourceLevelList) > 0 {
		resourceLevel := resourceLevelList[0].(map[string]interface{})
		providerLevelCopy := make(map[string]interface{}, len(providerLevel))

		err = mergo.MergeWithOverwrite(&providerLevelCopy, providerLevel)
		if err != nil {
			panic(err)
		}
		err = mergo.MergeWithOverwrite(&providerLevelCopy, resourceLevel)
		if err != nil {
			panic(err)
		}

		providerLevel = providerLevelCopy
	}

	return providerLevel
}

func (p *Provider) NewNix(resource *schema.ResourceData) *Nix {
	var (
		options     []NixOption
		nixSettings = p.NixSettings(resource)
	)

	if showTrace, ok := nixSettings[KeyNixShowTrace].(bool); ok && showTrace {
		options = append(options, NixOptionShowTrace())
	}
	if num, ok := nixSettings[KeyNixCores].(int); ok && num > 0 {
		options = append(options, NixOptionCores(num))
	}
	if use, ok := nixSettings[KeyNixUseSubstitutes].(bool); ok && use {
		options = append(options, NixOptionUseSubstitutes())
	}

	// TODO: check nix version
	options = append(options, NixOptionExperimentalFeatures(NixFeatureCommand))

	options = append(options, NixOptionSsh(p.NewSsh(resource)))

	return NewNix(options...)
}

func (p *Provider) NixSettings(resource *schema.ResourceData) map[string]interface{} {
	return p.settings(KeyNix, resource)
}

func (p *Provider) NewSsh(resource *schema.ResourceData) *Ssh {
	var (
		options     []SshOption
		sshSettings = p.SshSettings(resource)
	)

	if sshConfig, ok := sshSettings[KeySshConfig].(map[string]interface{}); ok {
		sshConfigStrings := make(map[string]string, len(sshConfig))
		for k, v := range sshConfig {
			sshConfigStrings[k] = v.(string)
		}
		options = append(options, SshOptionConfigMap(sshConfigStrings))
	}

	return NewSsh(options...)
}

func (p *Provider) SshSettings(resource *schema.ResourceData) map[string]interface{} {
	return p.settings(KeySsh, resource)
}

//

func (p *Provider) Build(ctx context.Context, resource *schema.ResourceData) (Derivations, error) {
	nix := p.NewNix(resource)
	defer nix.Close()

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

	command := nix.Build(
		NixBuildCommandOptionFile(buildWrapper),
		NixBuildCommandOptionArgStr("configuration", configurationAbs),
		NixBuildCommandOptionJSON(),
		NixBuildCommandOptionNoLink(),
	)
	defer command.Close()

	select {
	case <-ctx.Done():
		return nil, nil
	default:
	}

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

func (p *Provider) Push(ctx context.Context, resource *schema.ResourceData, drvs Derivations) error {
	nix := p.NewNix(resource)
	defer nix.Close()

	address, err := p.Address(resource.Get(KeyAddress))
	if err != nil {
		return err
	}

	for _, drv := range drvs {
		for _, path := range drv.Outputs {
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			command := nix.Copy(
				NixCopyCommandOptionTo(NixCopyProtocolSSH, address.String()),
				NixCopyCommandOptionPath(path),
			)
			defer command.Close()

			err = command.Execute(nil)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Provider) Switch(ctx context.Context, resource *schema.ResourceData, drvs Derivations) error {
	address, err := p.Address(resource.Get(KeyAddress))
	if err != nil {
		panic(err)
	}

	var (
		nix         = p.NewNix(resource)
		nixSettings = p.NixSettings(resource)
		profilePath = nixSettings[KeyNixProfile].(string)
		outName     = nixSettings[KeyNixOutputName].(string)
		drvPath     = drvs[len(drvs)-1].Outputs[outName]
		ssh         = nix.Ssh.With(SshOptionHost(address.String()))

		nixProfile        = nix.Profile()
		nixProfileInstall = NewRemoteCommand(
			ssh,
			nixProfile.Install(
				NixProfileInstallCommandOptionProfile(profilePath),
				NixProfileInstallCommandOptionDerivation(drvPath),
			),
		)

		activationScript = nixSettings[KeyNixActivationScript].(string)
		activationAction = nixSettings[KeyNixActivationAction].(string)
		activation       = NewRemoteCommand(
			ssh,
			CommandFromString(
				activationScript,
				activationAction,
			),
		)
		skipActivation = false
	)

	switch activationAction {
	case NixActivationActionNone:
		skipActivation = true
	case NixActivationActionSwitch:
	case NixActivationActionBoot:
	case NixActivationActionTest:
	case NixActivationActionDryActivate:
	default:
		return errors.Errorf("unsupported activation action: %q", activationAction)
	}

	err = nixProfileInstall.Execute(nil)
	if err != nil {
		return err
	}

	if !skipActivation {
		err = activation.Execute(nil)
	}

	return err
}

func (p *Provider) Close() error { return nil }

//

func NewProvider(d *ResourceData) (*Provider, error) {
	p := &Provider{ResourceData: d}

	err := p.init()
	if err != nil {
		return nil, err
	}
	return p, nil
}
