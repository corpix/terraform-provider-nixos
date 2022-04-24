{ config, ... }: {
  config = {
    fileSystems.rootfs = {
      label = "rootfs";
      device = "/dev/sda";
      fsType = "ext4";
      mountPoint = "/";
    };
    boot.loader.grub.devices = ["/dev/sda"];
  };
}
