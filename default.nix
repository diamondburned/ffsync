{ lib, pkgs ? <nixpkgs> }:

pkgs.buildGoModule {
	name = "ffsync";
	version = "0.0.0-1";

	src = ./.;

	buildInputs = with pkgs; [
		ffmpeg
		opusTools
	];

	CGO_ENABLED = "0";

	vendorSha256 = "1x1igf9g5kq6dcf1xv9nji5x3iclcdh41p698w8sqpzjdjli3vbr";
}
