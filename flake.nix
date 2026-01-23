{
  description = "inzure - a library for automated security testing of Azure environments";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs = { self, nixpkgs, flake-utils }: flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = nixpkgs.legacyPackages.${system};
      version = "1.3.0";

    in rec {

      apps.inzure = {
        type = "app";
        program = "${packages.inzure}/bin/inzure";
      };

      packages.default = packages.inzure;

      packages.inzure = pkgs.buildGoModule {
        inherit version;
        pname = "inzure";
        src = ./.;
        doCheck = false;
        postUnpack = ''
        chmod -R +w $sourceRoot
        rm -r "$sourceRoot/pkg/inzure/qs"
        rm -r "$sourceRoot/pkg/inzure/gen"
        '';
        vendorHash = "sha256-v4ywx0Q7EUyNt8qB7WvKiKBVVG28dAlbb/zrQFW60SY=";
        modRoot = "./cmd/inzure";
      };
    }
  );
}
