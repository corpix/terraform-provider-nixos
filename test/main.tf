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
    port = 666
    config = {
      userKnownHostsFile = "/dev/null"
      strictHostKeyChecking = "no"
      pubKeyAuthentication = "no"
      passwordAuthentication = "yes"
    }
  }
  bastion {
    host = "127.0.0.1"
    port = 777
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
    port = 2222
    config = {
      userKnownHostsFile = "/dev/null"
      strictHostKeyChecking = "no"
      pubKeyAuthentication = "no"
      passwordAuthentication = "yes"
    }
  }
  bastion {
    host = "127.0.0.1"
    port = 2222
  }

  secret {
    source = "./secrets/key1"
    destination = "/root/secrets/key1"
  }
  secret {
    source = "./secrets/subdir/key2"
    destination = "/root/secrets/subdir/key2"
    owner = "nobody"
    group = "nogroup"
    permissions = 400
  }
}
