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

		vars = mkOption {
			type = types.attrsOf types.str;
			default = {};
			example = {
				"FFSYNC_BITRATE"   = "192k";
				"FFSYNC_FREQUENCY" = "1m";
			};
			description = "Environment variables for ffsync";
		};

		influx = mkOption {
			type = types.submodule {
				options = {
					address = mkOption {
						type = types.str;
						default = "";
						description = "InfluxDB address";
					};
					username = mkOption {
						type = types.str;
						default = "";
						description = "InfluxDB username";
					};
					password = mkOption {
						type = types.str;
						default = "";
						description = "InfluxDB password";
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
			default = pkgs.callPackage ./package.nix {};
			type = types.package;
		};
	};

	config = mkIf (cfg.src != "" && cfg.dst != "") {
		users.users.ffsync = {
			home       = cfg.dst;
			group      = "users";
			createHome = true;
		};

		systemd.services.ffsync = {
			description = "ffsync";
			after    = [ "influxdb.service" ];
			wants    = [ "influxdb.service" ];
			wantedBy = [ "multi-user.target" ];
			environment = {
				FFSYNC_INFLUX_ADDRESS  = cfg.influx.address;
				FFSYNC_INFLUX_DATABASE = cfg.influx.database;
				FFSYNC_INFLUX_USERNAME = cfg.influx.username;
				FFSYNC_INFLUX_PASSWORD = cfg.influx.password;
			} // cfg.vars;
			path = with pkgs; [ ffmpeg ];
			serviceConfig = {
				ExecStart = ''${cfg.package}/bin/ffsync \
					${lib.escapeShellArg cfg.src} \
					${lib.escapeShellArg cfg.dst}
				'';
				Type  = "simple";
				User  = "ffsync";
				Group = "users";
				Restart = "on-failure";
				KillMode    = "control-group";
				KillSignal  = "SIGINT";
				LimitNICE   = 5; # lowish
				LimitNOFILE = 1024;
				ReadOnlyPaths  = cfg.src;
				ReadWritePaths = cfg.dst;
				NoNewPrivileges = true;
				RemoveIPC  = true;
				PrivateTmp = true;
				ProtectSystem = "strict";
				ProtectHome = "read-only";
			};
		};
	};
}
