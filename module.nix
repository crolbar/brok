inputs: {
  config,
  pkgs,
  lib,
  ...
}: let
  inherit (lib.options) mkOption mkEnableOption;

  inherit (lib) types mkIf optional;
  cfg = config.services.brok;

  pkg = inputs.self.packages.x86_64-linux.brokd;
  pkgBin = inputs.self.packages.x86_64-linux.brokctl;
in {
  options.services.brok = {
    enable = mkEnableOption "brok";
    package = mkOption {
      type = types.package;
      default = pkg;
    };
  };
  config = mkIf cfg.enable {
    home.packages =
      optional (cfg.package != null) pkgBin;

    systemd.user.services.brokd = {
      Unit = {
        Description = "brokd";
        After = ["graphical-session.target"];
        PartOf = ["graphical-session.target"];
      };
      Service = {
        ExecStart = "${lib.getExe' cfg.package "brokd"}";
        Restart = "on-failure";
      };
      Install.WantedBy = ["graphical-session.target"];
    };
  };
}
