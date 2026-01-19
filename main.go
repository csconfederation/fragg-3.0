// =============================================================================
// DISCLAIMER: Comments in this file were generated with AI assistance to help
// users find and understand code for reference while building FraGG 3.0.
// =============================================================================

// Package main is the entry point for the eco-rating application.
// This tool parses CS2 demo files to calculate advanced player performance
// ratings based on economic impact, round swing, and various statistical metrics.
//
// The application supports two primary modes:
//   - Single demo mode: Parse a single .dem file and export player statistics
//   - Cumulative mode: Batch process multiple demos from a cloud bucket by tier
//
// Usage:
//
//	eco-rating -demo=path/to/demo.dem              # Single demo
//	eco-rating -cumulative -tier=contender         # Cumulative mode
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"eco-rating/bucket"
	"eco-rating/config"
	"eco-rating/downloader"
	"eco-rating/export"
	"eco-rating/model"
	"eco-rating/output"
	"eco-rating/parser"
)

// main initializes the application, parses command-line flags, loads configuration,
// and routes execution to either cumulative mode or single demo parsing mode.
func main() {
	configPath := flag.String("config", "", "Path to configuration file (defaults to config.json in executable directory)")
	cumulative := flag.Bool("cumulative", false, "Enable cumulative mode to fetch all demos for a tier")
	tier := flag.String("tier", "", "Tier to filter demos (challenger, contender, elite, premier, prospect, recruit)")
	demoPath := flag.String("demo", "", "Path to a single demo file to parse")
	demoDir := flag.String("demo-dir", "", "Directory for downloaded demos")
	outputPath := flag.String("output", "stats.csv", "Output path for exported stats (CSV)")
	flag.Parse()

	cfgPath := *configPath
	if cfgPath == "" {
		if _, err := os.Stat("config.json"); err == nil {
			cfgPath = "config.json"
		} else {
			exePath, err := os.Executable()
			if err != nil {
				cfgPath = "config.json"
			} else {
				cfgPath = filepath.Join(filepath.Dir(exePath), "config.json")
			}
		}
	}

	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *cumulative {
		cfg.Cumulative = true
	}
	if *tier != "" {
		cfg.Tier = *tier
	}
	if *demoDir != "" {
		cfg.DemoDir = *demoDir
	}
	if *demoPath != "" {
		cfg.DemoPath = *demoPath
	}

	exporter := export.NewFileExportOption(*outputPath)

	if cfg.Cumulative {
		if cfg.Tier == "" {
			log.Fatal("Tier must be specified in cumulative mode (use -tier flag or set in config)")
		}
		tiers := config.ParseTiers(cfg.Tier)
		for _, t := range tiers {
			if !config.IsValidTier(t) {
				log.Fatalf("Invalid tier '%s'. Valid tiers: %v", t, config.ValidTiers())
			}
		}

		runCumulativeMode(cfg, tiers, exporter)
		return
	}

	if cfg.DemoPath != "" {
		parseSingleDemo(cfg.DemoPath, cfg.EnableLogging, exporter)
		return
	}

	fmt.Println("Usage:")
	fmt.Println("  Cumulative mode: eco-rating -cumulative -tier=contender")
	fmt.Println("  Single demo:     eco-rating -demo=path/to/demo.dem")
	fmt.Println("  Or set demo_path in config.json")
	fmt.Println()
	flag.PrintDefaults()
}

// ParseResult holds the outcome of parsing a single demo file.
// It contains player statistics, map information, and any errors encountered.
type ParseResult struct {
	DemoKey string                        // Unique identifier for the demo file
	Players map[uint64]*model.PlayerStats // Map of Steam ID to player statistics
	MapName string                        // Name of the map played (e.g., de_dust2)
	Tier    string                        // Competitive tier (e.g., contender, elite)
	Logs    string                        // Debug/parsing logs if enabled
	Error   error                         // Any error encountered during parsing
}

// downloadedDemo represents a demo file that has been downloaded and extracted.
type downloadedDemo struct {
	Key  string // Original bucket key/path for the demo
	Path string // Local filesystem path to the extracted .dem file
}

// runCumulativeMode processes all demos for the specified tiers from the cloud bucket.
// It downloads demos, parses them in parallel, aggregates statistics across all games,
// and exports the final results. This is the primary mode for batch processing.
func runCumulativeMode(cfg *config.Config, tiers []string, exporter export.ExportOption) {
	log.Printf("Running in cumulative mode for tiers: %v", tiers)

	client := bucket.NewClient(cfg.BaseURL)
	dl := downloader.NewDownloader(cfg.DemoDir)
	aggregator := output.NewAggregator()

	for _, tier := range tiers {
		log.Printf("\n=== Processing tier: %s ===", tier)

		log.Printf("Fetching demo list from %s%s...", cfg.BaseURL, cfg.Prefix)
		demos, err := client.GetAllDemosByTier(cfg.Prefix, tier)
		if err != nil {
			log.Printf("Failed to get demos for tier %s: %v", tier, err)
			continue
		}

		log.Printf("Found %d demos for tier '%s'", len(demos), tier)

		var downloadedDemos []downloadedDemo

		log.Printf("Downloading demos...")
		for i, demo := range demos {
			log.Printf("[%d/%d] Downloading: %s", i+1, len(demos), demo.Key)

			url := client.GetDownloadURL(demo.Key)
			demoPath, err := dl.DownloadAndExtract(url)
			if err != nil {
				log.Printf("  Error downloading: %v", err)
				continue
			}

			downloadedDemos = append(downloadedDemos, downloadedDemo{Key: demo.Key, Path: demoPath})
		}

		log.Printf("Downloaded %d demos for tier %s, starting parallel parsing...", len(downloadedDemos), tier)

		successCount, allLogs := parseDemosToAggregator(cfg, downloadedDemos, aggregator, tier)

		if len(allLogs) > 0 {
			log.Printf("\n========== PARSING LOGS (%s) ==========", tier)
			for _, logOutput := range allLogs {
				fmt.Println(logOutput)
			}
			log.Printf("========== END LOGS ==========\n")
		}

		log.Printf("Completed processing %d/%d demos for tier %s", successCount, len(downloadedDemos), tier)
	}

	aggregator.Finalize()

	results := aggregator.GetResults()

	if err := exporter.ExportAggregated(results); err != nil {
		log.Fatalf("Failed to export aggregated stats: %v", err)
	}

	log.Printf("\nAggregated stats for %d players across %d tiers exported successfully", len(results), len(tiers))
}

// parseDemosToAggregator processes multiple demos in parallel using a worker pool.
// It returns the count of successfully parsed demos and collected log output.
// The number of workers is capped at 8 or the number of CPU cores, whichever is lower.
func parseDemosToAggregator(cfg *config.Config, downloadedDemos []downloadedDemo, aggregator *output.Aggregator, tier string) (int, []string) {
	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8
	}
	log.Printf("Using %d parallel workers", numWorkers)

	jobs := make(chan downloadedDemo, len(downloadedDemos))
	results := make(chan ParseResult, len(downloadedDemos))

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				players, mapName, logs, err := parseDemoWithLogs(job.Path, cfg.EnableLogging)
				results <- ParseResult{
					DemoKey: job.Key,
					Players: players,
					MapName: mapName,
					Tier:    tier,
					Logs:    logs,
					Error:   err,
				}
			}
		}()
	}

	for _, demo := range downloadedDemos {
		jobs <- demo
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var allLogs []string
	successCount := 0
	processedCount := 0

	for result := range results {
		processedCount++
		if result.Error != nil {
			log.Printf("[%d/%d] Parse error for %s: %v", processedCount, len(downloadedDemos), result.DemoKey, result.Error)
			continue
		}

		aggregator.AddGame(result.Players, result.MapName, result.Tier)
		successCount++
		log.Printf("[%d/%d] Parsed: %s (map: %s, players: %d)", processedCount, len(downloadedDemos), result.DemoKey, result.MapName, len(result.Players))

		if result.Logs != "" {
			allLogs = append(allLogs, fmt.Sprintf("=== %s ===\n%s", result.DemoKey, result.Logs))
		}
	}

	return successCount, allLogs
}

// parseSingleDemo parses a single demo file and exports the results.
// This is used when the -demo flag is provided or demo_path is set in config.
func parseSingleDemo(demoPath string, enableLogging bool, exporter export.ExportOption) {
	demo, err := os.Open(demoPath)
	if err != nil {
		log.Fatalf("Failed to open demo: %v", err)
	}
	defer demo.Close()

	p := parser.NewDemoParserWithLogging(demo, enableLogging)
	if err := p.Parse(); err != nil {
		log.Fatalf("Failed to parse demo: %v", err)
	}

	if err := exporter.Export(p.GetPlayers()); err != nil {
		log.Fatalf("Failed to export stats: %v", err)
	}

	log.Printf("Results exported successfully")
}

// parseDemoWithLogs opens and parses a demo file, returning player stats, map name,
// log output, and any error. This is the core parsing function used by both modes.
func parseDemoWithLogs(demoPath string, enableLogging bool) (map[uint64]*model.PlayerStats, string, string, error) {
	demo, err := os.Open(demoPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to open demo: %w", err)
	}
	defer demo.Close()

	p := parser.NewDemoParserWithLogging(demo, enableLogging)
	if err := p.Parse(); err != nil {
		return nil, "", "", fmt.Errorf("failed to parse demo: %w", err)
	}

	return p.GetPlayers(), p.GetMapName(), p.GetLogs(), nil
}
