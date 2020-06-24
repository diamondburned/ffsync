{ config, lib, pkgs, ... }: 

let cfg = config.services.ffsync;

in {
	options.services.ffsync = {
		src = mkOption {
			type = types.str;
			description = "Source directory to watch";
		};

		dst = mkOption {
			type = types.str;
			description = "Destination directory to convert to";
		};

		influx = mkOption {
			type = types.submodule {
				options = {
					address = mkOption {
						type = types.str;
						description = "InfluxDB address";
					};
					token = mkOption {
						type = types.str;
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
			default = pkgs.callPackage ./default.nix;
			type = types.package;
		};
	};

	config = mkIf (cfg.src != "" && cfg.dst != "") {
		users.users.ffsync = {
			home       = cfg.dst;
			createHome = true;
		};

		systemd.services.ffsync = {
			description = "ffsync";
			after    = [ "influxdb.service" ];
			wantedBy = [ "multi-user.target" ];
			environment = {
				FFSYNC_INFLUX_ADDRESS  = cfg.influx.address;
				FFSYNC_INFLUX_TOKEN    = cfg.influx.token;
				FFSYNC_INFLUX_DATABASE = cfg.influx.database;
			};
			serviceConfig = {
				ExecStart = ''${cfg.package} \
					${lib.escapeShellArg cfg.src} \
					${lib.escapeShellArg cfg.dst}
				'';
				Type  = "simple";
				User  = "ffsync";
				Group = "ffsync";
				Restart = "on-failure";
				NoNewPrivileges = true;
				LimitNICE   = 5; # lowish
				LimitNPROC  = 64;
				LimitNOFILE = 128;
				PrivateTmp     = true;
				PrivateDevices = true;
				ProtectHome    = true;
				ProtectSystem  = "full";
				ReadOnlyPaths  = cfg.src;
				ReadWritePaths = cfg.dst;
			};
		};
	};
}
