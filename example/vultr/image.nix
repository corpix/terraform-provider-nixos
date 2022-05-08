{ pkgs, lib, ... }:
let
  inherit (builtins)
    readFile
  ;
  inherit (lib)
    mkForce
    mkDefault
  ;
in {
  config = {
    users = rec {
      mutableUsers = false;
      extraUsers.root = {
        isNormalUser = false;
        hashedPassword = users.root.hashedPassword;
        openssh.authorizedKeys.keys = [
          (readFile ./authorized_key)
        ];
      };
      users.root.hashedPassword = "!";
    };

    services = {
      openssh.enable = true;
      openssh.passwordAuthentication = false;
      haveged.enable = true;
    };
    systemd.enableEmergencyMode = mkForce true;
    systemd.services = {
      "autovt@tty1".enable = mkForce false;
      "autovt@".enable = mkForce true;
      sshd.wantedBy = mkForce ["multi-user.target"];
    };

    networking.usePredictableInterfaceNames = false;
    networking.dhcpcd.allowInterfaces = ["eth*"];

    boot = {
      initrd.enable = true;
      initrd.includeDefaultModules = true;
      consoleLogLevel = 5;
      cleanTmpDir = true;
      loader.timeout = mkForce 3;
      loader.grub = {
        version = 2;
        configurationName = "Latest";
        configurationLimit = 10;
        splashImage = null;
      };
      kernelParams = ["console=tty1"];
      kernelPackages = mkDefault pkgs.linuxPackages_latest;
    };

    console.font = "Lat2-Terminus16";
    console.keyMap = "us";
    console.earlySetup = true;
    i18n.defaultLocale = "en_US.UTF-8";
    time.timeZone = "UTC";
    documentation.nixos.enable = false;
  };
}
