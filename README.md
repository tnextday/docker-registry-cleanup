docker-registry-cleanup
=======================

[简体中文](README_ZH.md)

docker-registry-cleanup is a tool for batch cleaning gitlab image repositories. It supports multiple ways to match the images that need to be cleaned up, and is easy to integrate into gitlab CI.

# Note

Cannot be used in the gitlab registry, reference:

[#37810 - GitLab registry API - UNAUTHORIZED issue](https://gitlab.com/gitlab-org/gitlab-ce/issues/37810)


# Usage

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

Clean up the image in the `foo/bar` project for more than one month except the version number, but at least keep the last 5

```
export REGISTRY_USER=<USER>
export REGISTRY_PASSWORD=<PASSWORD>
docker-registry-cleanup -r foo/bar --older-then 1month --keep-n 5 --exclude '^v?[0-9.]+$'
```

Please refer to [`.gitlab-ci.yml`](.gitlab-ci.yml) for use in CI.