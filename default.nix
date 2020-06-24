{ lib, pkgs ? <nixpkgs> }:

pkgs.buildGoModule {
	name = "ffsync";
	version = "0.0.0-1";

	src = ./.;

	buildInputs = with pkgs; [
		ffmpeg
		opusTools
	];

	vendorSha256 = "0k36dz5y448nxvx9nh36qs8z9968knyalvhss6k1m67m2m0kqxjy";
}
