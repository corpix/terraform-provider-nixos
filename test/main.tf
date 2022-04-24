terraform {
  required_providers {
    nixos = {
      source = "corpix/nixos"
      version = "0.0.1"
    }
  }
}

provider "nixos" {
  nix {
    cores = 2
  }
  ssh {
    config = {
      port = "22"
    }
  }
}

resource "nixos_instance" "test" {
  address = ["172.17.0.1", "fd00:17::1"]
  configuration = "test.nix"
  nix {
    cores = 1
  }
  ssh {
    config = {
      port = "2222"
    }
  }
}
