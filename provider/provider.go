package provider

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/imdario/mergo"
	"github.com/pkg/errors"
)

//

type Provider struct {
	ResourceBox

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

func (p *Provider) resolveSettings(r schemaResource, path ...string) interface{} {
	var (
		value interface{}
		ok    bool
		fold  = func(key string, value interface{}) interface{} {
			switch v := value.(type) {
			case nil:
				value = r.Get(key)
			case *schema.Set:
				for _, vv := range v.List() {
					switch vvv := vv.(type) {
					case map[string]interface{}:
						value = vvv[key]
						if ok {
							break
						}
					default:
						goto fail
					}
				}
			default:
				goto fail
			}
			return value
		fail:
			panic(fmt.Sprintf(
				"got unhandled type %T at %q while walking path %q on resource %#v",
				value, key, strings.Join(path, "."), r,
			))
		}
	)

	value = fold(path[0], nil)
	for _, key := range path[1:] {
		value = fold(key, value)
	}

	//

	switch v := value.(type) {
	case *schema.Set:
		value = v.List()
	}

	return value
}

// retrieve hashmap from set with maxItems == 1
func (p *Provider) settings(resource ResourceBox, path ...string) map[string]interface{} {
	providerLevel := map[string]interface{}{}
	settings, _ := p.resolveSettings(p, path...).([]interface{})
	if len(settings) == 0 {
		return providerLevel
	}
	providerLevel = settings[0].(map[string]interface{})
	if resource == nil {
		return providerLevel
	}

	resourceLevelSettings := p.resolveSettings(resource, path...).([]interface{})
	if len(resourceLevelSettings) > 0 {
		resourceLevel := resourceLevelSettings[0].(map[string]interface{})
		providerLevelCopy := make(map[string]interface{}, len(providerLevel))

		err := mergo.MergeWithOverwrite(&providerLevelCopy, providerLevel)
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

// retrieve list of hashmap's from set with maxItems == infinity
func (p *Provider) settingsSet(resource ResourceBox, path ...string) []map[string]interface{} {
	settings, _ := p.resolveSettings(p, path...).([]interface{})
	providerLevel := make([]map[string]interface{}, len(settings))
	if len(settings) == 0 {
		return providerLevel
	}

	for n, setting := range settings {
		providerLevel[n] = setting.(map[string]interface{})
	}
	if resource == nil {
		return providerLevel
	}

	resourceLevelSettings := p.resolveSettings(resource, path...).([]interface{})
	if len(resourceLevelSettings) > 0 {
		resourceLevel := make([]map[string]interface{}, len(resourceLevelSettings))
		for n, resourceLevelSetting := range resourceLevelSettings {
			resourceLevel[n] = resourceLevelSetting.(map[string]interface{})
		}
		providerLevel = append(providerLevel, resourceLevel...)
	}

	return providerLevel
}

func (p *Provider) NixSettings(resource ResourceBox) map[string]interface{} {
	return p.settings(resource, KeyNix)
}

func (p *Provider) SshSettings(resource ResourceBox) map[string]interface{} {
	return p.settings(resource, KeySsh)
}

func (p *Provider) SshBastionSettings(resource ResourceBox) map[string]interface{} {
	return p.settings(resource, KeySsh, KeySshBastion)
}

func (p *Provider) SecretsSet(resource ResourceBox) []map[string]interface{} {
	return p.settingsSet(resource, KeySecret)
}

//

func (p *Provider) NewNix(resource ResourceBox) *Nix {
	var (
		options  []NixOption
		settings = p.NixSettings(resource)
	)

	// NOTE: should be first option in set
	// because other options may rely on mode
	if mode, ok := settings[KeyNixMode].(int); ok {
		options = append(options, NixOptionMode(NixMode(mode)))
	}

	if showTrace, ok := settings[KeyNixShowTrace].(bool); ok && showTrace {
		options = append(options, NixOptionShowTrace())
	}
	if num, ok := settings[KeyNixCores].(int); ok && num > 0 {
		options = append(options, NixOptionCores(num))
	}
	if use, ok := settings[KeyNixUseSubstitutes].(bool); ok && use {
		options = append(options, NixOptionUseSubstitutes())
	}

	options = append(
		options,
		NixOptionSsh(p.NewSsh(resource)),
	)

	return NewNix(options...)
}

func (p *Provider) NewSsh(resource ResourceBox) *Ssh {
	var (
		options []SshOption

		settings        = p.SshSettings(resource)
		configMap       = p.SshConfigMap(settings)
		bastionSettings = p.SshBastionSettings(resource)
	)

	if len(bastionSettings) > 0 {
		bastionHost, _ := bastionSettings[KeySshHost].(string)
		if bastionHost != "" {
			bastionConfigMap := p.SshConfigMap(bastionSettings)
			if bastionConfigMap.Len() > 0 {
				bastionConfigOption := SshOptionConfigMap(bastionConfigMap.Pairs())
				bastion := NewSsh(
					bastionConfigOption,
					SshOptionNonInteractive(),
					SshOptionIORedirection("%h", "%p"),
					SshOptionHost(bastionHost),
				)
				command, arguments, _ := bastion.Command()
				configMap.Set(
					SshConfigKeyProxyCommand,
					strings.Join(append([]string{command}, arguments...), " "),
				)
				options = append(
					options,
					bastionConfigOption,
				)
			}
		}
	}

	if configMap.Len() > 0 {
		options = append(
			options,
			SshOptionConfigMap(configMap.Pairs()),
		)
	}

	return NewSsh(options...)
}

func (p *Provider) SshConfigMap(settings map[string]interface{}) *SshConfigMap {
	sshConfigMap := NewSshConfigMap()
	if sshHost, ok := settings[KeySshHost].(string); ok && len(sshHost) > 0 {
		sshConfigMap.Set(SshConfigKeyHost, sshHost)
	}
	if sshUser, ok := settings[KeySshUser].(string); ok && len(sshUser) > 0 {
		sshConfigMap.Set(SshConfigKeyUser, sshUser)
	}
	if sshPort, ok := settings[KeySshPort].(int); ok && sshPort > 0 {
		sshConfigMap.Set(SshConfigKeyPort, strconv.Itoa(sshPort))
	}
	if sshConfig, ok := settings[KeySshConfig].(map[string]interface{}); ok {
		for k, v := range sshConfig {
			sshConfigMap.Set(k, v.(string))
		}
	}
	return sshConfigMap
}

func (p *Provider) NewSecrets(resource ResourceBox) *Secrets {
	schemaSecrets := p.settingsSet(resource, KeySecret)
	definedSecrets := make([]*Secret, len(schemaSecrets))

	n := 0
	for _, schemaSecret := range schemaSecrets {
		if schemaSecret[KeySecretSource] == nil && schemaSecret[KeySecretDestination] == nil {
			// FIXME: in case no secrets providen set will contain
			// single item with defaults values
			// I don't know why, but looks like terraform SDK
			// automagically set DefaultFunc for sets and
			// this interfere with my wrapper in resource.go
			// I hate you, terraform, poorly engineered piece of crap!
			continue
		}

		definedSecrets[n] = &Secret{
			Source:      schemaSecret[KeySecretSource].(string),
			Destination: schemaSecret[KeySecretDestination].(string),
			Owner:       schemaSecret[KeySecretOwner].(string),
			Group:       schemaSecret[KeySecretGroup].(string),
			Permissions: schemaSecret[KeySecretPermissions].(int),
		}
		n++
	}
	return NewSecrets(definedSecrets[:n])
}

//

func (p *Provider) Build(ctx context.Context, resource ResourceBox) (Derivations, error) {
	nix := p.NewNix(resource)
	defer nix.Close()

	nixSettings := p.NixSettings(resource)
	buildWrapperPath, _ := nixSettings[KeyNixBuildWrapper].(string)
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
	configurationSettings := resource.Get(KeySettings).(string)

	command := nix.Build(
		NixBuildCommandOptionFile(buildWrapper),
		NixBuildCommandOptionArgStr("configuration", configurationAbs),
		NixBuildCommandOptionArgStr("settings", configurationSettings),
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

func (p *Provider) Push(ctx context.Context, resource ResourceBox, drvs Derivations) error {
	nix := p.NewNix(resource)
	defer nix.Close()

	address, err := p.Address(resource.Get(KeyAddress))
	if err != nil {
		return err
	}

	secretsCopy, err := p.NewSecrets(resource).Copy(nix.Ssh.With(SshOptionHost(address.String())))
	if err != nil {
		return err
	}
	defer secretsCopy.Close()

	err = secretsCopy.Execute(nil)
	if err != nil {
		return err
	}

	//

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

func (p *Provider) Switch(ctx context.Context, resource ResourceBox, drvs Derivations) error {
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

func NewProvider(d ResourceBox) (*Provider, error) {
	p := &Provider{ResourceBox: d}

	err := p.init()
	if err != nil {
		return nil, err
	}
	return p, nil
}
