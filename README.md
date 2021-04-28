# ffsync

A small application to synchronize a directory with another and transcode MP3,
FLAC and AAC files to Opus 96k.

## Usage

```nix
{ config, pkgs, lib, ... }:

{
	imports = [
		(builtins.fetchGit { url = "https://github.com/diamondburned/ffsync.git"; })
	];

	services.ffsync = {
		src  = "/mnt/Music/";
		dst  = "/mnt/Music.opus/";
		vars = {
			"FFSYNC_BITRATE" = "192k";
		};
	};
}
```
