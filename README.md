# Docker Volume Loopback

## Overview

The `docker-volume-loopback` is a [Docker volume driver] that allows creating volumes that are fixed in size.
Fixed size volumes are valuable because they can be used to "reserve" disk space for a particular container and make sure
it will be available later when needed or limit the disk space available to a container so that it won't abuse the host
and affect other processes running there.

This plugin makes use of Linux [loop devices] that can be used to present a file on a filesystem as a regular block device.
Using loop devices allow for great flexibility as there are next to 0 prerequisites and they can be used on arbitrary
Linux hosts. Volumes themselves are stored as regular files and are very easy to manage or even can be moved between
different hosts. All of that being said, there are some potential performance and durability implications one may need
to take into account when using loop devices and this particular plugin - they are discussed in more detail in the
["Known Issues and Limitations"](#known-issues-and-limitations) section below.

Generally speaking, an [LVM-based docker volume driver] is a more reliable and efficient alternative in case one can use
LVM or setting it up is feasible (e.g., may not be an option on managed hosts).


## Demo

<div align="center">
  <img src="./docs/demo.svg">
</div>


## Features

### Regular & Sparse volumes

The plugin supports `ext4` and `xfs` filesystems that together with `sparse` option (see details in ["Usage"](#usage)
section) can be used to achieve different levels of disk space reservation guarantees.

When `sparse` option is enabled (disabled by default) the driver would create a [sparse file] to back the volume. This
means that even though file will appear to be of a given size, it's not going to be taking that much of disk space.
Instead, its physical size is going to be equal to its actual usage but won't be able to grow beyond the certain size limit.
This can be useful when there is a need to limit disk space available but it might not be necessary to ensure it actually
will be available. As a side effect one can create volumes larger than there is actually disk space available for the
purpose of "over-subscription" when volumes are rarely used to their full size.

When regular files are used to back volumes the driver will first attempt to allocate as much disk space as was requested
ensure volume can fit onto disk. After that volume is formatted with a filesystem of choice and behavior differs. When
`xfs` is used the volume data file remains a "regular file" and volume still uses as much disk space as was requested
providing a guarantee that the entire disk space is going to be available. In case `ext4` is used, the data file is being
converted to a sparse file.
In other words, both in both cases driver verifies that there is enough disk space available at the moment of creation
but then relaxes the reservation (in case of `ext4`) or keeps it place (in case of `xfs`) depending on filesystem used.  

The table below shows actual data file size on disk before and after formatting depending on whether `sparse` option is 
used and how volume is formatted.

| FS   | Sparse        | Regular     |
| ---- | ------------- | ----------- |
| xfs  | 0% / 1%       | 100% / 100% |
| ext4 | 0% / 3%       | 100% / 3%   |

### Extensive Logging

The plugin is designed to be as reliable as possible and its code is written in way that is slightly more explicit than
one could be used to just to make sure all exceptional situations are handled. Although, we are all mere human beings
cannot predict everything and things may fail. When that happens its crucial to be able to reproduce the issue and get
as many details as possible so that one can pin point the source of failure and come up with a solution. For that purpose
plugin is built around a "tracing" concept where every call of any significant interest is being accompanied by a so-called
trace identifier that one can use to get a complete overview of what the plugin does. This also can be used for audit
purposes or to understand how exactly plugin works. One can chose between 5 log levels - from `error` to `trace` to get
different levels of details.

There are log samples available for a somehow corner case scenario (so that we can get some warnings!) for a volume
creation operation when the plugin has to store its data files on an older filesystem like `ext3` that does not support
`fallocate` and therefore it falls back to a slower `dd`:

* 0 - [ERROR](./docs/example.0_error.log)
* 1 - [WARNING](./docs/example.1_warning.log)
* 2 - [INFO](./docs/example.2_info.log)
* 3 - [DEBUG](./docs/example.3_debug.log)
* 4 - [TRACE](./docs/example.4_trace.log)

## Installation

### Automatic

Plugin is compatible with [Docker's managed plugin system] and therefore can be installed as simple as:
```bash
$ docker plugin install ashald/docker-volume-loopback
  Plugin "ashald/docker-volume-loopback" is requesting the following privileges:
   - mount: [/dev]
   - mount: [/]
   - allow-all-devices: [true]
   - capabilities: [CAP_SYS_ADMIN]
  Do you grant the above permissions? [y/N] y
  latest: Pulling from ashald/docker-volume-loopback
  ...
  Installed plugin ashald/docker-volume-loopback
```

### Manual

Plugin can be run "manually" - just as an executable on the same host as Docker daemon. Plugin default options are such
that it should be possible to run it with minimum or no configuration. Plugin itself does not talk to Docker but it's the
other way around - [Docker expects to find plugin's socket in one of few pre-defined locations].

## Configuration

Regardless of the way plugin is installed certain aspects of its behavior can be controlled. Both command line arguments
and environment variables are supported.

| Env Var         | Argument          | Default                                             | Comment                                               |
| --------------- | ----------------- | --------------------------------------------------- | ----------------------------------------------------- |
| `DATA_DIR`      | `--data-dir`      | `/var/lib/docker-volume-loopback`                   | Persistent dir to store volumes' data                 |
| `MOUNT_DIR`     | `--mount-dir`     | `/mnt`                                              | Dir to mount volumes so Docker can access them        |
| `STATE_DIR`     | `--state-dir`     | `/run/docker-volume-loopback`                       | Volatile dir to keep track of currently used volumes  |
| `LOG_LEVEL`     | `--log-level`     | `2`                                                 | 0-4 for error/warning/info/debug/trace                |
| `LOG_FORMAT`    | `--log-format`    | `nice`                                              | `json` / `text` / `nice`                              |
| `SOCKET`        | `--socket`        | `/run/docker/plugins/docker-volume-loopback.sock`   | Name of the socket determines plugin name             |
| `DEFAULT_SIZE`  | `--default-size`  | `1GiB`                                              |                                                       |

When Docker's managed plugin system configuration can be adjusted via environment variables with the exception for
`SOCKET` and `MOUNT_DIR` that have to be set to specific values. As for `STATE_DIR` and `DATA_DIR` they adjusted to 
`/srv/run/docker-volume-loopback` and `/srv/var/lib/docker-volume-loopback` respectively - host's file system can be
accessed via `/srv` prefix. 

## Usage

### Examples

Please note that examples below assume that the driver is run in manual mode and therefore is available as `-d docker-volume-loopback`.
In managed mode (automatic installation) that will become `-d ashald/docker-volume-loopback`.

Create a regular volume using default filesystem (`xfs`) and size (`1Gib`):
```bash
$ docker volume create -d docker-volume-loopback foobar 
```

Create a sparse volume using custom filesystem (`ext4`) and size (`100Mib`):
```bash
$ docker volume create -d docker-volume-loopback foobar -o sparse=true -o fs=ext4 -o size=100MiB 
```

Create a regular volume using default filesystem (`xfs`) and size (`1Gib`) but adjust volume's root ownership and permissions:
```bash
$ docker volume create -d docker-volume-loopback foobar -o uid=1000 -o gid=2000 -o mode=777 
```

### Options

| Option            | Default                                       | Comment                                                               |
| ----------------- |---------------------------------------------- | --------------------------------------------------------------------- |
| `size`            | Set by `DEFAULT_SIZE` driver config option    | Size in bytes or with a unit suffix K/M/T/P and Ki/Mi/Ti/Pi           |
| `sparse`          | `false`                                       | Whether to reserve disk space or just set a limit: `true` or `false`  |
| `fs`              | `xfs`                                         | Filesystem to format volume with: `xfs` or `ext4`                     |
| `uid`             | `-1`                                          | UID to set as owner of the volume's root, `-1` means do not adjust    |
| `gid`             | `-1`                                          | GID to set as owner of the volume's root, `-1` means do not adjust    |
| `mode`            | `0`                                           | Mode to set for volume's root, octal with up to 4 positions           |

## Known Issues and Limitations

### Platforms

Only designed to be working on Linux.
Potentially may work on macOS given that it uses an Alpine VM behind the scenes but this has not been tested.

### Minimum Size

The minimum allowed volume size is 20 MB (20,000,000 bytes) and is necessary to fit any of supported filesystems.
If smaller volume is needed it's advised to consider using [Docker's native `tmpfs` volume driver] that also supports
limiting disk space available.

### Performance

[Loop devices are notorious for their "bad" performance]. While it's not arguable that they incur and some overhead both
in terms of CPU and memory it, it always should be evaluated in a context of a concrete use case. Generally speaking,
if one does not constantly `fsync` after each write to a filesystem based on a loopback device \[and just let kernel do its job\]
the performance seems to be comparable to regular block devices. Although, it must be noted that in such cases write
operations are susceptible to the "double caching" issue where data are cached first while being written to the loopback
device backed filesystem and then data cached again while changes are bing finally committed to the backing block device.
This means that there may be a delay (on average, up to a 2 x 30 seconds = 1 minute) before writes will become durable
by being committed to the backing block device. Also, cache buffers are usually freed and committed more often when
system runs low on memory which means that usage of loopback devices is unlikely to degrade performance of the system as
a whole but system running low on memory is likely to have loopback devices performing worse. For the record, cache memory
is not being counted against `memory.max_usage_in_bytes` cgroup controller and therefore is ignored by Docker.

Last but not least, the release of Linux kernel `v4.4` includes "[Faster and leaner loop device with Direct I/O and Asynchronous I/O support]"
which [circumvents the "double buffering" issue].

In terms of CPU, while no comprehensive benchmarks have been done during development of the plugin, the overhead seem to
be negligible.

### Filesystem Compatibility

Each volume entirely stored on disk as a single file. The filesystem those files are stored on makes a difference in
some cases. When using a `sparse` volume type its data file is being created with a call to `truncate` which works
instantaneously. Otherwise, the driver would attempt using `fallocate` to create a "regular" file that would actually
claim the disk space and works as fast as `truncate`. Unfortunately `fallocate` is only known to work with newer
filesystems such as `ext4` and `xfs` and therefore will fail if underlying filesystem is an older one (e.g., `ext3`).
In this case plugin will detect a failure and fall back to using `dd` which is universally compatible but significantly
slower: depending on backing block device its write throughput may very between 10s of MiB/s to several GiB/s.

### Kernel Compatibility

When dealing with volumes based on `XFS` filesystem the driver depends on `mkfs.xfs` from `xfsprogs` package.
Starting from `v3.2.3` (released on 2015-06-10) of the package newly created `XfS` filesystems default to use of a newer
metadata format that is only supported by Linux kernel `v3.16` and above.

This manifests as a failure during an attempt to mount a volume - the filesystem will be initialized properly upon
volume creation but kernel will not be able to mount it.  

This is unlikely to be an issue if _manual_ installation mode is used as the version of `xfsprogs` available via Linux
distribution-specific package manager is likely to be compatible with the version of Linux kernel required.
When _automatic_ installation is used though the plugin is \[effectively\] distributed as a container image based on
Alpine Linux which \[at the moment of writing\] uses at least `v4.19.0` of `xfsprogs` package. This means that in case
the plugin will be installed on a system based on older version of kernel it won't be able to use `XFS` filesystem.

There is a workaround available that can be used to circumvent the issue: use an extra `-m crc=0` parameter when calling
`mkfs.xfs` which will force use of older version of metadata compatible with older versions of kernel. Unfortunately this
cannot be a default behavior as this flag was added only to `xfsprogs` starting from `v3.2.0` and therefore is unlikely
to be available on similar systems based on outdated versions of kernel \[in case _manual_ installation mode is used\].


While the workaround is trivial in essence it is trickier to implement as would require fir amount of code to run
dynamic checks for versions of kernel and `mkfs.xfs`. Hence a conscious decision to not implement this behavior initially.
In case there will be interest if support for older systems it can be easily added and should be requested via a GitHub
issue.

Below is the list of distributions known to be affected by this:

| Distribution | Version               | Release Date | End of Life Date |
| ------------ | --------------------- | ------------ | ---------------- |
| Ubuntu       | 14.04 LTS Trusty Tahr | 2014-04-17   | 2019-04-01       |


Entries are going to be removed from the table upon reaching EOL.
May the list above be exhausted this notice will be dropped altogether.


## Development

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

### Go

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

### Source Code

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


### Test

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

### Build

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

### Versioning

This project follow [Semantic Versioning](https://semver.org/)

### Changelog

This project follows [keep a changelog](https://keepachangelog.com/en/1.0.0/) guidelines for changelog.

### Contributors

Please see [CONTRIBUTORS.md](./CONTRIBUTORS.md)

## License

See [LICENSE.txt](./LICENSE.txt)

[Docker volume driver]: https://docs.docker.com/storage/volumes/
[loop devices]: https://en.wikipedia.org/wiki/Loop_device
[LVM-based docker volume driver]: https://github.com/containers/docker-lvm-plugin
[sparse file]: https://en.wikipedia.org/wiki/Sparse_file
[Docker's managed plugin system]: https://docs.docker.com/engine/extend/
[Docker expects to find plugin's socket in one of few pre-defined locations]: https://docs.docker.com/engine/extend/plugin_api/#plugin-discovery
[Docker's native `tmpfs` volume driver]: https://docs.docker.com/storage/tmpfs/
[Loop devices are notorious for their "bad" performance]: https://serverfault.com/questions/166748/performance-of-loopback-filesystems
[Faster and leaner loop device with Direct I/O and Asynchronous I/O support]: https://kernelnewbies.org/Linux_4.4#Faster_and_leaner_loop_device_with_Direct_I.2FO_and_Asynchronous_I.2FO_support
[circumvents the "double buffering" issue]: https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/?id=bc07c10a3603a5ab3ef01ba42b3d41f9ac63d1b6
