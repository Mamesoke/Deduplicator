# 🔍 Deduplication

**Deduplication** is a fast and lightweight command-line tool to scan directories for duplicate files based on SHA-256 hashes. It helps you detect redundant files and recover disk space with clear, structured output in both human-readable and JSON formats.

---

## ✨ Features

- 🚀 Fast and efficient directory scanning
- 🔐 Hash-based duplicate detection (SHA-256)
- 🧠 Smart grouping by content, not by name
- 💬 Output formats: `pretty` (for humans) or `json` (for scripts, GUIs)
- 🧩 Easily extensible and modular architecture (written in Go)

---

## 📦 Installation

### With `go install`:

```bash
go install deduplication # (WIP)
```

### Manual build:
```bash
git clone https://github.com/mamesoke/deduplication.git
cd deduplication
go build -o deduplication
```

---

### 🧪 Usage
```bash
go run main.go -dir=/path/to/scan --format=pretty

go run main.go -dir=/path/to/scan --format=json
```

Options
| Flag     | Description                     | Default    |
|----------|---------------------------------|------------|
| -dir     | Directory path to scan          | (required) |
| --format | Output format: pretty or json   | pretty     |

### 🖥 Example Output

## Pretty
```
🔁 Grupo #1 — 3 archivos duplicados (Hash: abc123...)
    Tamaño por archivo: 5210 bytes | Total duplicado: 10420 bytes
    - /docs/a.pdf
    - /old/a_copy.pdf
    - /backups/a.pdf

📊 Resumen:
  - Total de grupos de duplicados: 2
  - Total de archivos duplicados: 3
  - Espacio potencial recuperable: 10.50 MB
```

## JSON
``` json
{
  "groups": [
    {
      "hash": "abc123...",
      "files": [
        { "path": "/docs/a.pdf", "size": 5210, "lastModified": 1712178000 },
        ...
      ]
    }
  ],
  "total_duplicated_files": 3,
  "total_wasted_bytes": 10420,
  "total_groups": 2
}

```

---

## Roadmap
Roadmap
- Parallel file hashing with worker pool
- File filters (by extension, size, etc.)
- Actions on duplicates (delete, move, dry-run)
- .dedupignore file support
- Export to CSV/YAML
- GUI client on top of the same engine

## 👨‍💻 Author
Built by @mamesoke — contributions and PRs welcome!

## 📄 License
MIT © 2025