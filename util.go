package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var inputReader *bufio.Reader

func ReadYesNo() bool {
	if inputReader == nil {
		inputReader = bufio.NewReader(os.Stdin)
	}
	read, err := inputReader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %s\n", err.Error())
		os.Exit(1)
	}
	read = strings.ToLower(strings.TrimSpace(read))

	if read == "" || read == "y" || read == "yes" || read == "true" {
		return true
	}
	return false
}

func GetRateLimitReset(resp *http.Response) string {
	i, err := strconv.ParseInt(resp.Header.Get("X-Ratelimit-Reset"), 10, 64)
	if err != nil {
		return "in approximately 1 hour from now"
	}

	return "on " + time.Unix(i, 0).Format(time.RFC822)
}

func RequestAPI(method, url string) *http.Response {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error creating request: %s\n", err.Error())
		os.Exit(1)
	}

	if personalAccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+personalAccessToken)
		if verbose {
			fmt.Printf("Using personal access token: %s\n", personalAccessToken)
		}
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error making request to GitHub API %s\n", err.Error())
		os.Exit(1)
	}

	return resp
}

func CheckToken(token string) bool {
	if token == "" {
		fmt.Println("No token provided. You can generate one at github.com/settings/tokens")
		return false
	}

	var login struct {
		Login string `json:"login"`
	}

	resp := RequestAPI("GET", "https://api.github.com/user")
	parseJson(resp, &login)

	if !quieterQuiet {
		fmt.Printf("Authenticated as @%s! Rate Limit: %s of %s requests remaining.\n", login.Login, resp.Header.Get("X-Ratelimit-Remaining"), resp.Header.Get("X-Ratelimit-Limit"))
	}
	return true
}

func parseJson(resp *http.Response, result any) {
	err := json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error marshaling JSON: %s\n", err.Error())
		os.Exit(1)
	}
}
