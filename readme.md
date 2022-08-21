# terraform-provider-nixos

Terraform provider to provision NixOS instances (via SSH).

This provider is in alpha stage. Things may change.

## examples

[Example directory](./example) contains examples for some cloud providers which I use. You may add your own, pull requests are welcome.

Here is a simple example which shows how to apply configuration from `test.nix` to `127.0.0.1`:

> test.tf

```hcl
terraform {
  required_providers {
    nixos = {
      source = "corpix/nixos"
      version = "0.0.14"
    }
  }
}

provider "nixos" {}

resource "nixos_instance" "test" {
  address = ["127.0.0.1"]
  configuration = "test.nix"
}
```

> test.nix

```nix
{ config, ... }: {
  config = {
    fileSystems.rootfs = {
      label = "rootfs";
      device = "/dev/sda";
      fsType = "ext4";
      mountPoint = "/";
    };
    boot.loader.grub.devices = ["/dev/sda"];
  };
}
```

## install

### with Nix

You could use following derivation to install latest version:

```nix
{ pkgs ? import <nixpkgs> {}, lib ? pkgs.lib, ... }: let
  terraform = pkgs.terraform_1;
  mkProvider = terraform.plugins.mkProvider;
in terraform.withPlugins (p: [
  (mkProvider rec {
    owner = "corpix";
    repo = "terraform-provider-nixos";
    rev = "0.0.14";
    version = rev;
    sha256 = "sha256-4QATev3WtwpEwc4/+JjOBfvUVzUre15VZT7tXLkSrXM=";
    vendorSha256 = null;
    provider-source-address = "registry.terraform.io/corpix/nixos";
  })
])
```

### with Terraform

You could install provider from Terraform registry.

> Registry may be blocked for your country by HashiCorp (for rexample it is blocked for Russia)

Add this to your `.tf` file:

```hcl
terraform {
  required_providers {
    nixos = {
      source = "corpix/nixos"
      version = "0.0.14"
    }
  }
}
```

Then run `terraform init`.

## tips

### how should I see what the fuck is going on?

Sometimes Nix need to copy really big derivation.
We need a way to print progress.

At this moment Nix stderr is printed to terraform `INFO` logging level.

So you need one of this:

- `export TF_LOG=INFO` before running Terraform
- `TF_LOG=INFO terraform apply` to apply logging level to single Terraform run

## release

- `make docs` (regenerate docs)
- `git tag 0.0.1` (replace `0.0.1` with version)
- `make release` (will get latest tag from git)
- `make release/sign` (will sign release with `gpg`)
- `make release/publish` (will publish github release, requires `GITHUB_TOKEN` env var)
