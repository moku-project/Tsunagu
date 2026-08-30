{
  description = "Tsunagu";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      systems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];

      perSystem = { pkgs, lib, ... }: {

        packages = import ./nix/packages.nix { inherit pkgs lib; };

        devShells.default = pkgs.mkShell {
          name = "tsunagu-dev";

          packages = with pkgs; [
            go
            gopls
            golangci-lint
            air
            sqlc
            gqlgen

            # Java 21 for the sandbox (Gradle/Kotlin)
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

            python3

            # Node for compiling LNReader plugin .ts sources (esbuild via npx)
            nodejs_22
            # For muxing downloaded HLS anime streams into playable files
            ffmpeg
          ];

          # Set JAVA_HOME to JDK 17
          shellHook = ''
            export JAVA_HOME=${pkgs.jdk21}/lib/openjdk
            export PATH=$JAVA_HOME/bin:$PATH
            echo "Java 21 ready: $(java -version 2>&1 | head -n1)"
          '';
        };
      };

      flake = { };
    };
}
