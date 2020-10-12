{ lib, pkgs ? <nixpkgs> }:

pkgs.buildGoModule {
	name = "ffsync";
	version = "0.0.0-1";

	src = ./.;

	buildInputs = with pkgs; [
		ffmpeg
	];

	CGO_ENABLED = "0";

	vendorSha256 = "0rr3nj0zkbca2c7n0hx5li46x44wdl32qhzhi6x76bb0anpi1zcw";
}
