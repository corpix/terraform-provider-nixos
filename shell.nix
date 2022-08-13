let
  nixpkgs = <nixpkgs>;
  config = {};
  pkgs = import nixpkgs { inherit config; };

  inherit (pkgs)
    writeScript
    stdenv
    fetchgit
    buildGoModule
  ;

  shellWrapper = writeScript "shell-wrapper" ''
    #! ${stdenv.shell}
    set -e

    exec -a shell ${pkgs.fish}/bin/fish --login --interactive "$@"
  '';

  terraform-plugin-docs = buildGoModule rec {
    pname = "terraform-plugin-docs";
    version = "0.8.1";
    src = fetchgit {
      url = "https://github.com/hashicorp/terraform-plugin-docs";
      rev = "v${version}";
      sha256 = "sha256-B1d/03RuR7Ns8VlRzcq86gAmuGDzY4yZAW9EFNW6SLE=";
    };
    vendorSha256 = "sha256-4soVDzu4gHT+Aq8/E4D4ib2aJu0/05mWgrVOs54ZW5E=";
  };

  terraform = pkgs.terraform_1.withPlugins (p: [
    p.null
    p.external
    p.vultr
    (import ./default.nix {
      inherit pkgs;
      targets = [
        { GOOS = "linux";  GOARCH = "amd64"; }
      ];
    })
  ]);
in stdenv.mkDerivation rec {
  name = "nix-shell";
  buildInputs = with pkgs; [
    glibcLocales bashInteractive man
    nix cacert curl utillinux coreutils
    git jq yq-go tmux findutils gnumake
    go gopls golangci-lint
    terraform terraform-ls terraform-plugin-docs
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

    # tokens which are required during tests/release
    # export TF_VAR_VULTR_API_KEY=$(pass ...)
    # export GITHUB_TOKEN=$(pass ...)

    if [ ! -z "$PS1" ]
    then
      export SHELL="${shellWrapper}"
      exec "$SHELL"
    fi
  '';
}
