{ buildGoModule }: src: desc: buildGoModule {
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

  CGO_ENABLED = "0";
  ldflags = ["-extldflags=-static"];
}
