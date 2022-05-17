# terraform-provider-nixos

Terraform provider to provision NixOS instances (via SSH).

This provider is in alpha stage. Things may change.

## examples

[Example directory](./example) contains examples for some cloud providers which I use. You may add your own, pull requests are welcome.

## release

- `git tag 0.0.1` (replace `0.0.1` with version)
- `make release` (will get latest tag from git)
- `make release/sign` (will sign release with `gpg`)
