package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

type Repository struct {
	Name     string `json:"name"`
	CloneURL string `json:"clone_url"`
}

var (
	account     string
	accountType string
	outputDir   string
	startPage   int
	perPage     int
	verbose     bool
	noPrompt    bool
	all         bool
	dryRun      bool
	totalCloned int
)

func main() {
	flag.StringVar(&accountType, "type", "org", "What type of GitHub account to clone, either \"org\" or \"user\"")
	flag.StringVar(&accountType, "t", "org", "What type of GitHub account to clone, either \"org\" or \"user\"")
	flag.StringVar(&outputDir, "output", "AccountName", "The directory to clone the repos to")
	flag.StringVar(&outputDir, "o", "AccountName", "The directory to clone the repos to")
	flag.IntVar(&startPage, "page", 1, "The page number to start cloning from")
	flag.IntVar(&startPage, "p", 1, "The page number to start cloning from")
	flag.IntVar(&perPage, "per-page", 100, "The number of entities to clone per page")
	flag.IntVar(&perPage, "pp", 100, "The number of entities to clone per page")
	flag.BoolVar(&verbose, "verbose", false, "Whether to print verbose output")
	flag.BoolVar(&verbose, "v", false, "Whether to print verbose output")
	flag.BoolVar(&noPrompt, "no-prompt", false, "With this flag enabled, there are no interactive prompts")
	flag.BoolVar(&noPrompt, "np", false, "With this flag enabled, there are no interactive prompts")
	flag.BoolVar(&all, "all", false, "With this flag enabled, all pages will be cloned without prompting")
	flag.BoolVar(&all, "a", false, "With this flag enabled, all pages will be cloned without prompting")
	flag.BoolVar(&dryRun, "dry-run", false, "With this flag enabled, no repos will be cloned")
	flag.BoolVar(&dryRun, "dr", false, "With this flag enabled, no repos will be cloned")

	checkVars()
	clone(startPage)
}

func checkVars() {
	flag.Parse()

	if accountType != "org" && accountType != "user" {
		fmt.Println("Invalid account type provided")
		os.Exit(1)
	}
	if accountType == "org" {
		accountType = "orgs"
	}

	if startPage < 1 {
		fmt.Println("Invalid page number provided")
		os.Exit(1)
	}

	if perPage < 1 {
		fmt.Println("Invalid items number provided")
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

func clone(page int) {
	requestUrl := fmt.Sprintf("https://api.github.com/%s/%s/repos?page=%d&per_page=%d", accountType, account, page, perPage)
	resp, err := http.Get(requestUrl)
	if err != nil || resp.StatusCode != 200 {
		println("Error making request to GitHub API")
		panic(err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			println("Error closing response body")
			panic(err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		println("Error reading response body")
		panic(err)
	}

	var repos []Repository
	if err := json.Unmarshal(body, &repos); err != nil {
		println("Error unmarshalling response body")
		panic(err)
	}

	if !all {
		fmt.Printf("Cloning %d repositories from %s starting from page %d..\n", len(repos), account, page)
	}

	if !dryRun && outputDir != "." {
		err := os.MkdirAll(outputDir, 0755)
		if err != nil {
			fmt.Printf("Error creating output directory %s: %s\n", outputDir, err.Error())
		}
	}

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

	if len(repos) == perPage {
		fmt.Printf("\nCloned %d repositories from %s!\n\n", len(repos), account)
		if all {
			fmt.Printf("Cloning page %d of %s\n", page+1, account)
			clone(page + 1)
		} else if noPrompt {
			fmt.Println("There seem to be more repos, but interactive prompts are disabled.")
		} else {
			fmt.Printf("Continue cloning on page %d of %s? (Y/n): ", page+1, account)
			read, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				panic(err)
			}
			read = strings.ToLower(strings.TrimSpace(read))

			if read == "" || read == "y" || read == "yes" {
				fmt.Printf("Cloning page %d of %s\n", page+1, account)
				clone(page + 1)
			}
		}
	}
	var allString string
	if all {
		allString = "all "
	}
	fmt.Printf("Done cloning %s%d repos from %s. Bye bye!\n", allString, totalCloned, account)
	os.Exit(0)
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
	if err != nil {
		fmt.Printf("Error cloning repository %s: %s\n", repo.Name, err.Error())
	}
}
