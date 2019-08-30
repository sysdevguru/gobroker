# Integration Test

Integration test runs in Google Cloud Build and please refer to cloudbuild.yml.
This directory contains the test code, as well as some docker-compose to
setup static cluster to interact with. For the cluster setup, see [docker-compose.yml](./docker-compose.yml) file in this directory.

In order to run the cluster,

```sh
$ docker-compose up -d
```

and check the status

```sh
$ docker-compose ps
```

also check logs for each container

```sh
$ docker-compose logs gobrokersvc
```

## Cleanup

Some state is left on disk and you may start over by

```sh
$ make clean
```
