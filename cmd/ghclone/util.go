package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	inputReader *bufio.Reader
	rateLimit   = &rateLimiting{}
)

type rateLimiting struct {
	Limit     int
	Remaining int
	Reset     int64
	lock      sync.Mutex
}

func (r *rateLimiting) GetRateLimitReset() string {
	if r.Reset == 0 {
		return "in approximately 1 hour from now"
	}
	return "on " + time.Unix(r.Reset, 0).Format(time.RFC822)
}

func (r *rateLimiting) Update(req *http.Response) {
	if _, hasLimit := req.Header["X-Ratelimit-Limit"]; !hasLimit {
		return
	}
	limit := req.Header["X-Ratelimit-Limit"][0]
	remaining := req.Header["X-Ratelimit-Remaining"][0]
	reset := req.Header["X-Ratelimit-Reset"][0]

	limitI32, err := strconv.ParseInt(limit, 10, 32)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error parsing rate limiting limit: %s\n", err.Error())
		return
	}
	remainingI32, err := strconv.ParseInt(remaining, 10, 32)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error parsing rate limiting remaining: %s\n", err.Error())
		return
	}
	resetI64, err := strconv.ParseInt(reset, 10, 64)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error parsing rate limiting reset: %s\n", err.Error())
		return
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	r.Limit = int(limitI32)
	r.Remaining = int(remainingI32)
	r.Reset = resetI64
	if verbose {
		fmt.Printf("Rate limit updated: %v of %v requests remaining, resetting %s\n", r.Remaining, r.Limit, r.GetRateLimitReset())
	}
}

func (r *rateLimiting) IsExceeded() bool {
	if r == nil {
		return false
	}
	return r.Remaining <= 0
}

func ReadYesNo() bool {
	if inputReader == nil {
		inputReader = bufio.NewReader(os.Stdin)
	}
	read, err := inputReader.ReadString('\n')
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error reading input: %s\n", err.Error())
		os.Exit(1)
	}
	read = strings.ToLower(strings.TrimSpace(read))

	if read == "" || read == "y" || read == "yes" || read == "true" {
		return true
	}
	return false
}

func ValidateToken() bool {
	const requestURL = "https://api.github.com/user"
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error creating request: %s\n", err.Error())
		os.Exit(1)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+personalAccessToken)

	if verbose {
		fmt.Printf("Using personal access token: %s\n", personalAccessToken)
		fmt.Printf("Request URL: %s\n", requestURL)
	}

	resp, err := http.DefaultClient.Do(req) //nolint:bodyclose
	rateLimit.Update(resp)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error making request to GitHub API: %v\n", err)
		os.Exit(1)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode != 200 {
		return false
	}

	var login struct {
		Login string `json:"login"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&login); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// If verbose, then this has already been printed, but also don't print if quiet is set
	if !verbose && !quiet {
		fmt.Printf("Authenticated as @%s! Rate limit has %v of %v requests remaining, resetting %s\n",
			login.Login, rateLimit.Remaining, rateLimit.Limit, rateLimit.GetRateLimitReset())
	}
	return true
}
