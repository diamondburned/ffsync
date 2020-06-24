{ config, lib, pkgs, ... }: 

with lib;

let cfg = config.services.ffsync;

in {
	options.services.ffsync = {
		src = mkOption {
			type = types.str;
			default = "";
			description = "Source directory to watch";
		};

		dst = mkOption {
			type = types.str;
			default = "";
			description = "Destination directory to convert to";
		};

		influx = mkOption {
			type = types.submodule {
				options = {
					address = mkOption {
						type = types.str;
						default = "";
						description = "InfluxDB address";
					};
					token = mkOption {
						type = types.str;
						default = "";
						description = "InfluxDB token";
					};
					database = mkOption {
						type = types.str;
						default = "ffsync";
						description = "InfluxDB database name";
					};
				};
			};
		};

		package = mkOption {
			default = pkgs.callPackage ./default.nix {};
			type = types.package;
		};
	};

	config = mkIf (cfg.src != "" && cfg.dst != "") {
		systemd.services.ffsync = {
			description = "ffsync";
			after    = [ "influxdb.service" ];
			wants    = [ "influxdb.service" ];
			wantedBy = [ "multi-user.target" ];
			environment = {
				FFSYNC_INFLUX_ADDRESS  = cfg.influx.address;
				FFSYNC_INFLUX_TOKEN    = cfg.influx.token;
				FFSYNC_INFLUX_DATABASE = cfg.influx.database;
			};
			path = with pkgs; [ ffmpeg opusTools ];
			serviceConfig = {
				ExecStart = ''${cfg.package}/bin/ffsync \
					${lib.escapeShellArg cfg.src} \
					${lib.escapeShellArg cfg.dst}
				'';
				Type  = "simple";
				Restart = "on-failure";
				NoNewPrivileges = true;
				LimitNICE   = 5; # lowish
				LimitNPROC  = 64;
				LimitNOFILE = 128;
				DynamicUser    = true;
				PrivateTmp     = true;
				PrivateDevices = true;
				ProtectHome    = true;
				ProtectSystem  = "strict";
				ReadOnlyPaths  = cfg.src;
				ReadWritePaths = cfg.dst;
				InaccessiblePaths = "/";
			};
		};
	};
}
