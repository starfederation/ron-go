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
        default = pkgs.stdenvNoCC.mkDerivation {
          pname = "ron-go-tests";
          version = "0";
          src = self;
          nativeBuildInputs = [ pkgs.go_1_25 ];

          buildPhase = ''
            runHook preBuild
            export HOME=$TMPDIR
            export GOCACHE=$TMPDIR/go-cache
            ln -s ${ron}/testdata testdata
            go test -count=1 ./...
            runHook postBuild
          '';

          installPhase = ''
            runHook preInstall
            touch $out
            runHook postInstall
          '';
        };
      });

      devShells = forAllSystems (pkgs: {
        default = pkgs.mkShell {
          packages = [
            pkgs.go_1_25
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
