{ pkgs ? import <nixpkgs> {}, lib ? pkgs.lib, ... }: let
  terraform = pkgs.terraform_1;
  mkProvider = terraform.plugins.mkProvider;
in terraform.withPlugins (p: [
  (mkProvider rec {
    owner = "corpix";
    repo = "terraform-provider-nixos";
    rev = "0.0.11";
    version = rev;
    sha256 = "sha256-xXwv4g+IJtz3xEdhjF6MRqREbmj8Ubu8s/jvNxh17lk=";
    vendorSha256 = null;
    provider-source-address = "registry.terraform.io/corpix/nixos";
  })
])
