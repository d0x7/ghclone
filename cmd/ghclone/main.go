package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"xiam.li/meta"
)

type Repository struct {
	Name     string `json:"name"`
	CloneURL string `json:"clone_url"`
}

var (
	account             string
	accountType         string
	userType            bool
	outputDir           string
	startPage           int
	perPage             int
	verbose             bool
	quiet               bool
	quieterQuiet        bool
	noPrompt            bool
	all                 bool
	dryRun              bool
	personalAccessToken string
	version             bool
)

func main() {
	flag.StringVar(&accountType, "type", "org", "What type of GitHub account to clone, either \"org\" or \"user\"")
	flag.StringVar(&accountType, "t", "org", "What type of GitHub account to clone, either \"org\" or \"user\"")
	flag.BoolVar(&userType, "user", false, "Alias to --type user")
	flag.BoolVar(&userType, "u", false, "Alias to --type user")
	flag.StringVar(&outputDir, "output", "AccountName", "The directory to clone the repos to")
	flag.StringVar(&outputDir, "o", "AccountName", "The directory to clone the repos to")
	flag.IntVar(&startPage, "page", 1, "The page number to start cloning from")
	flag.IntVar(&startPage, "p", 1, "The page number to start cloning from")
	// flag.IntVar(&perPage, "per-page", 100, "The number of entities to clone per page")
	flag.IntVar(&perPage, "pp", 100, "The number of entities to clone per page")
	flag.BoolVar(&verbose, "verbose", false, "Whether to print verbose output")
	flag.BoolVar(&verbose, "V", false, "Whether to print verbose output")
	flag.BoolVar(&quiet, "quiet", false, "Whether to suppress unnecessary, informational, output")
	flag.BoolVar(&quiet, "q", false, "Whether to suppress unnecessary, informational, output")
	flag.BoolVar(&quieterQuiet, "quieter-quiet", false, "Whether to suppress all output. Errors will still be printed")
	flag.BoolVar(&quieterQuiet, "qq", false, "Whether to suppress all output. Errors will still be printed")
	flag.BoolVar(&noPrompt, "no-prompt", false, "With this flag enabled, there are no interactive prompts")
	flag.BoolVar(&noPrompt, "np", false, "With this flag enabled, there are no interactive prompts")
	flag.BoolVar(&all, "all", false, "With this flag enabled, all pages will be cloned without prompting")
	flag.BoolVar(&all, "a", false, "With this flag enabled, all pages will be cloned without prompting")
	flag.BoolVar(&dryRun, "dry-run", false, "With this flag enabled, no repos will be cloned")
	flag.BoolVar(&dryRun, "dry", false, "With this flag enabled, no repos will be cloned")
	flag.BoolVar(&dryRun, "dr", false, "With this flag enabled, no repos will be cloned")
	flag.StringVar(&personalAccessToken, "token", "", "GitHub Fine-grained Token to use for authentication\nNeeds Repository->Metadata->Read-Only and Repository->Contents->Read-Only permissions (latter not when dry-run is enabled)")
	flag.StringVar(&personalAccessToken, "pat", "", "GitHub Personal Access Token to use for authentication\nNeeds full repo access permissions. Use of fine-grained tokens are recommended over this")
	flag.BoolVar(&version, "version", false, "Print the version and exit")
	flag.BoolVar(&version, "v", false, "Print the version and exit")

	checkVars()
	clone(startPage)
}

func checkVars() {
	flag.Parse()

	if version {
		fmt.Printf("%s %s (%s), built on %s\n", filepath.Base(os.Args[0]), meta.VersionOr("v0.0.0-dev+dirty"), meta.ShortSHAOr(strings.Repeat("x", 40)), meta.DateOr(time.Now()))
		os.Exit(0)
	}

	if quieterQuiet {
		quiet = true
	}

	if accountType != "org" && accountType != "user" {
		_, _ = fmt.Fprint(os.Stderr, "Invalid account type provided")
		os.Exit(1)
	} else if accountType == "org" && !userType {
		accountType = "orgs"
	} else if accountType == "user" || userType {
		accountType = "users"
	}

	if startPage < 1 {
		_, _ = fmt.Fprint(os.Stderr, "Invalid page number provided")
		os.Exit(1)
	}

	if perPage < 1 {
		_, _ = fmt.Fprint(os.Stderr, "Invalid items number provided")
		os.Exit(1)
	}

	if account = flag.Arg(0); account == "" {
		fmt.Printf("Usage: %s [options] <account name>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	if outputDir == "AccountName" {
		outputDir = account
	}

	if personalAccessToken != "" && !ValidateToken() {
		_, _ = fmt.Fprint(os.Stderr, "Couldn't authenticate with GitHub API. Please check your token.\n")
		os.Exit(1)
	} else if personalAccessToken == "" && !quieterQuiet {
		fmt.Println("No personal access token provided. This may result in rate limiting. Check out token/pat flag using --help to find out more.")
	}
}

func printConfig() {
	if quiet {
		return
	}

	// Print current configuration
	var config strings.Builder
	if dryRun {
		config.WriteString("Dry run cloning")
	} else {
		config.WriteString("Cloning")
	}
	if all {
		config.WriteString(" all repos of ")
	} else {
		config.WriteString(" ")
	}

	config.WriteString(accountType)
	config.WriteString("/")
	config.WriteString(account)

	config.WriteString(" into ")
	if outputDir == "." {
		config.WriteString("the current directory.")
	} else {
		config.WriteString("\"")
		config.WriteString(outputDir)
		config.WriteString("\".")
	}

	if !all {
		config.WriteString(" Starting at page ")
		config.WriteString(fmt.Sprintf("%d", startPage))
		config.WriteString(" with ")
		config.WriteString(fmt.Sprintf("%d", perPage))
		config.WriteString(" repositories per page and prompts ")
		if noPrompt {
			config.WriteString("disabled.")
		} else {
			config.WriteString("enabled.")
		}
	}

	fmt.Println(config.String())
}

var (
	hasRetried  = false
	firstRun    = true
	totalCloned int
)

//goland:noinspection GoUnhandledErrorResult
func clone(page int) {
	if firstRun {
		printConfig()
	}
	req := buildRequest(page)
	resp, err := http.DefaultClient.Do(req)
	rateLimit.Update(resp)
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("Error closing response body: %s\n", err.Error())
			os.Exit(1)
		}
	}(resp.Body)
	handleError(resp, err, page)

	var repos []Repository
	err = json.NewDecoder(resp.Body).Decode(&repos)
	if err != nil {
		fmt.Printf("Error decoding response body: %s\n", err.Error())
		os.Exit(1)
	}
	if !all {
		fmt.Printf("Cloning %d repositories from %s starting from page %d…\n", len(repos), account, page)
	}
	createOutputDir()

	var waitGroup sync.WaitGroup
	for _, repo := range repos {
		if verbose {
			fmt.Printf("Cloning %s...\n", repo.CloneURL)
		}
		waitGroup.Add(1)
		go func(repo Repository) {
			defer waitGroup.Done()
			cloneRepo(repo)
			totalCloned++
		}(repo)
	}
	waitGroup.Wait()
	firstRun = false

	handleNextPage(len(repos), page)
	var allString string
	if all {
		allString = "all "
	}
	if !quiet {
		fmt.Printf("Done cloning %s%d repos from %s. Bye bye!\n", allString, totalCloned, account)
	}
	os.Exit(0)
}

func buildRequest(page int) *http.Request {
	requestURL := fmt.Sprintf("https://api.github.com/%s/%s/repos?page=%d&per_page=%d", accountType, account, page, perPage)
	if accountType == "orgs" && personalAccessToken != "" {
		fmt.Println("type is all including private repos")
		requestURL += "&type=all"
	}
	req, _ := http.NewRequest("GET", requestURL, nil)
	if personalAccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+personalAccessToken)
	}
	req.Header.Set("User-Agent", "d0x7/github-clone")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if verbose {
		fmt.Printf("Request URL: %s\n", requestURL)
	}
	return req
}

func handleError(resp *http.Response, err error, page int) {
	if err == nil && resp.StatusCode == 200 {
		return
	}

	// Check for rate limiting
	if resp.StatusCode == 403 && rateLimit.IsExceeded() {
		_, _ = fmt.Fprintf(os.Stderr, "Rate limit exceeded. Please try again %s.\n", rateLimit.GetRateLimitReset())
		if personalAccessToken == "" {
			_, _ = fmt.Fprint(os.Stderr, "Authenticated users have a higher rate limit (5000 instead of 60 requests per hour). Please consider using a token.")
		}

		os.Exit(2)
	}

	if resp.StatusCode != 404 {
		_, _ = fmt.Fprintf(os.Stderr, "Error making request to GitHub API: %v\n", err)
		os.Exit(1)
	}

	// Given account could not be found. Either try or exit depending on the noPrompt flag

	if noPrompt {
		_, _ = fmt.Fprint(os.Stderr, "Account not found. Please ensure it is spelled correctly, because it's neither a user nor an organization.\n")
		os.Exit(2)
	}

	if hasRetried {
		_, _ = fmt.Fprint(os.Stderr, "Account still not found. Please ensure it is spelled correctly, because it's neither a user nor an organization.\n")
		os.Exit(2)
	}

	oppositeType := "users"
	currentType := "User"
	if accountType == "users" {
		oppositeType = "orgs"
		currentType = "Organization"
	}

	_, _ = fmt.Fprintf(os.Stderr, "%s not found. Please ensure it is spelled correctly and that it is the correct account type.\n", currentType)
	fmt.Printf("Do you wanna try again using the %s account type? (y/n) ", strings.ToLower(currentType))

	if ReadYesNo() {
		hasRetried = true
		accountType = oppositeType
		clone(page)
		return
	}
	if !quieterQuiet {
		fmt.Println("Exiting...")
	}
	os.Exit(2)
}

func createOutputDir() {
	if !dryRun && outputDir != "." {
		err := os.MkdirAll(outputDir, 0o755)
		if err != nil {
			fmt.Printf("Error creating output directory %s: %s\n", outputDir, err.Error())
		}
	}
}

func handleNextPage(repos, page int) {
	if repos != perPage {
		return
	}
	if !quiet {
		fmt.Printf("\nCloned %d repositories from %s!\n\n", repos, account)
	}

	if all {
		if !quiet {
			fmt.Printf("Cloning page %d of %s\n", page+1, account)
		}
		clone(page + 1)
		return
	}

	if noPrompt && !quieterQuiet {
		fmt.Println("There seem to be more repos, but interactive prompts are disabled.")
		return
	}

	fmt.Printf("Continue cloning on page %d of %s? (Y/n): ", page+1, account)
	if ReadYesNo() {
		if !quiet {
			fmt.Printf("Cloning page %d of %s\n", page+1, account)
		}
		clone(page + 1)
	}

	//
	//if !all && !noPrompt {
	//	fmt.Printf("Continue cloning on page %d of %s? (Y/n): ", page+1, account)
	//	if !ReadYesNo() {
	//		if !quieterQuiet {
	//			fmt.Println("Exiting...")
	//		}
	//		os.Exit(0)
	//	}
	//}

	//
	//if all {
	//	if !quiet {
	//		fmt.Printf("Cloning page %d of %s\n", page+1, account)
	//	}
	//	clone(page + 1)
	//} else if noPrompt {
	//	if !quieterQuiet {
	//		fmt.Println("There seem to be more repos, but interactive prompts are disabled.")
	//	}
	//} else {
	//	fmt.Printf("Continue cloning on page %d of %s? (Y/n): ", page+1, account)
	//	if ReadYesNo() {
	//		if !quiet {
	//			fmt.Printf("Cloning page %d of %s\n", page+1, account)
	//		}
	//		clone(page + 1)
	//	}
	//}
}

func cloneRepo(repo Repository) {
	if dryRun {
		var outDir string
		if outputDir == "." {
			outDir = repo.Name
		} else {
			outDir = outputDir + "/" + repo.Name
		}
		fmt.Printf("Would have cloned %s to %s\n", repo.CloneURL, outDir)
		return
	}

	cmd := exec.Command("git", "clone", repo.CloneURL)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_SSH_COMMAND='ssh -oBatchMode=yes'")
	if outputDir != "." {
		cmd.Dir = outputDir
	}
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err := cmd.Run()

	// Check if err is of type ExitError
	// This is the only way to check if the command exited with a non-zero exit code
	// https://stackoverflow.com/a/10385867

	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() == 128 {
		fmt.Printf("Skipping %s because it already exists\n", repo.Name)
		return
	}

	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error cloning repository %s: %v\n", repo.Name, err)
	}
}
