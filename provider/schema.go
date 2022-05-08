package provider

import (
	"context"
	"encoding/json"
	"hash/crc32"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

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

	//

	KeyNix             = "nix"
	KeyNixBuildWrapper = "build_wrapper"

	KeyNixProfile          = "profile"
	KeyNixOutputName       = "output"
	KeyNixActivationScript = "activation_script"
	KeyNixActivationAction = "activation_action"

	KeyNixShowTrace      = "show_trace"
	KeyNixCores          = "cores"
	KeyNixUseSubstitutes = "use_substitutes"

	//

	KeySsh       = "ssh"
	KeySshConfig = "config"

	KeyDerivations       = "derivations"
	KeyDerivationPath    = "path"
	KeyDerivationOutputs = "outputs"
)

var (
	ProviderSchemaSsh = SchemaWithDefaultFuncCtr(DefaultMapFromSchema, &schema.Schema{
		Description: "SSH protocol settings",
		Type:        schema.TypeSet,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				KeySshConfig: {
					Description: "OpenSSH configuration map",
					Type:        schema.TypeMap,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Optional:    true,
					DefaultFunc: DefaultSshConfig,
				},
			},
		},
		Optional: true,
	})

	ProviderSchemaNix = SchemaWithDefaultFuncCtr(DefaultMapFromSchema, &schema.Schema{
		Description: "Nix package manager configuration options",
		Type:        schema.TypeSet,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
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
		KeyNix: ProviderSchemaNix,
		KeySsh: ProviderSchemaSsh,
	}

	//

	ProviderResourceMap = map[string]*schema.Resource{
		KeyNixosInstance: {
			Description: "NixOS instance",

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

				KeyNix: ProviderSchemaNix,
				KeySsh: ProviderSchemaSsh,

				KeyDerivations: {
					Description: "List of derivations which is built during apply",
					Type:        schema.TypeList,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
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
						},
					},
					Optional: true,
					Computed: true,
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
			ResourceData: d,
			Schema:       p.Schema,
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
