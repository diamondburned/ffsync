{ lib, pkgs ? <nixpkgs> }:

pkgs.buildGoModule {
	name = "ffsync";
	version = "0.0.0-1";

	src = ./.;

	CGO_ENABLED = "0";

	vendorSha256 = "0lyxkf2yi5pg68f0462f7ar1m4vsaal09cfbmz3fkhwdzj221i6c";
}
