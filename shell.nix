let
  nixpkgs = <nixpkgs>;
  config = {};
  pkgs = import nixpkgs { inherit config; };

  inherit (pkgs)
    writeScript
    stdenv
    buildGoModule
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

  terraform = pkgs.terraform_1.withPlugins (p: [
    p.null
    p.external
    p.vultr
    (import ./default.nix { inherit pkgs; })
  ]);
in stdenv.mkDerivation rec {
  name = "nix-shell";
  buildInputs = with pkgs; [
    glibcLocales bashInteractive man
    nix cacert curl utillinux coreutils
    git jq yq-go tmux findutils gnumake
    go gopls golangci-lint
    terraform terraform-ls
    github-cli
    nixos-generators
    zip
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
