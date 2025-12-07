{
	description = "Build dependencies";

	inputs = {
		nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
		sushy-lib = {
			url = "github:sushydev/nix-lib";
			inputs.nixpkgs.follows = "nixpkgs";
		};
	};

	outputs = { self, nixpkgs, sushy-lib }:
		let
			supportedSystems = [ "x86_64-linux" "aarch64-linux" ];

			mkDepsBundle = (system:
				let
					pkgs = import nixpkgs { inherit system; };
				in
				pkgs.stdenv.mkDerivation {
					name = "dependencies-bundle";
					dontUnpack = true;

					nativeBuildInputs = [
						pkgs.cacert
					];

					installPhase = ''
						mkdir -p $out/bin $out/etc/ssl/certs
						cp -L ${pkgs.cacert}/etc/ssl/certs/ca-bundle.crt $out/etc/ssl/certs/ca-certificates.crt
					'';
				}
			);
		in
		{
			packages = nixpkgs.lib.genAttrs supportedSystems (system: {
				default = mkDepsBundle system;
			});

			devShells = sushy-lib.forPlatforms sushy-lib.platforms.default (system: 
				let
					pkgs = import nixpkgs { inherit system; };
				in 
				{
					default = pkgs.mkShell {
						buildInputs = [
							pkgs.go
							pkgs.gnumake
						];
					};
				}
			);
		};
}
