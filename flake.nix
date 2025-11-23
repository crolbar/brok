{
  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs = inputs: let
    system = "x86_64-linux";
    pkgs = import inputs.nixpkgs {inherit system;};
  in {
    devShells.${system}.default = pkgs.mkShell {
      packages = with pkgs; [];
    };

    packages.${system} = {
      default = inputs.self.packages.${system}.brokctl;

      brokd = pkgs.buildGoModule {
        pname = "brokd";
        version = "v0.1";
        src = ./brokd;
        vendorHash = null;
      };

      brokctl = pkgs.buildGoModule {
        pname = "brokctl";
        version = "v0.1";
        src = ./brokctl;
        vendorHash = null;
      };
    };
  };
}
