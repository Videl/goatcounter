{
  description = "GoatCounter build with Nix";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-24.05";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs =
    { self
    , nixpkgs
    , flake-utils
    ,
    }:
    flake-utils.lib.eachDefaultSystem (
      system:
      let
        pkgs = import nixpkgs { inherit system; };
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "goatcounter";
          version = "0.1.0";
          src = ./.;
          vendorHash = null; # Set this to the correct hash or null for the first build

          nativeBuildInputs = [ pkgs.git ];

          preBuild = ''
            export HOME=$(mktemp -d)
          '';

          buildPhase = ''
            runHook preBuild
            go build -ldflags="-X zgo.at/goatcounter/v2.Version=$(git log -n1 --format='%h_%cI')" ./cmd/goatcounter
            runHook postBuild
          '';

          installPhase = ''
            mkdir -p $out/bin
            cp goatcounter $out/bin/
          '';

          # Handle the local replace directive
          # modPostPatch = ''
          #   substituteInPlace go.mod --replace \
          #     'replace zgo.at/bgrun => ./bgrun' \
          #     'replace zgo.at/bgrun => '"$src"'/bgrun'
          # '';
        };

        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            go
            git
          ];
        };
      }
    );
}
# {
#   inputs = {
#     nixpkgs.url = "nixpkgs/nixos-24.05";
#     utils.url = "github:numtide/flake-utils";
#     gomod2nix = {
#       url = "github:nix-community/gomod2nix";
#       inputs.nixpkgs.follows = "nixpkgs";
#     };
#   };
#
#   outputs =
#     inputs @ { self
#     , nixpkgs
#     , utils
#     , gomod2nix
#     ,
#     }:
#     utils.lib.eachDefaultSystem (system:
#     let
#       pkgs = import nixpkgs {
#         inherit system;
#         overlays = [ gomod2nix.overlays.default ];
#       };
#       version = "2.5";
#     in
#     rec {
#       packages.hello =
#         let
#           inherit (pkgs) stdenv lib;
#         in
#         stdenv.mkDerivation {
#           pname = "hello";
#           inherit version;
#
#           src = ./.;
#           buildPhase = ''
#             export GOCACHE=$(mktemp -d)
#             export GOMODCACHE=$(mktemp -d)
#             go build ./cmd/goatcounter
#           '';
#
#           buildInputs = with pkgs; [ go_1_21 ];
#
#           # nativeBuildInputs = [ autoreconfHook ];
#         };
#       packages.goatcounter =
#         pkgs.buildGoApplication
#           {
#             pname = "goatcounter";
#             version = "2.5";
#             # subPackages = ./cmd/goatcounter;
#             pwd = ./cmd/goatcounter;
#             src = ./.;
#             # ldflags = [ "-X zgo.at/goatcounter/v2.Version=2.5" ];
#             # pwd = ./.;
#             # src = ./cmd/goatcounter;
#             # preBuild = ''
#             #   ${pkgs.templ}/bin/templ generate
#             # '';
#             # fixupPhase =
#             #   # pkgs.lib.mkAfter
#             #   ''
#             #     mkdir -p $out/assets
#             #     cp -r $src/assets $out/assets
#             #     # ls $src
#             #     # echo `pwd`
#             #     # ls $out/bin
#             #   '';
#             modules = ./gomod2nix.toml;
#           };
#       defaultPackage = packages.goatcounter;
#       packages.default = packages.goatcounter;
#       packages.container = pkgs.dockerTools.buildImage {
#         name = "goatcounter";
#         tag = "flake";
#         created = "now";
#         config = {
#           ExposedPorts = {
#             "8081/tcp" = { };
#           };
#           # contents = [
#           #   ./assets
#           # ];
#
#           # pathsToLink = [ "/bin" ];
#           # paths = [
#           #   pkgs.coreutils
#           #   pkgs.bash
#           #   pkgs.emacs
#           #   pkgs.vim
#           #   pkgs.nano
#           # ];
#
#           copyToRoot =
#             packages.goatcounter;
#           # pkgs.buildEnv
#           #   {
#           #     name = "image-root";
#           #     paths = [ packages.default ];
#           #     pathsToLink = [ "/bin" ];
#           #   };
#           Cmd = [ "${packages.goatcounter}/bin/goatcounter" ];
#           WorkingDir = "${packages.goatcounter}/";
#         };
#       };
#
#       devShells.default =
#         pkgs.mkShell
#           {
#             # buildInputs = with pkgs; [
#             #   go_1_21
#             #   gopls
#             #   gotools
#             #   go-tools
#             #   gomod2nix.packages.${system}.default
#             #   sqlite-interactive
#             # ];
#             packages = with pkgs; [
#               go_1_21
#               gomod2nix.packages.${system}.default
#             ];
#           };
#     });
# }
