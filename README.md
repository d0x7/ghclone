# GH Clone

[![goreleaser](https://github.com/d0x7/ghclone/actions/workflows/goreleaser.yaml/badge.svg)](https://github.com/d0x7/ghclone/actions/workflows/goreleaser.yaml)
![License](https://img.shields.io/badge/license-MIT-blue)

A simple cli tool that clones whole GitHub accounts or organizations.

## Installation

You can either download the latest release from the [release page](https://github.com/d0x7/ghclone/releases/latest), or
you can install it via `go install`:

```bash
go install xiam.li/ghclone/cmd/ghclone@latest
```

`ghclone` will then be available in your `$GOPATH/bin` directory.

## Usage

The command syntax is as follows:

```bash
ghclone [flags] <account>
```

If the `--type`/`-t` flag is not supplied, the tool will try a organization first and if that fails, it will ask to t

The `account` argument can either be a GitHub username or an organization name.

To clone a GitHub organization, run the following command:

```bash
ghclone --all golang
```

This will clone all repositories of the `golang` organization into the `golang` directory.

If there are more than 100 repositories, you will be prompted if you wanna clone the next page too, unless
the `--all`/`-a` flag is supplied, then all repositories will be downloaded without a prompt.
If `--all`/`-a` is not supplied but `--no-prompt`/`-np` is, the tool will stop after the first page, without a prompt.

If you want to clone a personal account, run the following command:

```bash
ghclone -u octocat
```

This would download the first page (by default 100 repositories, customizable with the `--per-page`/`-pp` flag) and if
there are more repositories left, it will prompt you if you wanna clone the next page and so on.

## Flags

| Flag                      | Description                                                                                                                                                                            | Default      |
|---------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------|
| `--help`/`-h`             | Shows the help.                                                                                                                                                                        |              |
| `--version`/`-v`          | Shows the version.                                                                                                                                                                     |              |
| `--type`/`-t`             | The type of the GitHub account. Can be either `user` or `org`.                                                                                                                         | `org`        |
| `--user`/`-u`             | Alias to `--type user`                                                                                                                                                                 |              |
| `--page`/`-p`             | The page to start cloning from.                                                                                                                                                        | `1`          |
| `--per-page`/`-pp`        | The amount of repositories per page.                                                                                                                                                   | `100`        |
| `--verbose`/`-V`          | Whether to print verbose output.                                                                                                                                                       | `false`      |
| `-quiet`/`-q`             | Whether to suppress unnecessary, informational, output.                                                                                                                                | `false`      |
| `-quieter-quiet`/`-qq`    | Whether to suppress all output. Errors will still be printed.                                                                                                                          | `false`      |
| `--no-prompt`/`-np`       | Whether to prompt the user to clone the next page.                                                                                                                                     | `false`      |
| `--all`/`-a`              | Whether to clone all pages.                                                                                                                                                            | `false`      |
| `--dry-run`/`--dry`/`-dr` | Whether to only print the repositories that would be cloned.                                                                                                                           | `false`      |
| `--output`/`-o`           | The output directory.                                                                                                                                                                  | Account Name |
| `--token`                 | GitHub Fine-grained Token for use in authentication. Optional to avoid rate limiting.<br/>Needs Repository->Metadata->Read-Only and Repository->Contents->Read-Only permissions.       |              |
| `--pat`                   | GitHub Personal Access Token for use in authentication. Optional to avoid rate limiting.<br/>Needs full repo access permissions. Use of fine-grained tokens are recommended over PATs. |              |

**Note:** A PAT can be supplied via the token flag and vice-versa; they're both parsed into the same field, and there's
no difference whether of the two flags are used.

## License

[MIT](LICENSE) © 2023 Dorian Heinrichs
