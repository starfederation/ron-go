{
  description = "Go reference implementation for RON";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    ron = {
      url = "github:starfederation/ron/47c128ee658a0f49cd8e2d8b5bb571958a498f26";
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
          version = "0.0.9";
          src = self;
          vendorHash = "sha256-Q9o76rYrQauzcyHUEtPbfuSWtzG5sMu0zYhaQ7I03hI=";
          proxyVendor = true;

          preCheck = ''
            export RON_TESTDATA_DIR=${ron}/testdata
          '';
        };
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = [
            pkgs.go_1_26
            pkgs.gopls
          ];

        };
      });
    };
}
