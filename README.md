# bracc-download

`bracc-download` is a downloader for open data sources used by the br/acc movement.

The focus is raw data acquisition: download datasets to disk without extraction, transformation, or ingestion.

## Usage

List configured jobs:

```bash
go run ./cmd/bracc list
```

Download all matching jobs to a destination directory:

```bash
go run ./cmd/bracc download DESTINATION
```

Filter by URL substring:

```bash
go run ./cmd/bracc list --url-filter transparencia
go run ./cmd/bracc download ../data --url-filter cvm.gov.br
```

Enable verbose logs:

```bash
go run ./cmd/bracc list --verbose
go run ./cmd/bracc download ../data --url-filter receita --verbose
```

Configured data sources are built into the binary, and downloaded files are organized by source URL.
