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
