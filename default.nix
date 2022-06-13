{ pkgs      ? import <nixpkgs> {}
, namespace ? "registry.terraform.io/corpix"
, repo      ? "terraform-provider-${name}"
, name      ? "nixos"
, version   ? "0.0.1"
}: let
  inherit (builtins)
    toString
    baseNameOf
    filterSource
    trace
  ;
  inherit (pkgs)
    symlinkJoin
    buildGoModule
  ;
  inherit (pkgs.lib)
    concatStringsSep
    hasPrefix
    hasSuffix
  ;
  inherit (pkgs.nix-gitignore)
    gitignoreSourcePure
  ;

  ##

  providerSourceFilter = name: type:
    let bname = baseNameOf name;
    in
      ((type == "regular")
       && ((hasSuffix ".go" name)
           || (hasSuffix ".s" name)
           || (hasSuffix "/go.sum" name)
           || (hasSuffix "/go.mod" name)
           || (hasSuffix "/provider/nix_conf_wrapper.nix" name)
           || (hasSuffix "/vendor/modules.txt" name)))
      || ((type == "directory")
          && !(hasPrefix "." bname));
  sources = let src = filterSource providerSourceFilter ./.;
            in trace "sources: ${src}" src;

  ##

  builder = import ./builder.nix;
  build = platform: builder
    {
      name = repo;
      version = version;
      src = sources;

      provider-source-address = "${namespace}/${name}";

      inherit (platform)
        GOOS
        GOARCH
      ;
      inherit buildGoModule;
    };

  artifacts = map build
    [
      { GOOS = "linux";  GOARCH = "amd64"; }
      { GOOS = "linux";  GOARCH = "arm64"; }
      { GOOS = "darwin"; GOARCH = "amd64"; }
      { GOOS = "darwin"; GOARCH = "arm64"; }
    ];
in symlinkJoin {
  name = repo;
  paths = artifacts;
}
