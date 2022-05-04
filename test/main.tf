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
  address = ["127.0.0.1", "::1"]
  configuration = "test.nix"
  nix {
    cores = 1
  }
  ssh {
    config = {
      port = "2222"
      strictHostKeyChecking = "no"
    }
  }
}
