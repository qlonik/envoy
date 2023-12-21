{
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-parts.url = "github:hercules-ci/flake-parts";
    devshell.url = "github:numtide/devshell";
  };

  outputs = inputs@{ flake-parts, ... }:
    flake-parts.lib.mkFlake { inherit inputs; } {
      imports = [
        inputs.devshell.flakeModule
      ];
      systems = [ "x86_64-linux" "aarch64-linux" "aarch64-darwin" "x86_64-darwin" ];
      perSystem = { config, self', inputs', pkgs, system, ... }: {
        devshells.default = {
          devshell.packages = with pkgs; [
            clang
            libcxx
            libcxxabi
          ];
          commands = [
            {
              name = "bazel";
              help = "run bazel via bazelisk";
              command = ''${pkgs.bazelisk}/bin/bazelisk "$@"'';
            }
          ];
        };
      };
    };
}
