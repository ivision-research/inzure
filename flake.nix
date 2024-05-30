{
  description = "inzure - a library for automated security testing of Azure environments";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-23.11";
    flake-utils.url = "github:numtide/flake-utils";
  };
  outputs = { self, nixpkgs, flake-utils }: flake-utils.lib.eachDefaultSystem (system:
    let
      pkgs = nixpkgs.legacyPackages.${system};
      version = "1.0.0";

    in rec {

      apps.inzure = {
        type = "app";
        program = "${packages.cli}/bin/inzure";
      };

      packages.default = packages.lib;

      packages.cli = pkgs.buildGoModule {
        inherit version;
        pname = "inzure";
        src = ./.;
        doCheck = false;
        postUnpack = ''
        chmod -R +w $sourceRoot
        rm -r "$sourceRoot/pkg/inzure/qs"
        rm -r "$sourceRoot/pkg/inzure/gen"
        '';
        vendorHash = "sha256-xlusRvK/AZDknaq0uclDPneg1Ea3CyG6wcab4VCikVo=";
        modRoot = "./cmd/inzure";
      };
    }
  );
}
