## Phase 1 Learnings — Project Foundation
- **Go Multi-Module Build Issues:** When moving `main.go` to a subdirectory (like `cmd/server/main.go`), Go's linker will fail if the root directory still contains `package main` files but lacks a `main()` function. Deleting or using `//go:build ignore` is necessary.
- **Strict Dependency Management:** The TRD forbade GORM, and standardizing on `pgx/v5` and `sqlc` forces a focus on performance and raw SQL control, which is better for financial applications.
- **Environment Variance:** Loading config for multiple consumers dynamically from `CONSUMER_*` env vars allows the service to scale to new apps without code changes.
