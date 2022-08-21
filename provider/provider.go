package provider

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/pkg/errors"
)

//

type (
	Provider struct {
		ResourceBox

		addressFilter   []*CIDR
		addressPriority map[*IPNet]int
	}
)

func (p *Provider) Address(rawAddrs interface{}) (IP, error) {
	var ip IP
	if rawAddrs == nil {
		return ip, errors.Errorf("rawAddrs should not be nil")
	}

	var addrs = FilterIPAddress(p.addressFilter, ToIPAddrs(rawAddrs))
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
	)

	fold := func(key string, value interface{}) interface{} {
		switch valueTyped := value.(type) {
		case nil:
			value = r.Get(key)
		case *schema.Set:
			for _, item := range valueTyped.List() {
				switch itemTyped := item.(type) {
				case map[string]interface{}:
					value = itemTyped[key]
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

	for _, key := range path {
		value = fold(key, value)
		if value == nil {
			return nil
		}
	}

	//

	switch v := value.(type) {
	case *schema.Set:
		value = v.List()
	}

	return value
}

func (p *Provider) merge(resources ...map[string]interface{}) map[string]interface{} {
	r := map[string]interface{}{}

	// FIXME: maybe we could improve this?
	// Every time you try to merge two schema parts
	// you need to figure out:
	//   - what fields have default values
	//   - what field have been provided by user
	// But this garbage SDK provides no way to know about this,
	for _, resource := range resources {
		for key, value := range resource {
			rvalue := reflect.ValueOf(value)

			switch rvalue.Kind() {
			case reflect.Map:
				current, ok := r[key]
				if ok {
					r[key] = p.merge(
						current.(map[string]interface{}),
						value.(map[string]interface{}),
					)
				} else {
					r[key] = value
				}
			default:
				r[key] = value
			}
		}
	}

	return r
}

// retrieve hashmap from set with maxItems == 1
func (p *Provider) settings(resource ResourceBox, path ...string) map[string]interface{} {
	providerLevel := map[string]interface{}{}
	settings, _ := p.resolveSettings(p, path...).([]interface{})
	if len(settings) != 0 {
		providerLevel = settings[0].(map[string]interface{})
	}
	if resource == nil {
		return providerLevel
	}
	resourceLevelSettings, _ := p.resolveSettings(resource, path...).([]interface{})
	if len(resourceLevelSettings) > 0 {
		resourceLevel := resourceLevelSettings[0].(map[string]interface{})
		providerLevel = p.merge(providerLevel, resourceLevel)
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

	resourceLevelSettings, _ := p.resolveSettings(resource, path...).([]interface{})
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

func (p *Provider) BastionSettings(resource ResourceBox) map[string]interface{} {
	return p.settings(resource, KeyBastion)
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

func (p *Provider) SecretsSettings(resource ResourceBox) map[string]interface{} {
	return p.settings(resource, KeySecrets)
}

func (p *Provider) SecretsSet(resource ResourceBox) []map[string]interface{} {
	return p.settingsSet(resource, KeySecret)
}

//

func (p *Provider) NewNix(ctx context.Context, resource ResourceBox) *Nix {
	settings := p.NixSettings(resource)
	options := []NixOption{
		NixOptionWithCommandOptions(CommandOptionTflogTee(ctx)),
	}

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
		bastionSettings = p.BastionSettings(resource)
	)

	bastionHost, _ := bastionSettings[KeySshHost].(string)
	if bastionHost != "" {
		bastionConfigMap := p.SshConfigMap(bastionSettings)
		// NOTE: base ssh configuration (ssh {}) extended with bastion ssh configuration (ssh { bastion {} })
		extendedBastionConfigMap := configMap.Copy()
		extendedBastionConfigMap.Extend(bastionConfigMap)

		bastion := NewSsh(
			SshOptionConfigMap(extendedBastionConfigMap),
			SshOptionNonInteractive(),
			SshOptionIORedirection("%h", "%p"),
			SshOptionHost(bastionHost),
		)
		command, arguments, _ := bastion.Command()
		configMap.Set(
			SshConfigKeyProxyCommand,
			strings.Join(append([]string{command}, arguments...), " "),
		)
	}

	if configMap.Len() > 0 {
		options = append(
			options,
			SshOptionConfigMap(configMap),
		)
	}

	return NewSsh(options...)
}

func (p *Provider) NewSecrets(resource ResourceBox) (*Secrets, error) {
	schemaSecrets := p.SecretsSettings(resource)
	providerName := schemaSecrets[KeySecretsProvider].(string)

	var provider SecretsProvider
	switch strings.ToLower(providerName) {
	case string(SecretsProviderNameFilesystem):
		// NOTE: this does nothing now, but we may have filesystem options in future
		_ = p.settings(resource, KeySecrets, KeySecretsProviderFilesystem)
		provider = NewSecretsProviderFilesystem()
	case string(SecretsProviderNameCommand):
		settings := p.settings(resource, KeySecrets, KeySecretsProviderCommand)

		name := settings[KeySecretsProviderCommandName].(string)
		argumentsRaw := settings[KeySecretsProviderCommandArguments].([]interface{})
		arguments := make([]string, len(argumentsRaw))
		for n, argument := range argumentsRaw {
			arguments[n] = argument.(string)
		}
		environmentRaw := settings[KeySecretsProviderCommandEnvironment].(map[string]interface{})
		environment := make(map[string]string, len(environmentRaw))
		for k, value := range environmentRaw {
			environment[k] = value.(string)
		}

		provider = NewSecretsProviderCommand(
			name,
			arguments,
			environment,
		)
	case string(SecretsProviderNameGopass):
		settings := p.settings(resource, KeySecrets, KeySecretsProviderGopass)
		store, _ := settings[KeySecretsProviderGopassStore].(string)
		provider = NewSecretsProviderGopass(store)
	default:
		return nil, errors.Errorf(
			"unsupported secrets provider %q, supported providers are: %v",
			providerName, SecretsProviders,
		)
	}

	//

	schemaSecretsSet := p.SecretsSet(resource)
	definedSecrets := make([]*SecretDescription, len(schemaSecretsSet))

	n := 0
	for _, schemaSecret := range schemaSecretsSet {
		if schemaSecret[KeySecretSource] == nil && schemaSecret[KeySecretDestination] == nil {
			// FIXME: in case no secrets providen set will contain
			// single item with defaults values
			// probably this is because of my wrapper around schema sets
			continue
		}

		definedSecrets[n] = &SecretDescription{
			Source:      schemaSecret[KeySecretSource].(string),
			Destination: schemaSecret[KeySecretDestination].(string),
			Owner:       schemaSecret[KeySecretOwner].(string),
			Group:       schemaSecret[KeySecretGroup].(string),
			Permissions: schemaSecret[KeySecretPermissions].(int),
		}
		n++
	}
	return NewSecrets(provider, definedSecrets[:n]), nil
}

//

func (p *Provider) Build(ctx context.Context, resource ResourceBox) (Derivations, error) {
	nix := p.NewNix(ctx, resource)
	defer nix.Close()

	system := resource.Get(KeySystem).(string)

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
		NixBuildCommandOptionArgStr("system", system),
		NixBuildCommandOptionArgStr("settings", configurationSettings),
		NixBuildCommandOptionArgStr("configuration", configurationAbs),
		NixBuildCommandOptionJSON(),
		NixBuildCommandOptionNoLink(),
		NixBuildCommandOptionMemoize(configurationAbs),
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

func (p *Provider) CopySecrets(ctx context.Context, resource ResourceBox, secrets *Secrets) error {
	address, err := p.Address(resource.Get(KeyAddress))
	if err != nil {
		return err
	}
	ssh := p.NewSsh(resource).With(SshOptionHost(address.String()))
	defer ssh.Close()

	secretsCopy, err := secrets.Copy(ssh)
	if err != nil {
		return err
	}
	defer secretsCopy.Close()

	err = secretsCopy.Execute(nil)
	if err != nil {
		return err
	}

	return nil
}

func (p *Provider) Push(ctx context.Context, resource ResourceBox, drvs Derivations) error {
	nix := p.NewNix(ctx, resource)
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

func (p *Provider) Switch(ctx context.Context, resource ResourceBox, drvs Derivations) error {
	address, err := p.Address(resource.Get(KeyAddress))
	if err != nil {
		return err
	}

	var (
		nix         = p.NewNix(ctx, resource)
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
