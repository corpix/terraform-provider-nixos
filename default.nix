{ pkgs      ? import <nixpkgs> {}
, namespace ? "registry.terraform.io/corpix"
, name      ? "nixos"
, version   ? "0.0.1"
}: let
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
    name = "terraform-provider-${name}";
    version = version;
    provider-source-address = "${namespace}/${name}";
  }
