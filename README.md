# upstream-watch

`upstream-watch` is a git-ops tool to monitor an upstream repository and execute update steps if necessary.
It only supports `git`.

I wrote this tool to support my personal container infrastructe, which is completely managed via a single
git repository.
The target repository could look like this:

```sh
    .
    ├── config.yaml
    ├── README.md
    ├── service-1
    │   ├── config.yaml
    │   ├── docker-compose.yml
    │   └── README.md
    ├── service-2
    │   ├── config.yaml
    │   ├── docker-compose.yml
    │   └── README.md
    └── upstream-watch
```

There are two services, each in its own subfolder.
Each of these services holds a `README.md` (which is not interesting), a `docker-compose.yml` that defines
the containers and a `config.yaml`, which is the configuration file of `upstream-watch` for this specific service.

In case of an update to any of these files in a subfolder, `upstream-watch` will execute the pre- and post-hooks
defined in the corresponding `config.yaml`.

An example for a `config.yaml`:

```sh
    pre_update_commands: ["docker compose down", "docker compose pull"]
    post_update_commands: ["docker compose up -d"]
```

`upstream-watch` will stop all containers, pull updates from the registry and start them afterwards.
Of course, you can do almost anything in these hooks, depending on the needs of your service.

### Requirements on the server

* Pull access to upstream repository
* `git` installed
