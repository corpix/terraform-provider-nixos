package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type schemaResource interface {
	Get(key string) interface{}
}

//

func init() {
	schema.DescriptionKind = schema.StringMarkdown
}

//

const (
	KeyAddressFilter   = "address_filter"
	KeyAddressPriority = "address_priority"
	KeyNixosInstance   = "nixos_instance"
	KeyAddress         = "address"
	KeyConfiguration   = "configuration"
	KeySettings        = "settings"
	KeyRetry           = "retry"
	KeyRetryWait       = "retry_wait"

	//

	KeyNix             = "nix"
	KeyNixMode         = "mode"
	KeyNixBuildWrapper = "build_wrapper"

	KeyNixProfile          = "profile"
	KeyNixOutputName       = "output"
	KeyNixActivationScript = "activation_script"
	KeyNixActivationAction = "activation_action"

	KeyNixShowTrace      = "show_trace"
	KeyNixCores          = "cores"
	KeyNixUseSubstitutes = "use_substitutes"

	//

	KeySsh        = "ssh"
	KeySshHost    = "host"
	KeySshUser    = "user"
	KeySshPort    = "port"
	KeySshConfig  = "config"
	KeySshBastion = "bastion"

	//

	KeySecrets = "secrets"

	KeySecretsFingerprint              = "secrets_fingerprint"
	KeySecretsFingerprintSum           = "sum"
	KeySecretsFingerprintSalt          = "salt"
	KeySecretsFingerprintKdfIterations = "kdf_iterations"

	KeySecretsProvider                   = "provider"
	KeySecretsProviderFilesystem         = "filesystem"
	KeySecretsProviderCommand            = "command"
	KeySecretsProviderCommandName        = "name"
	KeySecretsProviderCommandArguments   = "arguments"
	KeySecretsProviderCommandEnvironment = "environment"
	KeySecretsProviderGopass             = "gopass"
	KeySecretsProviderGopassStore        = "store"

	//

	KeySecret            = "secret"
	KeySecretSource      = "source"
	KeySecretDestination = "destination"
	KeySecretOwner       = "owner"
	KeySecretGroup       = "group"
	KeySecretPermissions = "permissions"

	//

	KeyDerivations       = "derivations"
	KeyDerivationPath    = "path"
	KeyDerivationOutputs = "outputs"
)

var (
	ProviderSchemaSshMap = map[string]*schema.Schema{
		KeySshHost: {
			Description: "SSH remote hostname",
			Type:        schema.TypeString,
			Optional:    true,
		},
		KeySshUser: {
			Description: "SSH remote user name",
			Type:        schema.TypeString,
			Optional:    true,
			Default:     DefaultUser,
		},
		KeySshPort: {
			Description: "SSH remote port",
			Type:        schema.TypeInt,
			Optional:    true,
		},
		KeySshConfig: {
			Description: "SSH configuration map",
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			DefaultFunc: DefaultSshConfig,
		},
	}
	ProviderSchemaSsh = SchemaWithDefaultFuncCtr(DefaultMapFromSchema, &schema.Schema{
		Description: "SSH protocol settings",
		Type:        schema.TypeSet,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: SchemaMapExtend(
				ProviderSchemaSshMap,
				map[string]*schema.Schema{
					KeySshBastion: {
						Description: "SSH configuration for bastion server",
						Type:        schema.TypeSet,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: ProviderSchemaSshMap,
						},
						Optional: true,
					},
				},
			),
		},
		Optional: true,
	})

	ProviderSchemaSecretsProviderFilesystem = SchemaWithDefaultFuncCtr(DefaultMapFromSchema, &schema.Schema{
		Description: "Filesystem secrets provider settings",
		Type:        schema.TypeSet,
		MinItems:    0,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{},
		},
		Optional: true,
	})
	ProviderSchemaSecretsProviderCommand = SchemaWithDefaultFuncCtr(DefaultMapFromSchema, &schema.Schema{
		Description: "Command secrets provider settings",
		Type:        schema.TypeSet,
		MinItems:    0,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				KeySecretsProviderCommandName: {
					Description: "Provider command name (will be looked up in PATH) to use to retrieve secret",
					Type:        schema.TypeString,
					Required:    true,
				},
				KeySecretsProviderCommandArguments: {
					Description: "Provider command arguments to prepend to secret source",
					Type:        schema.TypeList,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Optional:    true,
				},
				KeySecretsProviderCommandEnvironment: {
					Description: "Provider command environment variables",
					Type:        schema.TypeMap,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Optional:    true,
				},
			},
		},
		Optional: true,
	})
	ProviderSchemaSecretsProviderGopass = SchemaWithDefaultFuncCtr(DefaultMapFromSchema, &schema.Schema{
		Description: "GoPass secrets provider settings",
		Type:        schema.TypeSet,
		MinItems:    0,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				KeySecretsProviderGopassStore: {
					Description: "Password store directory location",
					Type:        schema.TypeString,
					Optional:    true,
				},
			},
		},
		Optional: true,
	})
	ProviderSchemaSecretsMap = map[string]*schema.Schema{
		KeySecretsProvider: {
			Description: fmt.Sprintf("Secrets provider to use, available: %v", SecretsProviders),
			Type:        schema.TypeString,
			Optional:    true,
			Default:     string(SecretsProviderNameFilesystem),
		},
		KeySecretsProviderFilesystem: ProviderSchemaSecretsProviderFilesystem,
		KeySecretsProviderCommand:    ProviderSchemaSecretsProviderCommand,
		KeySecretsProviderGopass:     ProviderSchemaSecretsProviderGopass,
	}
	ProviderSchemaSecrets = SchemaWithDefaultFuncCtr(DefaultMapFromSchema, &schema.Schema{
		Description: "Describes secrets settings",
		Type:        schema.TypeSet,
		MinItems:    0,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: ProviderSchemaSecretsMap,
		},
		Optional: true,
	})
	ProviderSchemaSecretsChecksum = &schema.Schema{
		Description: "Secrets salt, kdf iterations, checksum which are used to maintain state",
		Type:        schema.TypeMap,
		Elem:        &schema.Schema{Type: schema.TypeString},
		Computed:    true,
	}
	// FIXME: do we need defaults here?
	ProviderSchemaSecret = SchemaWithDefaultFuncCtr(DefaultMapFromSchema, &schema.Schema{
		Description: "Describes secret which should be transfered to host",
		Type:        schema.TypeSet,
		MinItems:    0,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				KeySecretSource: {
					Description: "Secret file on the host which should be transfered to destination",
					Type:        schema.TypeString,
					Required:    true,
				},
				KeySecretDestination: {
					Description: "Secret file destination on the target host",
					Type:        schema.TypeString,
					Required:    true,
				},
				KeySecretOwner: {
					Description: "Secret file owner username",
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "root",
				},
				KeySecretGroup: {
					Description: "Secret file owner groupname",
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "root",
				},
				KeySecretPermissions: {
					Description: "Secret file destination permissions (in octal)",
					Type:        schema.TypeInt,
					Optional:    true,
					Default:     600,
				},
			},
		},
		Optional: true,
	})

	ProviderSchemaDerivationsComputedMap = map[string]*schema.Schema{
		// TODO: support custom Nix store paths
		// probably this change will be breaking because
		// to gain more wide compatibility we need to trim
		// current Nix store prefix
		// or maybe we could do it better?
		// (goal: Alice has /nix/store, Bob /storage/nix -> minimize state changes between Alice & Bob)
		KeyDerivationPath: {
			Description: "Path to the derivation in Nix store",
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
		},
		KeyDerivationOutputs: {
			Description: "Derivation outputs paths in the Nix store",
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Computed:    true,
		},
	}

	ProviderSchemaNix = SchemaWithDefaultFuncCtr(DefaultMapFromSchema, &schema.Schema{
		Description: "Nix package manager configuration options",
		Type:        schema.TypeSet,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				KeyNixMode: {
					Description: "Nix mode (0 - compat, 1 - default)",
					Type:        schema.TypeInt,
					Optional:    true,
					Default:     int(NixModeCompat),
				},
				KeyNixBuildWrapper: {
					Description: "Path to the configuration wrapper in Nix language (function which returns drv_path & out_path)",
					Type:        schema.TypeString,
					Optional:    true,
				},
				KeyNixProfile: {
					Description: "Path to the current system profile",
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "/nix/var/nix/profiles/system",
				},
				KeyNixOutputName: {
					Description: "System derivation output name",
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "out",
				},
				KeyNixActivationScript: {
					Description: "Path to the system profile activation script",
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "/nix/var/nix/profiles/system/bin/switch-to-configuration",
				},
				KeyNixActivationAction: {
					Description: "Activation script action, one of: switch|boot|test|dry-activate",
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "switch",
				},
				KeyNixShowTrace: {
					Description: "Show Nix package manager trace on error",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     true,
				},
				KeyNixCores: {
					Description: "Number of CPU cores  which Nix should use to perform builds",
					Type:        schema.TypeInt,
					Optional:    true,
				},
				KeyNixUseSubstitutes: {
					Description: "Wether or not should Nix use substitutes",
					Type:        schema.TypeBool,
					Optional:    true,
					Default:     true,
				},
			},
		},
		Optional: true,
	})

	ProviderSchemaMap = map[string]*schema.Schema{
		KeyRetry: {
			Description: "Amount of retries for retryable operations",
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     5,
		},
		KeyRetryWait: {
			Description: "Amount of seconds to wait between retries",
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     5,
		},

		KeyAddressFilter: {
			Description: "List of network cidr's to filter addresses used to connect to nixos_instance resources",
			Type:        schema.TypeList,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			DefaultFunc: DefaultAddressFilter,
		},
		KeyAddressPriority: {
			Description: "Map of network cidr's with associated weight which will affect address ordering for nixos_isntance resource",
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeInt},
			Optional:    true,
			DefaultFunc: DefaultAddressPriority,
		},

		KeyNix:     ProviderSchemaNix,
		KeySsh:     ProviderSchemaSsh,
		KeySecrets: ProviderSchemaSecrets,
		KeySecret:  ProviderSchemaSecret,
	}

	//

	ProviderResourceMap = map[string]*schema.Resource{
		KeyNixosInstance: {
			Description: "NixOS instance",

			CustomizeDiff: instance.Diff,
			CreateContext: instance.Create,
			ReadContext:   instance.Read,
			UpdateContext: instance.Update,
			DeleteContext: instance.Delete,

			Schema: map[string]*schema.Schema{
				KeyAddress: {
					Description: "List of server addresses",
					Type:        schema.TypeList,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Required:    true,
				},
				KeyConfiguration: {
					Description: "Path to Nix derivation",
					Type:        schema.TypeString,
					Required:    true,
				},
				KeySettings: {
					Description: "Optional settings (encoded with HCL function jsonencode()) to pass into Nix configuration derivation as attribute set (any configuration key could be specified)",
					Type:        schema.TypeString,
					Optional:    true,
					Default:     "{}",
				},

				KeyNix:     ProviderSchemaNix,
				KeySsh:     ProviderSchemaSsh,
				KeySecrets: ProviderSchemaSecrets,
				KeySecret:  ProviderSchemaSecret,

				KeyDerivations: {
					Description: "List of derivations which is built during apply",
					Type:        schema.TypeList,
					Elem:        &schema.Resource{Schema: ProviderSchemaDerivationsComputedMap},
					Optional:    true,
					Computed:    true,
				},
			},
		},
	}

	ProviderSchema = schema.Provider{
		Schema:       ProviderSchemaMap,
		ResourcesMap: ProviderResourceMap,
	}
)

func HashAny(v interface{}) int {
	buf, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	// NOTE: copy pasted from "github.com/hashicorp/terraform-plugin-sdk/v2/internal/helper/hashcode"
	// because internal sucks when you need to fix a broken design
	h := int(crc32.ChecksumIEEE(buf))
	if h >= 0 {
		return h
	}
	if -h >= 0 {
		return -h
	}
	// h == MinInt
	return 0
}

func New() *schema.Provider {
	p := ProviderSchema

	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		p, err := NewProvider(&ResourceData{
			ResourceBox: d,
			Schema:      p.Schema,
		})
		if err != nil {
			return nil, diag.Diagnostics{{
				Severity: diag.Error,
				Summary:  err.Error(),
			}}
		}
		go func() {
			// FIXME: dear hashicorp dumbtards
			// how the fuck should people deallocate resources (tmp files, etc)
			// which was allocated by the provider?
			// (how I should call close()???)
			//
			// looks like this is working (for everything but not for panics!)
			<-ctx.Done()
			p.Close()
		}()

		return p, nil
	}
	return &p
}
