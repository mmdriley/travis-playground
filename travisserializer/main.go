package main

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

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

// running push builds, in this repository, on this branch
// sorted by ID descending
func runningBuilds() []Build {
	var builds Builds

	vs := url.Values{}
	vs.Add("build.state", "created,queued,received,started")
	vs.Add("build.event_type", "push")
	vs.Add("build.branch", travisBranch)
	vs.Add("sort_by", "id:desc")

	path := mustParseURL(fmt.Sprintf("/repo/%v/builds?%v", url.PathEscape(travisRepoSlug), vs.Encode()))

	resp, _, errs := gorequest.New().
		Get(travisEndpoint.ResolveReference(path).String()).
		Set("Travis-API-Version", "3").
		Set("Authorization", "token "+travisToken).
		EndStruct(&builds)

	if errs != nil || resp.StatusCode != http.StatusOK {
		panic("can't list running builds")
	}

	return builds.Builds
}

func isOldestRunningBuild() bool {
	builds := runningBuilds()
	foundThisBuild := false

	// The list of builds is ordered from newest to oldest.
	for _, build := range builds {
		if foundThisBuild {
			// There was a running build older than this one.
			fmt.Printf("Found older build %v (%v) in state %v\n", build.Number, build.ID, build.State)
			return false
		}

		if strconv.Itoa(build.ID) == travisBuildID {
			foundThisBuild = true
		}
	}

	// If we found this build, it was last in the list.

	if !foundThisBuild {
		// Sanity check -- we should always see the current build in the list.
		panic(fmt.Errorf("couldn't find this build, %v", travisBuildID))
	}

	return true
}

func cancelThisBuild() {
	fmt.Printf("Cancelling this build...\n")

	path := mustParseURL(fmt.Sprintf("/build/%v/cancel", travisBuildID))

	resp, _, errs := gorequest.New().
		Post(travisEndpoint.ResolveReference(path).String()).
		Set("Travis-API-Version", "3").
		Set("Authorization", "token "+travisToken).
		End()

	if errs != nil || resp.StatusCode != http.StatusOK {
		panic("couldn't cancel build")
	}

	// Wait for the build to be cancelled. Travis' build timeout is 2 hours.
	time.Sleep(3 * time.Hour)
}

func restartNewestCancelledBuild() {
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %v {start|finish}\n", os.Args[0])
		os.Exit(1)
	}

	command := os.Args[1]

	if command == "start" {
		if !isOldestRunningBuild() {
			cancelThisBuild()
		}
	} else if command == "finish" {
		restartNewestCancelledBuild()
	} else {
		fmt.Fprintf(os.Stderr, "ERROR: Invalid command %v\n", command)
		os.Exit(1)
	}
}
