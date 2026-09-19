{
  description = "Resource specification development environment";
  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
    devshell.url = "github:numtide/devshell";
    devshell.inputs.nixpkgs.follows = "nixpkgs";
  };
  outputs = { nixpkgs, devshell, ... }:
    let
      systems = [ "aarch64-darwin" "x86_64-darwin" "aarch64-linux" "x86_64-linux" ];
      eachSystem = nixpkgs.lib.genAttrs systems;
    in {
      devShells = eachSystem (system:
        let pkgs = import nixpkgs { inherit system; overlays = [ devshell.overlays.default ]; };
        in {
          default = pkgs.devshell.mkShell {
            name = "dsa-resource-spec";
            packages = [ pkgs.go_1_27 pkgs.python3 pkgs.gopls pkgs.git pkgs.direnv pkgs.nix-direnv ];
            env = [ { name = "GOTOOLCHAIN"; value = "local"; } ];
            commands = [
              { name = "check"; command = "go vet ./... && go test ./... && python3 -B -m unittest discover -s tests -v"; help = "Vet and test all packages"; }
            ];
          };
        });
    };
}
