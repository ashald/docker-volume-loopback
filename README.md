# Docker Volume Loopback

## Overview

### Sparse

|FS   | Sparse        | Regular    |
| --- | ------------- | ---------- |
|xfs  | 0%/1%         | 100%/100%  |
|ext4 | 0%/3%         | 100%/3%    |

## Installation

### Automatic

### Manual

## Usage

## Known Issues and Limitations

### Compatibility

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


## Development
Go 1.11 modules

```bash
go mod vendor
go mod tidy
```

## License

This is free and unencumbered software released into the public domain. See [LICENSE](./LICENSE)

[Loop devices are notorious for their "bad" performance]: https://serverfault.com/questions/166748/performance-of-loopback-filesystems
[Faster and leaner loop device with Direct I/O and Asynchronous I/O support]: https://kernelnewbies.org/Linux_4.4#Faster_and_leaner_loop_device_with_Direct_I.2FO_and_Asynchronous_I.2FO_support
[circumvents the "double buffering" issue]: https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/?id=bc07c10a3603a5ab3ef01ba42b3d41f9ac63d1b6