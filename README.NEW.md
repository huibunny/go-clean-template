# go-clean-template

## build

```bash

# run build.sh and it will output goclean file.
$ ./build.sh

```

## run

```bash

$ ./goclean -consul localhost:8500 -name hello -listen :9090

```

## function

* remove http port from config file and use the one from cmd arguments.
* support consul(register/deregister/kv)

