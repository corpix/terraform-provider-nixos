{ config, pkgs, lib, ... }:
let
  inherit (builtins)
    fromJSON
  ;
  inherit (lib)
    types
    mkOption
  ;
in {
  config = {
    users = {
      mutableUsers = false;
      extraUsers.root = {
        isNormalUser = false;
        hashedPassword = "!";
        shell = "${pkgs.fish}/bin/fish";
      };
    };

    fileSystems.rootfs = {
      label = "rootfs";
      device = "/dev/sda";
      fsType = "ext4";
      mountPoint = "/";
    };
    boot.loader.grub.devices = ["/dev/sda"];
  };
}
