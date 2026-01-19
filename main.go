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
	"eco-rating/heatmap"
	"eco-rating/model"
	"eco-rating/output"
	"eco-rating/parser"
)

func main() {
	// Command line flags
	configPath := flag.String("config", "", "Path to configuration file (defaults to config.json in executable directory)")
	cumulative := flag.Bool("cumulative", false, "Enable cumulative mode to fetch all demos for a tier")
	tier := flag.String("tier", "", "Tier to filter demos (challenger, contender, elite, premier, prospect, recruit)")
	demoPath := flag.String("demo", "", "Path to a single demo file to parse")
	outputDir := flag.String("output", "./demos", "Output directory for downloaded demos")
	generateHeatmaps := flag.Bool("heatmaps", false, "Generate per-player per-map heatmaps using cs-demo-manager CLI (overrides config)")
	disableHeatmaps := flag.Bool("no-heatmaps", false, "Disable cs-demo-manager integration (overrides config)")
	csdmCliPath := flag.String("csdm-cli", "C:\\Users\\ethan\\GolandProjects\\cs-demo-manager\\out\\cli.js", "Path to cs-demo-manager CLI build (out/cli.js)")
	csdmForceAnalyze := flag.Bool("csdm-force-analyze", false, "Force cs-demo-manager to re-analyze demos when generating heatmaps")
	flag.Parse()

	// Determine config path - default to config.json in current working directory
	cfgPath := *configPath
	if cfgPath == "" {
		// First try current working directory
		if _, err := os.Stat("config.json"); err == nil {
			cfgPath = "config.json"
		} else {
			// Fall back to executable directory
			exePath, err := os.Executable()
			if err != nil {
				cfgPath = "config.json"
			} else {
				cfgPath = filepath.Join(filepath.Dir(exePath), "config.json")
			}
		}
	}

	// Load configuration
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override config with command line flags if provided
	if *cumulative {
		cfg.Cumulative = true
	}
	if *tier != "" {
		cfg.Tier = *tier
	}
	if *outputDir != "./demos" {
		cfg.OutputDir = *outputDir
	}
	if *demoPath != "" {
		cfg.DemoPath = *demoPath
	}

	// CS Demo Manager integration is enabled by default (see config.enable_csdm).
	// CLI flags take precedence.
	if *generateHeatmaps {
		cfg.EnableCsdm = true
	}
	if *disableHeatmaps {
		cfg.EnableCsdm = false
	}

	// Cumulative mode - fetch all demos for a tier and aggregate stats
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

		runCumulativeModeMultiTier(cfg, tiers, cfg.EnableCsdm, *csdmCliPath, *csdmForceAnalyze)
		return
	}

	// Single demo mode - parse demo from config or flag
	if cfg.DemoPath != "" {
		parseSingleDemo(cfg.DemoPath, cfg.EnableLogging)
		return
	}

	// Default behavior - show usage
	fmt.Println("Usage:")
	fmt.Println("  Cumulative mode: eco-rating -cumulative -tier=contender")
	fmt.Println("  Single demo:     eco-rating -demo=path/to/demo.dem")
	fmt.Println("  Or set demo_path in config.json")
	fmt.Println()
	flag.PrintDefaults()
}

// ParseResult holds the result of parsing a single demo
type ParseResult struct {
	DemoKey string
	Players map[uint64]*model.PlayerStats
	MapName string
	Tier    string
	Logs    string
	Error   error
}

type downloadedDemo struct {
	Key  string
	Path string
}

func runCumulativeModeMultiTier(cfg *config.Config, tiers []string, generateHeatmaps bool, csdmCliPath string, csdmForceAnalyze bool) {
	log.Printf("Running in cumulative mode for tiers: %v", tiers)

	// Create bucket client
	client := bucket.NewClient(cfg.BaseURL)

	// Create downloader
	dl := downloader.NewDownloader(cfg.OutputDir)

	// Shared aggregator across all tiers
	aggregator := output.NewAggregator()

	// Track demos by map and by player for heatmap generation.
	playerNameBySteamID := make(map[string]string)
	demoPathsBySteamIDByMap := make(map[string]map[string][]string)
	var allDownloadedDemos []downloadedDemo

	for _, tier := range tiers {
		log.Printf("\n=== Processing tier: %s ===", tier)

		// Get all demos for this tier
		log.Printf("Fetching demo list from %s%s...", cfg.BaseURL, cfg.Prefix)
		demos, err := client.GetAllDemosByTier(cfg.Prefix, tier)
		if err != nil {
			log.Printf("Failed to get demos for tier %s: %v", tier, err)
			continue
		}

		log.Printf("Found %d demos for tier '%s'", len(demos), tier)

		// Download all demos for this tier
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

		allDownloadedDemos = append(allDownloadedDemos, downloadedDemos...)

		log.Printf("Downloaded %d demos for tier %s, starting parallel parsing...", len(downloadedDemos), tier)

		// Parse demos for this tier and add to aggregator
		successCount, allLogs := parseDemosToAggregator(cfg, downloadedDemos, aggregator, generateHeatmaps, playerNameBySteamID, demoPathsBySteamIDByMap, tier)

		// Print logs for this tier
		if len(allLogs) > 0 {
			log.Printf("\n========== PARSING LOGS (%s) ==========", tier)
			for _, logOutput := range allLogs {
				fmt.Println(logOutput)
			}
			log.Printf("========== END LOGS ==========\n")
		}

		log.Printf("Completed processing %d/%d demos for tier %s", successCount, len(downloadedDemos), tier)
	}

	// Finalize and export aggregated stats across all tiers
	aggregator.Finalize()

	results := aggregator.GetResults()

	// Upload to Google Sheets if configured, otherwise write to JSON file
	if cfg.UploadToSheet && cfg.StatsSheetURL != "" {
		log.Printf("Uploading stats to Google Sheets...")

		// Load service account credentials
		credentialsPath := "csc-extended-stats-c5e08c92e535.json"
		credentials, err := os.ReadFile(credentialsPath)
		if err != nil {
			log.Fatalf("Failed to read service account credentials from %s: %v", credentialsPath, err)
		}

		sheetName := cfg.StatsSheetName
		if sheetName == "" {
			sheetName = "ratings"
		}

		sheetsClient, err := output.NewSheetsClient(credentials, cfg.StatsSheetURL, sheetName)
		if err != nil {
			log.Fatalf("Failed to create Google Sheets client: %v", err)
		}

		if err := sheetsClient.UploadAggregatedStats(results); err != nil {
			log.Fatalf("Failed to upload stats to Google Sheets: %v", err)
		}

		log.Printf("\nAggregated stats for %d players across %d tiers uploaded to Google Sheets", len(results), len(tiers))
	} else {
		outputPath := "match_rating.json"
		if err := output.ExportAggregated(results, outputPath); err != nil {
			log.Fatalf("Failed to export aggregated stats: %v", err)
		}

		log.Printf("\nAggregated stats for %d players across %d tiers saved to: %s", len(results), len(tiers), outputPath)
	}

	if generateHeatmaps {
		log.Printf("Generating heatmaps...")
		g := heatmap.NewGenerator(csdmCliPath)
		g.Force = csdmForceAnalyze

		allDemoPaths := make([]string, 0, len(allDownloadedDemos))
		for _, dd := range allDownloadedDemos {
			allDemoPaths = append(allDemoPaths, dd.Path)
		}
		if err := g.AnalyzeDemos(allDemoPaths); err != nil {
			log.Printf("csdm analyze failed, skipping heatmaps: %v", err)
			return
		}
		if err := g.GeneratePlayerMapHeatmaps(cfg.HeatmapPath, playerNameBySteamID, demoPathsBySteamIDByMap); err != nil {
			log.Printf("Heatmap generation failed: %v", err)
		} else {
			log.Printf("Heatmaps generated in: %s", cfg.HeatmapPath)
		}
	}
}

func parseDemosToAggregator(cfg *config.Config, downloadedDemos []downloadedDemo, aggregator *output.Aggregator, generateHeatmaps bool, playerNameBySteamID map[string]string, demoPathsBySteamIDByMap map[string]map[string][]string, tier string) (int, []string) {
	var heatmapMu sync.Mutex

	// Set up worker pool for concurrent parsing
	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8
	}
	log.Printf("Using %d parallel workers", numWorkers)

	// Create channels for work distribution and result collection
	jobs := make(chan downloadedDemo, len(downloadedDemos))
	results := make(chan ParseResult, len(downloadedDemos))

	// Start workers
	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				players, mapName, logs, err := parseDemoWithLogs(job.Path, cfg.EnableLogging)
				if err == nil && generateHeatmaps {
					heatmapMu.Lock()
					for _, ps := range players {
						steamID := ps.SteamID
						if steamID == "" {
							continue
						}
						if ps.Name != "" {
							playerNameBySteamID[steamID] = ps.Name
						}
						byMap, ok := demoPathsBySteamIDByMap[steamID]
						if !ok {
							byMap = make(map[string][]string)
							demoPathsBySteamIDByMap[steamID] = byMap
						}
						if mapName != "" {
							byMap[mapName] = append(byMap[mapName], job.Path)
						}
					}
					heatmapMu.Unlock()
				}
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

	// Send jobs to workers
	for _, demo := range downloadedDemos {
		jobs <- demo
	}
	close(jobs)

	// Wait for all workers to finish in a separate goroutine
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
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

		// Collect logs if any
		if result.Logs != "" {
			allLogs = append(allLogs, fmt.Sprintf("=== %s ===\n%s", result.DemoKey, result.Logs))
		}
	}

	return successCount, allLogs
}

func parseSingleDemo(demoPath string, enableLogging bool) {
	demo, err := os.Open(demoPath)
	if err != nil {
		log.Fatalf("Failed to open demo: %v", err)
	}
	defer demo.Close()

	p := parser.NewDemoParserWithLogging(demo, enableLogging)
	p.Parse()

	outputPath := "match_rating.json"
	if err := p.ExportJSON(outputPath); err != nil {
		log.Fatalf("Failed to export JSON: %v", err)
	}

	log.Printf("Results saved to: %s", outputPath)
}

func parseDemoWithLogs(demoPath string, enableLogging bool) (map[uint64]*model.PlayerStats, string, string, error) {
	demo, err := os.Open(demoPath)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to open demo: %w", err)
	}
	defer demo.Close()

	p := parser.NewDemoParserWithLogging(demo, enableLogging)
	p.Parse()

	return p.GetPlayers(), p.GetMapName(), p.GetLogs(), nil
}
