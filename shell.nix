let
  nixpkgs = <nixpkgs>;
  config = {};
  pkgs = import nixpkgs { inherit config; };

  inherit (pkgs)
    writeScript
    stdenv
    buildGoModule
  ;
  inherit (pkgs.nix-gitignore)
    gitignoreSourcePure
  ;

  shellWrapper = writeScript "shell-wrapper" ''
    #! ${stdenv.shell}
    set -e

    exec -a shell ${pkgs.fish}/bin/fish --login --interactive --init-command='
      set -x root '"$root"'
      set config $root/.fish.conf
      set personal_config $root/.personal.fish.conf
      if test -e $personal_config
        source $personal_config
      end
      if test -e $config
        source $config
      end
    ' "$@"
  '';

  terraform = let
    mkProvider = src: desc: buildGoModule {
      pname = desc.name;
      version = desc.version;
      inherit src;

      subPackages = ["."];
      vendorSha256 = desc.vendorSha256 or null;

      # Terraform allow checking the provider versions, but this breaks
      # if the versions are not provided via file paths.
      postBuild = "mv $NIX_BUILD_TOP/go/bin/${desc.name}{,_v${desc.version}}";
      postInstall = with desc; ''
        dir=$out/libexec/terraform-providers/${provider-source-address}/${version}/''${GOOS}_''${GOARCH}
        mkdir -p "$dir"
        mv $out/bin/* "$dir/terraform-provider-$(basename ${provider-source-address})_${version}"
        rmdir $out/bin
      '';
      passthru = desc;
    };

    nixosSrc = gitignoreSourcePure ["/test" ./.gitignore] ./.;
    nixos = mkProvider nixosSrc {
      name = "terraform-provider-nixos";
      version = "0.0.1";
      provider-source-address = "registry.terraform.io/corpix/nixos";
    };
  in pkgs.terraform_1.withPlugins (p: [
    p.null
    p.external
    p.vultr
    nixos
  ]);

in stdenv.mkDerivation rec {
  name = "nix-shell";
  buildInputs = with pkgs; [
    glibcLocales bashInteractive man
    nix cacert curl utillinux coreutils
    git jq yq-go tmux findutils gnumake
    go gopls golangci-lint
    terraform terraform-ls
  ];
  shellHook = ''
    export root=$(pwd)

    if [ -f "$root/.env" ]
    then
      source "$root/.env"
    fi

    export LANG="en_US.UTF-8"
    export NIX_PATH="nixpkgs=${nixpkgs}"

    export TF_VAR_VULTR_API_KEY=

    if [ ! -z "$PS1" ]
    then
      export SHELL="${shellWrapper}"
      exec "$SHELL"
    fi
  '';
}
