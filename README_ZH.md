docker-registry-cleanup
=======================

docker-registry-cleanup 是一个批量清理 gitlab 镜像仓库的工具，支持多种方式匹配需要清理的镜像，方便集成到 gitlab CI 中。


# 重要提示

gitlab registry 无法使用，参考

[#37810 - GitLab registry API - UNAUTHORIZED issue](https://gitlab.com/gitlab-org/gitlab-ce/issues/37810)
[#40096 - pipeline user $CI_REGISTRY_USER lacks permission to delete its own images](https://gitlab.com/gitlab-org/gitlab-ce/issues/40096)

# 使用方法


```
Usage of docker-registry-cleanup

Options:
  -u, --user string              Registry login user name, environment: REGISTRY_USER
  -p, --password string          Registry login password, environment: REGISTRY_PASSWORD
      --base-url string          Registry base url, environment: REGISTRY_BASE_URL (default "https://registry-1.docker.io/")
  -r, --repository stringArray   [REQUIRED]Registry repository path list
  -t, --tag stringArray          Image tag regex list
  -e, --exclude stringArray      Exclude image tag regex list
  -n, --keep-n int               Keeps N latest matching tagsRegex for each registry repositories (default 10)
  -o, --older-then string        Tags to delete that are older than the given time, written in human readable form 1h, 1d, 1m
  -d, --dry-run                  Only print which images would be deleted
  -k, --insecure                 Allow connections to SSL sites without certs
  -v, --verbose                  Verbose output
  -V, --version                  Print version and exit
  -h, --help                     Print help and exit

```

# Example

清理`foo/bar`工程中一个月以上除版本号以外的镜像，但是至少会保留最近的 5 个

```
export REGISTRY_USER=<USER>
export REGISTRY_PASSWORD=<PASSWORD>
docker-registry-cleanup -r foo/bar --older-then 1month --keep-n 5 --exclude '^v?[0-9.]+$'
```

在 CI 中使用请参考 [`.gitlab-ci.yml`](.gitlab-ci.yml)