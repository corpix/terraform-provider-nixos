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
      user = "root"
      port = "22"
      # this is for totally insecure test server in docker
      # see: make run/sshd
      pubKeyAuthentication = "no"
      passwordAuthentication = "yes"
      strictHostKeyChecking = "no"
    }
  }
}

resource "nixos_instance" "test" {
  address = ["127.0.0.1", "::1"]
  configuration = "test.nix"
  nix {
    cores = 1
    activation_action = "" # skip activation because we run in docker
  }
  ssh {
    config = {
      port = "2222"
    }
  }
}
