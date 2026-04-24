# Adding a data source

Every new data source follows the same 5 steps.

## 1. Add an entry to `config/sources.yml`

```yaml
<mode_name>:
  <source_name>:
    url: "https://..."
    enabled: true
    user_agent: "market-digest <your-email>"   # only if the source requires it
```

If the source needs an API key: add `api_key_env: MYAPI_KEY` and read `os.Getenv("MYAPI_KEY")` in your fetcher. Never put the key in the YAML.

## 2. Extend `internal/config/config.go`

Add typed fields under `Sources` matching your new YAML shape.

## 3. Write a migration (if it needs its own table)

`migrations/000N_<name>.sql` — `jobs migrate` picks it up by numeric prefix.

## 4. Add a fetcher under `internal/<domain>/`

Follow the pattern from `internal/insiders/` — `fetch.go` (HTTP + parse + normalize + hash), `store.go` (INSERT OR IGNORE on hash), `rules.go` (alert evaluation against newly inserted rows), and tests for each.

## 5. Wire a subcommand in `jobs/cmd/`

Use `jobrun.Track(ctx, conn, "<name>", ...)` so the run shows up in the TUI Jobs screen.

Add it to a schedule (launchd plist or crontab) only after you're confident it works from `./bin/jobs <name>` by hand.
