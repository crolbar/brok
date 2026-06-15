{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = inputs: let
    system = "x86_64-linux";
    pkgs = import inputs.nixpkgs {inherit system;};
  in {
    devShells.${system}.default = pkgs.mkShell {
      packages = with pkgs; [
        go
        gopls
      ];
    };

    packages.${system} = {
      default = inputs.self.packages.${system}.brokctl;

      brokd = pkgs.buildGoModule {
        pname = "brokd";
        version = "v0.1";
        src = ./brokd;
        vendorHash = "sha256-tH8eTuuOQWvp8cZTCyPXgxcfG6E9cA8/WC29U/X4zCQ=";
      };

      brokctl = pkgs.buildGoModule {
        pname = "brokctl";
        version = "v0.1";
        src = ./brokctl;
        vendorHash = "sha256-tH8eTuuOQWvp8cZTCyPXgxcfG6E9cA8/WC29U/X4zCQ=";
      };
    };

    homeManagerModules = {
      default = import ./module.nix inputs;
    };
  };
}
