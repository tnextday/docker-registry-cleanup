// Copyright © 2019 tnextday <fw2k4@163.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	imageSpecV1 "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/nokia/docker-registry-client/registry"

	"github.com/spf13/pflag"
)

const (
	defaultBaseUrl = "https://registry-1.docker.io/"
)

var (
	baseUrl       string
	username      string
	password      string
	repositories  []string
	tagsRegex     []string
	excludesRegex []string
	keepsN        int
	olderThen     string
	insecure      bool
	dryRun        bool
	verbose       bool
	printVersion  bool
	help          bool
	durationRegex = regexp.MustCompile(`(\d+)\s*([a-z]+)`)

	AppVersion = "dev"
	BuildTime  = ""
)

type ImageTag struct {
	Name    string
	Created *time.Time
}

func parserDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	ss := durationRegex.FindStringSubmatch(strings.ToLower(s))
	if len(ss) == 0 {
		return 0, errors.New("can't parser the duration string")
	}
	i, _ := strconv.Atoi(ss[1])
	switch ss[2][:1] {
	case "h":
		return time.Hour * time.Duration(i), nil
	case "d":
		return time.Hour * 24 * time.Duration(i), nil
	case "m":
		return time.Hour * 24 * 30 * time.Duration(i), nil
	default:
		return 0, fmt.Errorf("unsupport duration unit: %s", ss[2])
	}
}

func verboseLogf(format string, v ...interface{}) {
	if !verbose {
		return
	}
	fmt.Printf(format, v...)
}

func registryLog(format string, v ...interface{}) {
	if !verbose {
		return
	}
	verboseLogf(format+"\n", v...)
}

func matchRegexList(s string, list []*regexp.Regexp) bool {
	for _, r := range list {
		if r.MatchString(s) {
			return true
		}
	}
	return false
}

func getImageDetail(r *registry.Registry, repo, tag string) (*imageSpecV1.Image, error) {
	manifest, err := r.ManifestV2(repo, tag)
	if err != nil {
		return nil, err
	}
	reader, err := r.DownloadBlob(repo, manifest.Config.Digest)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	img := imageSpecV1.Image{}
	err = decoder.Decode(&img)
	return &img, err
}

func usage() {
	fmt.Printf(`Usage of docker-registry-cleanup

Options:
`)
	pflag.PrintDefaults()
	os.Exit(0)
}

func main() {
	pflag.ErrHelp = nil
	pflag.Usage = usage
	pflag.StringVarP(&username, "user", "u", "", "Registry login user name, environment: REGISTRY_USER")
	pflag.StringVarP(&password, "password", "p", "", "Registry login password, environment: REGISTRY_PASSWORD")
	pflag.StringVar(&baseUrl, "base-url", defaultBaseUrl, "Registry base url, environment: REGISTRY_BASE_URL")
	pflag.StringArrayVarP(&repositories, "repository", "r", []string{}, "[REQUIRED]Registry repository path list")
	pflag.StringArrayVarP(&tagsRegex, "tag", "t", []string{}, "Image tag regex list")
	pflag.StringArrayVarP(&excludesRegex, "exclude", "e", []string{}, "Exclude image tag regex list")
	pflag.IntVarP(&keepsN, "keep-n", "n", 10, "Keeps N latest matching tagsRegex for each registry repositories")
	pflag.StringVarP(&olderThen, "older-then", "o", "", "Tags to delete that are older than the given time, written in human readable form 1h, 1d, 1m")
	pflag.BoolVarP(&dryRun, "dry-run", "d", false, "Only print which images would be deleted")
	pflag.BoolVarP(&insecure, "insecure", "k", false, "Allow connections to SSL sites without certs")
	pflag.BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	pflag.BoolVarP(&printVersion, "version", "V", false, "Print version and exit")
	pflag.BoolVarP(&help, "help", "h", false, "Print help and exit")
	pflag.CommandLine.SortFlags = false
	pflag.Parse()

	if printVersion {
		fmt.Println("App Version:", AppVersion)
		fmt.Println("Go Version:", runtime.Version())
		fmt.Println("Build Time:", BuildTime)
		os.Exit(0)
	}

	if help {
		usage()
	}

	var _repositories []string
	for _, r := range repositories {
		if r != "" {
			_repositories = append(_repositories, r)
		}
	}
	repositories = _repositories

	if len(repositories) == 0 {
		fmt.Println("The Registry repository is required!")
		fmt.Printf("try '%s -h' for more information\n", os.Args[0])
		os.Exit(1)
	}

	if username == "" {
		if env := os.Getenv("REGISTRY_USER"); env != "" {
			username = env
		}
	}
	if password == "" {
		if env := os.Getenv("REGISTRY_PASSWORD"); env != "" {
			password = env
		}
	}
	if baseUrl == defaultBaseUrl {
		if env := os.Getenv("REGISTRY_BASE_URL"); env != "" {
			baseUrl = env
		}
	}

	olderThenDuration, err := parserDuration(olderThen)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	verboseLogf("Registry base url: %v\n", baseUrl)
	verboseLogf("Registry user: %s\n", username)
	verboseLogf("Registry password: **HIDDEN**\n")

	if olderThen != "" {
		verboseLogf("Older then duration: %v\n", olderThenDuration)
	}

	var (
		tagRegs     []*regexp.Regexp
		excludeRegs []*regexp.Regexp
	)

	for _, rs := range tagsRegex {
		if regx, err := regexp.Compile(rs); err == nil {
			tagRegs = append(tagRegs, regx)
		} else {
			fmt.Printf("Compile regex %s error: %v\n", rs, err)
			os.Exit(1)
		}
	}
	for _, rs := range excludesRegex {
		if regx, err := regexp.Compile(rs); err == nil {
			excludeRegs = append(excludeRegs, regx)
		} else {
			fmt.Printf("Compile regex %s error: %v\n", rs, err)
			os.Exit(1)
		}
	}

	registryOpt := registry.Options{
		Username: username,
		Password: password,
		Insecure: insecure,
		Logf:     registryLog,
	}
	hub, err := registry.NewCustom(baseUrl, registryOpt)

	if err != nil {
		fmt.Printf("New registry error: %v\n", err)
		os.Exit(1)
	}

	for _, repo := range repositories {
		fmt.Printf("Searching in %v\n", repo)
		tags, err := hub.Tags(repo)
		if err != nil {
			fmt.Printf("List registry repository tags failed, path: %s, error: %v\n", repo, err)
			os.Exit(1)
		}
		var matchedTags []*ImageTag

		for _, tag := range tags {
			if tag == "latest" {
				verboseLogf("Skipped the latest tag\n")
				continue
			}

			if len(excludeRegs) > 0 && matchRegexList(tag, excludeRegs) {
				verboseLogf("Skipped tag because of exclude rule: %v\n", tag)
				continue
			}
			if len(tagRegs) > 0 && !matchRegexList(tag, tagRegs) {
				verboseLogf("Skipped tag: %v\n", tag)
			} else {
				verboseLogf("Matched tag: %v\n", tag)
				matchedTags = append(matchedTags, &ImageTag{Name: tag})
			}
		}
		if keepsN > 0 && len(matchedTags) <= keepsN {
			fmt.Printf("Skip because of less mathced tags(%v) then keeps N(%v)\n", len(matchedTags), keepsN)
			continue
		}
		for _, tag := range matchedTags {
			img, err := getImageDetail(hub, repo, tag.Name)
			if err != nil {
				fmt.Printf("Get image tag detail failed, tag: %s, error: %v\n", tag.Name, err)
				os.Exit(1)
			}
			tag.Created = img.Created
		}
		sort.Slice(matchedTags, func(i, j int) bool {
			return matchedTags[i].Created.After(*matchedTags[j].Created)
		})
		verboseLogf("Found %v matched tags in %v\n", len(matchedTags), repo)

		if keepsN > 0 {
			verboseLogf("The latest %v matched tags will be keeps\n", keepsN)
			matchedTags = matchedTags[keepsN:]
		}
		var tagsToDelete []*ImageTag
		if olderThenDuration > 0 {
			now := time.Now()
			for _, t := range matchedTags {
				createDuration := now.Sub(*t.Created)
				if createDuration > olderThenDuration {
					tagsToDelete = append(tagsToDelete, t)
				} else {
					verboseLogf("Tag %v will be keep because of it's create only %v\n", t.Name, createDuration)
				}
			}
		} else {
			tagsToDelete = matchedTags
		}

		verboseLogf("%v tags in %v will be delete\n", len(tagsToDelete), repo)

		deletedCount := 0
		for _, t := range tagsToDelete {
			if dryRun {
				fmt.Printf("[Dry run]%s will be delete\n", t.Name)
				continue
			}
			fmt.Printf("Delete %s ", t.Name)
			digest, err := hub.ManifestDigest(repo, t.Name)
			if err == nil {
				err = hub.DeleteManifest(repo, digest)
			}
			if err == nil {
				deletedCount++
				fmt.Println("OK")
			} else {
				fmt.Println("error:", err)
				os.Exit(1)
			}
		}
		fmt.Printf("%v/%v tags have been deleted in %v\n", deletedCount, len(tagsToDelete), repo)
	}
}
