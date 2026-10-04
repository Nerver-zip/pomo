{
  description = "A timer-first Pomodoro timer TUI with persistent task context written in Go.";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs =
    { nixpkgs, ... }:
    let
      allSystems = [
        "x86_64-linux"
        "aarch64-linux"
        "x86_64-darwin"
        "aarch64-darwin"
      ];
      forAllSystems =
        f:
        nixpkgs.lib.genAttrs allSystems (
          system:
          f {
            pkgs = import nixpkgs { inherit system; };
          }
        );
    in
    {
      packages = forAllSystems (
        { pkgs }:
        {
          default = pkgs.buildGoModule {
            pname = "pomo";
            version = "1.3.1";

            src = ./.;

            vendorHash = null;

            ldflags = [
              "-s"
              "-w"
            ];

            meta = with pkgs.lib; {
              description = "Timer-first Pomodoro TUI with persistent task context, progress bar, desktop notifications, and focus statistics";
              homepage = "https://github.com/Nerver-zip/pomo";
              license = licenses.mit;
              platforms = platforms.linux ++ platforms.darwin;
              mainProgram = "pomo";
            };
          };
        }
      );
    };
}
