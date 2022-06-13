{ name
, src
, provider-source-address
, version
, vendorSha256 ? null
, buildGoModule
, GOOS
, GOARCH
}: buildGoModule {
  pname = "${name}-${GOOS}-${GOARCH}";
  subPackages = ["."];

  inherit
    src
    version
    vendorSha256
  ;

  CGO_ENABLED = "0";
  ldflags = ["-extldflags=-static"];

  postConfigure = ''
    export GOOS=${GOOS}
    export GOARCH=${GOARCH}
  '';

  # Terraform allow checking the provider versions, but this breaks
  # if the versions are not provided via file paths.
  postBuild = ''
    (
      dir=$GOPATH/bin/${GOOS}_${GOARCH}
      if [[ -n "$(shopt -s nullglob; echo $dir/*)" ]]
      then
        mv $dir/* $dir/..
      fi
      if [[ -d $dir ]]
      then
        rmdir $dir
      fi
    )
    mv $GOPATH/bin/${name} $GOPATH/bin/${name}_v${version}
  '';
  postInstall = ''
    dir=$out/libexec/terraform-providers/${provider-source-address}/${version}/${GOOS}_${GOARCH}
    mkdir -p "$dir"
    mv $out/bin/* "$dir/terraform-provider-$(basename ${provider-source-address})_${version}"
    rmdir $out/bin
  '';
}
