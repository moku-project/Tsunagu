{ pkgs, lib }:

let
  version = "0.1.0";
  jdk = pkgs.jdk21;

  repoSrc = fileset: lib.fileset.toSource {
    root = ../.;
    inherit fileset;
  };

  grpcJava = pkgs.protoc-gen-grpc-java.overrideAttrs (_: {
    version = "1.65.0";
    src = pkgs.fetchurl {
      url =
        let
          p = pkgs.stdenv.hostPlatform;
          arch = if p.isAarch64 then "aarch_64" else "x86_64";
          os = if p.isDarwin then "osx" else "linux";
        in
        "https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.65.0/protoc-gen-grpc-java-1.65.0-${os}-${arch}.exe";
      hash = {
        "x86_64-linux" = "sha256-qfmnmHvko3xpqF5eiIU5Q1b9LxpH+EPGRh2k/Jn0B7M=";
        "aarch64-linux" = "sha256-MIQ16AuiiLCASz5Ei5rZdQC1RfgBKX636hQtuP6B+5w=";
        "x86_64-darwin" = "sha256-3O49dyCpLyR9vvwXKI24eK2ZgSk9spPIahr7YHSTXFM=";
        "aarch64-darwin" = "sha256-3O49dyCpLyR9vvwXKI24eK2ZgSk9spPIahr7YHSTXFM=";
      }.${pkgs.stdenv.hostPlatform.system};
    };
  });

  jre = pkgs.jre_minimal.override {
    inherit jdk;
    jdkOnBuild = jdk;
    modules = lib.splitString ","
      (lib.trim (builtins.readFile ../sandbox/build-config/jlink-modules.txt));
  };

  server = pkgs.buildGoModule {
    pname = "tsunagu-server";
    inherit version;

    src = repoSrc (lib.fileset.unions [ ../backend ]);
    modRoot = "backend";
    vendorHash = "sha256-bZ1T7nMgBrmD6hiN+1vuSDz/XxR0DxxnMxYnWP76FYU=";

    subPackages = [ "cmd/server" ];
    env.CGO_ENABLED = "0";
    ldflags = [ "-s" "-w" "-X" "main.serverVersion=${version}" ];
    doCheck = false;

    postInstall = "mv $out/bin/server $out/bin/tsunagu";
    meta.mainProgram = "tsunagu";
  };

  sandboxJar = pkgs.stdenv.mkDerivation (finalAttrs: {
    pname = "tsunagu-sandbox";
    inherit version;

    src = repoSrc (lib.fileset.unions [ ../sandbox ../proto ]);
    sourceRoot = "source/sandbox";

    nativeBuildInputs = [ pkgs.gradle jdk ];

    mitmCache = pkgs.gradle.fetchDeps {
      pkg = finalAttrs.finalPackage;
      data = ./sandbox-deps.json;
    };
    __darwinAllowLocalNetworking = true;

    gradleFlags = [ "-Dorg.gradle.java.installations.auto-download=false" ];
    gradleBuildTask = "shadowJar";
    doCheck = false;

    postPatch = ''
      chmod -R u+w .. ../proto
      rm -rf ../proto
      cp -rL ${../proto} ../proto
      chmod -R u+w ../proto

      substituteInPlace build.gradle.kts \
        --replace-fail 'artifact = "com.google.protobuf:protoc:$protobufVersion"' \
                       'path = "${pkgs.protobuf_25}/bin/protoc"' \
        --replace-fail 'artifact = "io.grpc:protoc-gen-grpc-java:$grpcVersion"' \
                       'path = "${grpcJava}/bin/protoc-gen-grpc-java"'
    '';

    installPhase = ''
      runHook preInstall
      install -Dm644 build/libs/sandbox-${version}-all.jar \
        $out/share/tsunagu/sandbox.jar
      runHook postInstall
    '';

    passthru.jar = "/share/tsunagu/sandbox.jar";
    meta.sourceProvenance = with lib.sourceTypes; [ fromSource binaryBytecode ];
  });

  full = pkgs.symlinkJoin {
    name = "tsunagu-${version}";
    paths = [ server ];
    nativeBuildInputs = [ pkgs.makeWrapper ];
    postBuild = ''
      wrapProgram $out/bin/tsunagu \
        --set TSUNAGU_SANDBOX_JAR ${sandboxJar}/share/tsunagu/sandbox.jar \
        --prefix PATH : ${lib.makeBinPath [ jre pkgs.ffmpeg-headless ]}
    '';
    meta.mainProgram = "tsunagu";
  };

in
{
  default = full;
  tsunagu = full;
  tsunagu-server = server;
  tsunagu-sandbox = sandboxJar;
  tsunagu-jre = jre;
}
