{ pkgs  ? import <nixpkgs> {} }: let
  inherit (pkgs)
    buildGoModule
  ;
  inherit (pkgs.nix-gitignore)
    gitignoreSourcePure
  ;

  ##

  mkProvider = import ./builder.nix {
    inherit buildGoModule;
  };

in mkProvider
  (gitignoreSourcePure [./.gitignore] ./.)
  {
    name = "terraform-provider-nixos";
    version = "0.0.1";
    provider-source-address = "registry.terraform.io/corpix/nixos";
  }
