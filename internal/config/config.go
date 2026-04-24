// Package config loads market-digest YAML configuration from <root>/config/.
// It prefers *.yml over *.example.yml so users can override without editing
// tracked files.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Profile   Profile
	Watchlist Watchlist
	Sources   Sources
}

type Profile struct {
	User struct {
		DisplayName string `yaml:"display_name"`
		Timezone    string `yaml:"timezone"`
	} `yaml:"user"`
	Risk struct {
		Tolerance      string `yaml:"tolerance"`
		MaxPositionPct int    `yaml:"max_position_pct"`
		Notes          string `yaml:"notes"`
	} `yaml:"risk"`
	Interests struct {
		Sectors []string `yaml:"sectors"`
		Themes  []string `yaml:"themes"`
		Avoid   []string `yaml:"avoid"`
	} `yaml:"interests"`
	Reporting struct {
		DollarThresholds struct {
			Watch int `yaml:"watch"`
			Act   int `yaml:"act"`
		} `yaml:"dollar_thresholds"`
		ClusterWindowDays int `yaml:"cluster_window_days"`
	} `yaml:"reporting"`
	Notifications struct {
		MacOS      bool   `yaml:"macos"`
		WebhookURL string `yaml:"webhook_url"`
	} `yaml:"notifications"`
}

type Watchlist struct {
	Tickers []WatchlistEntry `yaml:"tickers"`
}

type WatchlistEntry struct {
	Ticker string `yaml:"ticker"`
	Note   string `yaml:"note"`
	Added  string `yaml:"added"`
}

type Sources struct {
	Insiders   map[string]InsiderSource `yaml:"insiders"`
	AlertRules AlertRules               `yaml:"alert_rules"`
	HTTP       HTTPConfig               `yaml:"http"`
}

type InsiderSource struct {
	URL       string `yaml:"url"`
	Enabled   bool   `yaml:"enabled"`
	UserAgent string `yaml:"user_agent,omitempty"`
}

type AlertRules struct {
	WatchlistHit    AlertRule `yaml:"watchlist_hit"`
	AmountOver500k  AlertRule `yaml:"amount_over_500k"`
	AmountOver1m    AlertRule `yaml:"amount_over_1m"`
	Cluster3In7d    AlertRule `yaml:"cluster_3_in_7d"`
}

type AlertRule struct {
	Severity string `yaml:"severity"`
}

type HTTPConfig struct {
	TimeoutSeconds int `yaml:"timeout_seconds"`
	MaxRetries     int `yaml:"max_retries"`
	BackoffMS      int `yaml:"backoff_ms"`
}

// Load reads config from <root>/config, preferring *.yml over *.example.yml.
// Missing optional files are tolerated only where a fallback makes sense.
func Load(root string) (Config, error) {
	var cfg Config
	if err := loadYAML(filepath.Join(root, "config"), "profile", &cfg.Profile); err != nil {
		return cfg, err
	}
	if err := loadYAML(filepath.Join(root, "config"), "watchlist", &cfg.Watchlist); err != nil {
		return cfg, err
	}
	if err := loadYAML(filepath.Join(root, "config"), "sources", &cfg.Sources); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func loadYAML(dir, name string, out any) error {
	for _, candidate := range []string{name + ".yml", name + ".example.yml"} {
		path := filepath.Join(dir, candidate)
		body, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read %s: %w", candidate, err)
		}
		if err := yaml.Unmarshal(body, out); err != nil {
			return fmt.Errorf("parse %s: %w", candidate, err)
		}
		return nil
	}
	return fmt.Errorf("config file %s.yml (or .example.yml) not found in %s", name, dir)
}
