{ nixpkgs  ? <nixpkgs>
, system   ? builtins.currentSystem
, settings ? "{}"
, configuration
}:
let
  inherit (builtins)
    concatStringsSep
    fromJSON
  ;

  concat = concatStringsSep " ";

  ##

  configurationModule = { config, lib, pkgs, ... }:
    { imports = [ configuration ];
      config = fromJSON settings;
    };
  os = import "${nixpkgs}/nixos"
    { inherit system;
      configuration = configurationModule;
    };
in {
  currentSystem = system;

  # nix conf
  substituters        = concat os.config.nix.binaryCaches;
  trusted-public-keys = concat os.config.nix.binaryCachePublicKeys;

  drv_path = os.config.system.build.toplevel.drvPath;
  out_path = os.config.system.build.toplevel;
}
