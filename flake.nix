{
  description = "Tsunagu";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];

      perSystem = { pkgs, system, ... }: {
        devShells.default = pkgs.mkShell {
          name = "tsunagu-dev";

          packages = with pkgs; [
            go
            gopls
            golangci-lint
            air
            sqlc
            gqlgen

            jdk21
            kotlin
            gradle

            protobuf
            protoc-gen-go
            protoc-gen-go-grpc
            grpcurl
            buf

            docker-compose
            postgresql
            sqlite

            vips
            pkg-config
          ];
        };

        packages = { };
      };

      flake = { };
    };
}
