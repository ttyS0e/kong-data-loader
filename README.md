# Kong Data Loader

This program will load a (2.4 to 2.7) bootstrapped Kong schema with a huge amount of Consumers
and Rate Limiting Advanced plugins.

This is to test data loading in certain scenarios.

## Compiling

Execute `make build` to build using the local Golang SDK, or `make build-image` to construct a
container image using the configured container daemon.

## What Works

This program will:

* Create hundreds of thousands of Rate Limiting Advanced plugins, along with some consumers
* Attach a random count of between 1 and 20 RLA plugins to each created consumer
* Insert directly into the Kong schema

This program will NOT (yet, at least):
* Attach RBAC permissions - this is best used in a test environment, with RBAC disabled
* Update total counts in the database - you may see that this causes issues later...
* Trigger a data plane update after running - Kong in unaware of the database changes, you must execute the data plane sync manually (go into Manager UI and update something random)

## Usage

This program works with Kong Postgres datastores **only**.
It reads the Postgres configuration from the environment, the same way the Kong installation does:

* KONG_PG_HOST
* KONG_PG_DATABASE
* KONG_PG_USER
* KONG_PG_PASSWORD

There are additional flag arguments that you can pass to the program:

| Flag | Default | Description |
| ---- | ------- | ----------- |
| -redisHost | 127.0.0.1 | Hostname of the redis server that will handle the cluster rate limiting strategy counters |
| -redisport | 6379 | Port for the redis server |
| -redisdictionary | kong_rate_limiting_consumers | Override the default dictionary name for redis |
| -redisnamespace | L12Tt6QKCod1KLmT30RAz6GUj0KzVCp1 | Namespace key that will maange the rate limiting counters |
| -redissyncrate | 50 | Sync rate to apply to each rate limiting plugin |
| -redisusessl | false | Use SSL for redis connection |
| -workspace | default | NAME of the workspace that you want to create all the objects in |
| -createservices | false | Creates new services for dedicated use with this data loader, leaving false will use 20 random existing services |
| -servicecount | 20 | Number of services to attach the random consumers and plugins onto, if -createservices is true then this many will be created and used |
| -plugincount | 520000 | Total number of plugins (within 20 accuracy) to create |
