# terraform-provider-nixos

Terraform provider to provision NixOS instances (via SSH).

This provider is in alpha stage. Things may change.

## examples

[Example directory](./example) contains examples for some cloud providers which I use. You may add your own, pull requests are welcome.

## install

### with Nix

You could use following derivation to install latest version:

```nix
{ pkgs ? import <nixpkgs> {}, lib ? pkgs.lib, ... }: let
  terraform = pkgs.terraform_1;
  mkProvider = terraform.plugins.mkProvider;
in terraform.withPlugins (p: [
  # p.null
  # p.external
  # p.vultr
  (mkProvider rec {
    owner = "corpix";
    repo = "terraform-provider-nixos";
    rev = "0.0.6";
    version = rev;
    sha256 = "sha256-E+RGiebTZYlV4lf0dRWiCNv6i7rAADSyWkjpAJpVfUM=";
    vendorSha256 = null;
    provider-source-address = "registry.terraform.io/corpix/nixos";
  })
])
```



## release

- `make docs` (regenerate docs)
- `git tag 0.0.1` (replace `0.0.1` with version)
- `make release` (will get latest tag from git)
- `make release/sign` (will sign release with `gpg`)
- `make release/publish` (will publish github release, requires `GITHUB_TOKEN` env var)
