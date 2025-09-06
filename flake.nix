{
  description = "pgbox - PostgreSQL-in-Docker development environment";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "pgbox";
          version = "0.1.0";
          
          src = ./.;
          
          vendorHash = "sha256-qpDNiYOzuXGzyV6m5KG2vtamJKOO6dNgF/2ga82jUZA=";
          
          buildPhase = ''
            runHook preBuild
            go build -ldflags="-s -w" -o pgbox ./cmd/pgbox
            runHook postBuild
          '';
          
          installPhase = ''
            runHook preInstall
            mkdir -p $out/bin
            cp pgbox $out/bin/
            runHook postInstall
          '';
          
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
            maintainers = [ ];
          };
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            docker
            docker-compose
            go
            golangci-lint
            postgresql # For psql client
          ];
        };
      });
}
