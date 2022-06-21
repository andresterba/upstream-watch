# upstream-watch

`upstream-watch` is a git-ops tool to monitor an upstream repository and execute update steps if necessary.
It only supports `git`.

I wrote this tool to support my personal container infrastructure, which is completely managed via a single
git repository.
There are two different modes:

- single directory
- Subdiretorie per service

## Single directory

The target repository could look like this:

```sh
    .
    ├── .upstream-watch.yaml
    ├── README.md
    ├── .update-hooks.yaml
    ├── docker-compose.yml
    └── upstream-watch
```

All configuration files are in a single directory, which must be the root of the git directory.
Use this mode by setting `single_directory_mode: true`.
The rest of the needed configuration is identical to the subdirectory documentation.

## Use subdirectories for different services

The target repository could look like this:

```sh
    .
    ├── .upstream-watch.yaml
    ├── README.md
    ├── service-1
    │   ├── .update-hooks.yaml
    │   ├── docker-compose.yml
    │   └── README.md
    ├── service-2
    │   ├── .update-hooks.yaml
    │   ├── docker-compose.yml
    │   └── README.md
    └── upstream-watch
```

The `.upstream-watch.yaml` is the main configuration file for this instance of `upstream-watch`.
You can set the retry interval (in seconds) and folders that should be ignored.

```sh
    single_directory_mode: false
    retry_intervall: 10
    ignore_folders: [".git", ".test"]
```

There are two services, each in its own subfolder.
Each of these services holds a `README.md` (which is not interesting), a `docker-compose.yml` that defines
the containers and a `.update-hooks.yaml`, which is the configuration file of `upstream-watch` for this specific service.

In case of an update to any of these files in a subfolder, `upstream-watch` will execute the pre- and post-hooks
defined in the corresponding `.update-hooks.yaml`.

An example for a `.update-hooks.yaml`:

```sh
	pre_update_commands: ["docker compose down"]
	update_commands: ["docker compose pull"]
	post_update_commands: ["docker compose up -d"]
```

`upstream-watch` will stop all containers, pull updates from the registry and start them afterwards.
Of course, you can do almost anything in these hooks, depending on the needs of your service.

### Requirements on the server

* Pull access to upstream repository
* `git` installed
