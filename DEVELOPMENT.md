# Development

Plugin is designed to be developed and run on Linux. It's highly advised that a Vagrant VM is used for development purposes.
Assuming Vagrant is installed a VM can be bootstrapped with:
```bash
$ vagrant up
 ==> default: Booting VM...
 ==> default: Waiting for machine to boot. This may take a few minutes...
 ...
 ==> default: Machine booted and ready!
 ==> default: Running provisioner: docker...
 ==> default: Running provisioner: shell...
 
$ vagrant ssh
```

Once connected to VM - navigate to `/vagrant` and attempt building a plugin as described below. Please note that you
will have to run both plugin and its tests as `root`.  

## Go

In order to work on the provider, [Go](http://www.golang.org) should be installed first (version 1.11+ is *required*).
[goenv](https://github.com/syndbg/goenv) and [gvm](https://github.com/moovweb/gvm) are great utilities that can help a
lot with that and simplify setup tremendously. 

This plugin relies on Go 1.11 modules and uses `go mod` to manage vendored dependencies.

Vendor-in all missing dependencies:
```bash
$ go mod vendor
```

Remove unused dependencies: 
```bash
$ go mod tidy
```

## Source Code

Source code can be retrieved either with `go get`

```bash
$ go get -u -d github.com/ashald/docker-volume-loopback
```
or with `git`
```bash
$ git clone git@github.com:ashald/docker-volume-loopback.git .
```

Because this plugin uses Go 1.11 modules it doesn't have to be checked out into `$GOPATH` but can be built anywhere
on file system. 


## Test

There is a pretty much extensive test suite that checks various aspects of plugin behavior. In order to runs tests 
there should be an instance of the plugin running on the host - test runner will find it by looking up the process name
`docker-volume-loopback` and entering its namespaces (so that test suite will behave in the same way regardless of 
how plugin is being run). The example below shows an excerpt from the test output - the entire suite has more than 30
tests and executes in less than 20 seconds.

```bash
$ make test
  ./tests/run.sh
  --- Executing 'test_disk_space_limits' ---
  testDefaultVolumeSize
  testCustomVolumeSize
  
  Ran 2 tests.
  
  OK
  
  real	0m2.548s
  user	0m0.223s
  sys	0m0.235s
  
  ...
  
   ---> ALL TESTS PASSED
```

## Build

In order to build the driver use \[GNU\]make:
```bash
$ make build
  GOOS=linux GOARCH=amd64 go build -o docker-volume-loopback

```

Alternatively it's possible to build a managed plugin but only on Linux:
```bash
$ make plugin
  docker build -t ashald/docker-volume-loopback-rootfs .
  ...
  Successfully tagged ashald/docker-volume-loopback-rootfs:latest
  ...
  docker create --name docker-volume-loopback-rootfs ashald/docker-volume-loopback-rootfs || true
  ...
  docker export docker-volume-loopback-rootfs | tar -x -C ./plugin/rootfs
  ...
  docker plugin create docker-volume-loopback ./plugin
  ...
  docker plugin enable docker-volume-loopback
  ...
``` 

## Versioning

This project follow [Semantic Versioning](https://semver.org/)

## Changelog

This project follows [keep a changelog](https://keepachangelog.com/en/1.0.0/) guidelines for changelog.

## Contributors

Please see [CONTRIBUTORS.md](./CONTRIBUTORS.md)
