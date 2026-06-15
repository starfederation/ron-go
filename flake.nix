{
  description = "Go reference implementation for RON";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    ron = {
      url = "github:starfederation/ron";
      flake = false;
    };
  };

  outputs = { self, nixpkgs, ron }:
    let
      systems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-darwin"
        "x86_64-linux"
      ];
      forAllSystems = fn:
        nixpkgs.lib.genAttrs systems (system: fn (import nixpkgs { inherit system; }));
    in
    {
      checks = forAllSystems (pkgs: {
        default = pkgs.buildGo126Module {
          pname = "ron-go-tests";
          version = "0.0.3";
          src = self;
          vendorHash = "sha256-1oCcFzFYYNdSOJ3anzUvGO+YNDJ2UtHRuY5UIElRpgg=";
          proxyVendor = true;

          preCheck = ''
            ln -s ${ron}/testdata testdata
          '';
        };
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = [
            pkgs.go_1_26
            pkgs.gopls
          ];

          shellHook = ''
            if [ ! -e testdata ]; then
              ln -s ${ron}/testdata testdata
            elif [ -L testdata ]; then
              ln -sfn ${ron}/testdata testdata
            else
              echo "testdata exists and is not a symlink; leaving it alone" >&2
            fi
          '';
        };
      });
    };
}
