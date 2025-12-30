{
  description = "pgbox - PostgreSQL-in-Docker development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = {
    self,
    nixpkgs,
    flake-utils,
  }:
    flake-utils.lib.eachDefaultSystem (
      system: let
        pkgs = nixpkgs.legacyPackages.${system};
      in {
        packages.default = pkgs.buildGoModule rec {
          pname = "pgbox";
          # Read base version from VERSION file (single source of truth)
          baseVersion = builtins.replaceStrings ["\n"] [""] (builtins.readFile ./VERSION);
          version =
            if (self ? shortRev)
            then "${baseVersion}-${self.shortRev}"
            else baseVersion;

          src = ./.;

          vendorHash = "sha256-d74qMuSWFX12B9tPKT7DfTr50ssOmyCJLAMg6hSka78=";

          ldflags = [
            "-s"
            "-w"
            "-X main.version=${version}"
            "-X main.commit=${self.rev or "unknown"}"
          ];

          env.CGO_ENABLED = 0;

          buildInputs = with pkgs; [
            docker
            docker-compose
            postgresql # For psql client at runtime
          ];

          meta = with pkgs.lib; {
            description = "PostgreSQL-in-Docker development environment";
            homepage = "https://github.com/ahacop/pgbox";
            license = licenses.mit; # Update to match your license
            maintainers = [
              {
                name = "Ara Hacopian";
                github = "ahacop";
              }
            ];
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            docker
            docker-compose
            go
            golangci-lint
            goreleaser
            postgresql # For psql client
          ];
        };
      }
    );
}
