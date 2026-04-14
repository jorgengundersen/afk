{
  description = "afk - AFK automation tool";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "aarch64-linux" "x86_64-darwin" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {
      packages = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.buildGo126Module {
            pname = "afk";
            version = "2.0.1";
            src = ./.;
            vendorHash = null;
            preCheck = ''
              export HOME="$TMPDIR/home"
              mkdir -p "$HOME"
            '';
            meta = {
              description = "AFK automation tool";
              license = pkgs.lib.licenses.mit;
              mainProgram = "afk";
            };
          };
        });

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = [ pkgs.go_1_26 ];
          };
        });
    };
}
