package main

import (
	"fmt"
	"net/url"
	"os"

	"github.com/parnurzeal/gorequest"
)

func requireEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		fmt.Fprintf(os.Stderr, "ERROR: %v is not set\n", key)
		os.Exit(1)
	}

	return value
}

func mustParseURL(v string) *url.URL {
	url, err := url.Parse(v)
	if err != nil {
		panic(fmt.Errorf("can't parse %v as URL", v))
	}
	return url
}

var (
	travisEndpoint = mustParseURL(requireEnv("TRAVIS_ENDPOINT"))
	travisToken    = requireEnv("TRAVIS_TOKEN")

	travisBuildID     = requireEnv("TRAVIS_BUILD_ID")
	travisBuildNumber = requireEnv("TRAVIS_BUILD_NUMBER")

	travisEventType = requireEnv("TRAVIS_EVENT_TYPE")

	travisBranch   = requireEnv("TRAVIS_BRANCH")
	travisRepoSlug = requireEnv("TRAVIS_REPO_SLUG")
)

type Build struct {
	ID int

	Number string
	State  string
}

type Builds struct {
	Builds []Build
}

func runningBuilds() []Build {
	var builds Builds

	vs := url.Values{}
	vs.Add("build.state", "created,queued,received,started")
	vs.Add("build.event_type", "push")
	vs.Add("build.branch", travisBranch)
	vs.Add("sort_by", "id:desc")

	path := mustParseURL(fmt.Sprintf("/repos/%v/builds?%v", url.PathEscape(travisRepoSlug), vs.Encode()))

	resp, _, errs := gorequest.New().
		Get(travisEndpoint.ResolveReference(path).String()).
		Set("Travis-API-Version", "3").
		Set("Authorization", "token "+travisToken).
		EndStruct(&builds)

	if errs != nil || resp.StatusCode != 200 {
		panic("can't list running builds")
	}

	return builds.Builds
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %v {start|finish}\n", os.Args[0])
		os.Exit(1)
	}

	command := os.Args[1]

	if command == "start" {
		fmt.Printf("%+v\n", runningBuilds())
	} else if command == "finish" {

	} else {
		fmt.Fprintf(os.Stderr, "ERROR: Invalid command %v\n", command)
		os.Exit(1)
	}
}
