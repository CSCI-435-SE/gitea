{
  inputs = {
    nixpkgs.url = "https://channels.nixos.org/nixpkgs-unstable/nixexprs.tar.zst";
  };
  outputs =
    { nixpkgs, ... }:
    let
      # nixpkgs 26.11 dropped x86_64-darwin
      supportedSystems = [
        "aarch64-darwin"
        "aarch64-linux"
        "x86_64-linux"
      ];

      forEachSupportedSystem =
        f:
        nixpkgs.lib.genAttrs supportedSystems (
          system:
          let
            pkgs = import nixpkgs {
              inherit system;
            };
          in
          f { inherit pkgs; }
        );

      # only bump toolchain versions here
      toolchain = pkgs: {
        go = pkgs.go_1_26;
        nodejs = pkgs.nodejs_26;
        python3 = pkgs.python314;
        pnpm = pkgs.pnpm_11;
      };
    in
    {
      devShells = forEachSupportedSystem (
        { pkgs, ... }:
        {
          default =
            let
              inherit (pkgs) lib;
              inherit (toolchain pkgs)
                go
                nodejs
                python3
                pnpm
                ;

              # Platform-specific dependencies
              linuxOnlyInputs = lib.optionals pkgs.stdenv.hostPlatform.isLinux [
                pkgs.glibc.static
              ];

              linuxOnlyEnv = lib.optionalAttrs pkgs.stdenv.hostPlatform.isLinux {
                CFLAGS = "-I${pkgs.glibc.static.dev}/include";
                LDFLAGS = "-L ${pkgs.glibc.static}/lib";
              };
            in
            pkgs.mkShell {
              packages =
                with pkgs;
                [
                  # generic
                  git
                  git-lfs
                  gnumake
                  gnused
                  gnutar
                  gzip
                  zip

                  # frontend
                  nodejs
                  pnpm
                  cairo
                  pixman
                  pkg-config

                  # linting
                  python3
                  uv

                  # backend
                  go
                  gofumpt
                  sqlite
                ]
                ++ linuxOnlyInputs;

              env = {
                GO = "${go}/bin/go";
                GOROOT = "${go}/share/go";

                TAGS = "";
                STATIC = "true";
              }
              // linuxOnlyEnv;
            };
        }
      );

      # `nix fmt` runs the same formatting CI checks with `make fmt-check`
      formatter = forEachSupportedSystem (
        { pkgs, ... }:
        pkgs.writeShellApplication {
          name = "gitea-fmt";
          runtimeInputs = [
            (toolchain pkgs).go
            pkgs.git
            pkgs.gnumake
            pkgs.gnused
            pkgs.nixfmt
          ];
          text = ''
            cd "$(git rev-parse --show-toplevel)"
            make fmt
            nixfmt flake.nix
          '';
        }
      );
    };
}
