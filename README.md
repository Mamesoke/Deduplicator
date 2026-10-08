# Deduplicator

A Go command-line tool for finding duplicate files in a directory. It compares file contents using hashes, groups identical files, and reports how much disk space could be recovered. You can also preview or perform deletion of redundant copies.

## Features

- Recursively scans directories for files with matching sizes and contents.
- Calculates hashes with SHA-256 (default), SHA-1, SHA-512, or MD5.
- Uses multiple workers for scanning and hash calculation.
- Stores calculated hashes in `.dedupcache.json` inside the scanned directory to speed up future scans of unchanged files. Cache entries are specific to the selected algorithm and file modification time.
- Prints human-readable results or a JSON report.
- Excludes files and directories using patterns.
- Can preview or delete the duplicates it finds.

## Requirements

- Go 1.24.2 or later.

## Installation

Clone the repository and build the executable:

```sh
git clone https://github.com/Mamesoke/Deduplicator.git
cd Deduplicator
go build -o deduplicator .
```

On Windows, you can add the `.exe` extension:

```powershell
go build -o deduplicator.exe .
```

You can also run the program directly from source with `go run .`, without building a separate executable.

## Usage

The only required argument is `-dir`. Options use a single hyphen, as expected by Go's `flag` package.

```sh
go run . -dir="/path/to/scan"
```

In Windows PowerShell:

```powershell
.\deduplicator.exe -dir="C:\Users\Name\Documents"
```

Results are printed in a human-readable format by default. To request JSON:

```sh
go run . -dir="/path/to/scan" -format=json
```

### Options

| Option | Default | Description |
| --- | --- | --- |
| `-dir` | (required) | Directory to scan. |
| `-format` | `pretty` | Output format: `pretty` or `json`. |
| `-hash` | `sha256` | Hash algorithm: `sha256`, `sha1`, `sha512`, or `md5`. |
| `-exclude` | Built-in list | Filename pattern to exclude. Can be specified multiple times. |
| `-delete` | `false` | Deletes redundant copies after printing the results. |
| `-dry-run` | `false` | With `-delete`, shows which files would be deleted without removing them. |
| `-timings` | `false` | Logs the duration of scan and hash workers. |

The built-in exclusions are `.git`, `node_modules`, `.github`, `.idea`, `.vscode`, `vendor`, `dist`, `build`, `tmp`, `temp`, `.venv`, and `venv`. The `.dedupcache.json` cache file is also automatically skipped. Custom patterns are matched against each file or directory name, not its full path. For example:

```sh
go run . -dir="./project" -exclude="*.log" -exclude="output"
```

Custom exclusions are added to the built-in exclusions; they do not replace them.

### Previewing or deleting duplicates

Before deleting files, inspect the results or run a dry run:

```sh
go run . -dir="/path/to/scan" -delete -dry-run
```

To perform the deletion:

```sh
go run . -dir="/path/to/scan" -delete
```

**Warning:** deletion is permanent and does not prompt for confirmation. One file in each duplicate group is kept and the others are removed only after their contents have been compared byte-for-byte with the kept file. The scan order does not guarantee which file will be kept, so review the dry-run output and make a backup before using `-delete` without `-dry-run`.

### JSON report

With `-format=json`, the program writes a JSON report with this structure (the `groups` array and totals are empty/zero when no duplicates are found):

```json
{
  "groups": [
    {
      "hash": "content-hash",
      "files": [
        {
          "path": "/path/file-a",
          "size": 1234,
          "hash": "content-hash",
          "lastModified": 1712178000
        },
        {
          "path": "/path/file-b",
          "size": 1234,
          "hash": "content-hash",
          "lastModified": 1712178000
        }
      ]
    }
  ],
  "total_duplicated_files": 1,
  "total_wasted_bytes": 1234,
  "total_groups": 1
}
```

`total_duplicated_files` counts redundant copies, not every file in the groups. `lastModified` is the Unix modification time in seconds. The JSON report is written to standard output; scan status, errors, and deletion messages are written to standard error, so the standard output can be piped directly to a JSON parser.

## Development and tests

Run the project tests with:

```sh
go test ./...
```

Build all packages with:

```sh
go build ./...
```

## Project structure

```text
.
├── main.go                 # CLI flags and execution flow
└── deduplicator/
    ├── walker.go           # Directory traversal, exclusions, workers, and cache
    ├── hasher.go           # Hash algorithms
    ├── deduplicator.go     # Duplicate grouping
    ├── formatter.go        # Human-readable and JSON output
    ├── deleter.go          # Preview and deletion of copies
    └── *_test.go           # Tests
```

## Author and license

Developed by [@mamesoke](https://github.com/Mamesoke). See [LICENSE](LICENSE) for the MIT license and the repository for contribution information.
