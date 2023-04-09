# GH Clone

[![.github/workflows/release.yaml](https://github.com/d0x7/ghclone/actions/workflows/release.yaml/badge.svg)](https://github.com/d0x7/ghclone/actions/workflows/release.yaml)
![License](https://img.shields.io/badge/license-MIT-blue)

A simple cli tool that clones whole GitHub accounts or organizations.

## Installation

You can either download the latest release from the [releases page](https://github.com/d0x7/ghclone/releases/latest), or you can install it via `go install`:
```bash
go install github.com/d0x7/ghclone@latest
```

`ghclone` will then be available in your `$GOPATH/bin` directory.

## Usage

To clone a GitHub organization, run the following command:
```bash
ghclone --all golang
```

This will clone all repositories of the `golang` organization into the `golang` directory.

If there are more than 100 repositories, you will be prompted if you wanna clone the next page too, unless the `--all`/`-a` flag is supplied, then all repositories will be downloaded without a prompt.
If `--all`/`-a` is not supplied but `--no-prompt`/`-np` is, the tool will stop after the first page, without a prompt.

If you wanna clone a personal account, run the following command:
```bash
ghclone --type user octocat 
```

This would download the first page (by default 100 repositories, customizable with the `--per-page`/`-pp` flag) and if there are more repositories left, it will prompt you if you wanna clone the next page and so on.

## Flags

| Flag                | Description                                                    | Default       |
|---------------------|----------------------------------------------------------------|---------------|
| `--help`/`-h`       | Shows the help.                                                |               |
| `--type`/`-t`       | The type of the GitHub account. Can be either `user` or `org`. | `org`         |
| `--page`/`-p`       | The page to start cloning from.                                | `1`           |
| `--per-page`/`-pp`  | The amount of repositories per page.                           | `100`         |
| `--verbose`/`-v`    | Whether to print verbose output.                               | `false`       |
| `--no-prompt`/`-np` | Whether to prompt the user to clone the next page.             | `false`       |
| `--all`/`-a`        | Whether to clone all pages.                                    | `false`       |
| `--dry-run`/`-dr`   | Whether to only print the repositories that would be cloned.   | `false`       |
| `--output`/`-o`     | The output directory.                                          | Account Name  |

## License

[MIT](LICENSE) © 2023 Dorian Heinrichs
