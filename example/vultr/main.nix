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
    boot.initrd.availableKernelModules = [
      "ata_piix"
      "sr_mod"
      "uhci_hcd"
      "virtio_blk"
      "virtio_pci"
    ];
    users = {
      mutableUsers = false;
      extraUsers.root = {
        isNormalUser = false;
        hashedPassword = "!";
        shell = "${pkgs.fish}/bin/fish";
      };
    };

    fileSystems.nixos = {
      label = "nixos";
      device = "/dev/vda1";
      fsType = "ext4";
      mountPoint = "/";
    };
    boot.loader.grub.devices = ["/dev/vda"];
  };
}
