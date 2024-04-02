{
  description = "Search GitHub Data";
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  inputs.flake-utils.url = "github:numtide/flake-utils";
  inputs.devshell.url = "github:numtide/devshell";

  outputs = { nixpkgs, flake-utils, devshell, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          overlays = [ 
            devshell.overlays.default
          ];
        };
        packages = with pkgs; [
          act
          docker-compose
          go
          gh
        ];
      in
      {
        devShells.default = pkgs.devshell.mkShell rec {
          name = "searchgh";
          inherit packages;
          devshell.startup."setprompt" = pkgs.lib.noDepEntry ''
            export LP_MARK_PREFIX=" (searchgh) "
          '';
          devshell.startup."printpackages" = pkgs.lib.noDepEntry ''
            echo "[[ Packages ]]"
            echo "${builtins.concatStringsSep "\n" (builtins.map (p: p.name) packages)}"
            echo ""
          '';
        };
      }
    );
}
