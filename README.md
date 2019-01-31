# Performance
https://serverfault.com/questions/166748/performance-of-loopback-filesystems
https://kernelnewbies.org/Linux_4.4#Faster_and_leaner_loop_device_with_Direct_I.2FO_and_Asynchronous_I.2FO_support

https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/?id=bc07c10a3603a5ab3ef01ba42b3d41f9ac63d1b6


### Sparse

|FS   | Sparse        | Regular    |
| --- | ------------- | ---------- |
|xfs  | 0%/1%         | 100%/100%  |
|ext4 | 0%/3%         | 100%/3%    |


## Development
Go 1.11
```bash
go mod vendor
go mod tidy
```

## Test
TODO:
- check if xfs bug manifests after fallocate
- check if xfs bug manifests via docker containers
- check if '-o nouuid' works with ext4 
- add '-o nouuid'
- add test that volume can be remounted after adding some data

create sparse  ext4 volume of 1GiB on a 100 MiB disk - success
create regular ext4 volume of 1GiB on a 100 Mib disk - fail  

create sparse  xfs volume of 1GiB on a 100 MiB disk - success
create regular xfs volume of 1GiB on a 100 MiB disk - fail

on ext4/xfs vs ext3

not enough space - data file should be cleaned up

uid/gid - default & custom

mode - default & custom

inspect - should export status fields

