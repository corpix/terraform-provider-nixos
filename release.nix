{ pkgs ? import <nixpkgs> {}, lib ? pkgs.lib, ... }: let
  terraform = pkgs.terraform_1;
  mkProvider = terraform.plugins.mkProvider;
in terraform.withPlugins (p: [
  (mkProvider rec {
    owner = "corpix";
    repo = "terraform-provider-nixos";
    rev = "0.0.7";
    version = rev;
    sha256 = "sha256-Ede1i1EAjvSxGRG3CmetkIZf/C7pzw944PF1Bvuud/o=";
    vendorSha256 = null;
    provider-source-address = "registry.terraform.io/corpix/nixos";
  })
])
