package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var providerFactories = map[string]func() (*schema.Provider, error){
	"nixos": func() (*schema.Provider, error) {
		return New(), nil
	},
}

const nixosMinimalConfig = `
provider "nixos" {
  nix {
    activation_action = "" # skip activation because we are running in docker
  }
  ssh {
    port = 2222
    config = {
      userKnownHostsFile = "/dev/null"
      strictHostKeyChecking = "no"
      pubKeyAuthentication = "no"
      passwordAuthentication = "yes"
    }
  }
}

resource "nixos_instance" "test" {
  address = ["127.0.0.1", "::1"]
  configuration = "../test/test.nix"
}
`
const nixosSampleConfig = `
provider "nixos" {
  retry = 0
  nix {
    show_trace = true
    activation_action = "" # skip activation because we are running in docker
  }
  ssh {
    port = 666
    config = {
      userKnownHostsFile = "/dev/null"
      strictHostKeyChecking = "no"
      pubKeyAuthentication = "no"
      passwordAuthentication = "yes"
    }
    bastion {
      host = "127.0.0.1"
      port = 2222
      config = {
        userKnownHostsFile = "/dev/null"
        strictHostKeyChecking = "no"
        pubKeyAuthentication = "no"
        passwordAuthentication = "yes"
      }
    }
  }
  secrets {
    provider = "command"
    command {
      name = "echo"
      arguments = ["here is your node key"]
    }
  }
  secret {
    source = "node-key"
    destination = "/root/secrets/node-key"
  }
}

resource "nixos_instance" "test" {
  address = ["127.0.0.1", "::1"]
  configuration = "../test/test.nix"
  ssh {
    port = 2222
  }
}
`

//

func CheckEqual(t *testing.T, name, key string, value interface{}) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		got := state.RootModule().Resources[name].Primary.Attributes[key]
		if !assert.Equal(t, value, got) {
			return errors.Errorf("failed at name %q key %q: expected value %#v, got %#v", name, key, value, got)
		}
		return nil
	}
}

//

func TestProvider(t *testing.T) {
	if err := New().InternalValidate(); err != nil {
		t.Fatal(err)
	}
}

func TestResourceNixosInstance(t *testing.T) {
	resource.UnitTest(
		t,
		resource.TestCase{
			PreCheck:          func() {},
			ProviderFactories: providerFactories,
			Steps: []resource.TestStep{
				{
					Config: nixosMinimalConfig,
					Check: resource.ComposeTestCheckFunc(
						CheckEqual(t, "nixos_instance.test", "address.0", "127.0.0.1"),
						CheckEqual(t, "nixos_instance.test", "address.1", "::1"),
						CheckEqual(t, "nixos_instance.test", "address.2", ""),
						CheckEqual(t, "nixos_instance.test", "configuration", "../test/test.nix"),
					),
				},
				{
					Config: nixosSampleConfig,
					Check: resource.ComposeTestCheckFunc(
						CheckEqual(t, "nixos_instance.test", "address.0", "127.0.0.1"),
						CheckEqual(t, "nixos_instance.test", "address.1", "::1"),
						CheckEqual(t, "nixos_instance.test", "address.2", ""),
						CheckEqual(t, "nixos_instance.test", "configuration", "../test/test.nix"),
					),
				},
			},
		},
	)
}
