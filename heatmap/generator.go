package heatmap

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type Generator struct {
	NodePath string
	CliPath  string
	Event    string
	Source   string
	Force    bool
}

func NewGenerator(cliPath string) *Generator {
	return &Generator{
		NodePath: "node",
		CliPath:  cliPath,
		Event:    "kills",
		Source:   "ebot",
		Force:    false,
	}
}

func (g *Generator) AnalyzeDemos(demoPaths []string) error {
	demoPaths = uniqueNonEmpty(demoPaths)
	if len(demoPaths) == 0 {
		return nil
	}

	workers := runtime.NumCPU()
	if workers > 4 {
		workers = 4
	}
	if workers < 1 {
		workers = 1
	}

	total := int64(len(demoPaths))
	remaining := atomic.Int64{}
	remaining.Store(total)
	log.Printf("csdm analyze: queued %d demos with %d workers", total, workers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan string)
	var firstErr error
	var firstErrOnce sync.Once

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case demoPath, ok := <-jobs:
					if !ok {
						return
					}

					args := []string{
						g.CliPath,
						"analyze",
						demoPath,
					}
					if g.Source != "" {
						args = append(args, "--source", g.Source)
					}
					if g.Force {
						args = append(args, "--force")
					}

					cmd := exec.CommandContext(ctx, g.NodePath, args...)
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
					if err := cmd.Run(); err != nil {
						firstErrOnce.Do(func() {
							firstErr = err
							cancel()
						})
						return
					}

					left := remaining.Add(-1)
					log.Printf("csdm analyze: %d remaining", left)
				}
			}
		}()
	}

	for _, p := range demoPaths {
		select {
		case <-ctx.Done():
			break
		case jobs <- p:
		}
	}
	close(jobs)
	wg.Wait()

	if firstErr != nil {
		return firstErr
	}
	log.Printf("csdm analyze: complete")
	return nil
}

func (g *Generator) GeneratePlayerMapHeatmaps(outputRoot string, playerNameBySteamID map[string]string, demoPathsBySteamIDByMap map[string]map[string][]string) error {
	if outputRoot == "" {
		outputRoot = "heatmaps"
	}

	type heatmapJob struct {
		steamID      string
		mapName      string
		demoPathArg  string
		outputFile   string
		playerName   string
		playerFolder string
	}

	steamIDs := make([]string, 0, len(demoPathsBySteamIDByMap))
	for steamID := range demoPathsBySteamIDByMap {
		steamIDs = append(steamIDs, steamID)
	}
	sort.Strings(steamIDs)

	jobsList := make([]heatmapJob, 0)
	for _, steamID := range steamIDs {
		byMap := demoPathsBySteamIDByMap[steamID]
		playerName := playerNameBySteamID[steamID]
		if playerName == "" {
			playerName = steamID
		}
		playerDir := filepath.Join(outputRoot, sanitizePathComponent(playerName))
		if err := os.MkdirAll(playerDir, 0o755); err != nil {
			return fmt.Errorf("create player heatmap dir: %w", err)
		}

		maps := make([]string, 0, len(byMap))
		for mapName := range byMap {
			maps = append(maps, mapName)
		}
		sort.Strings(maps)

		for _, mapName := range maps {
			demoPaths := uniqueNonEmpty(byMap[mapName])
			if len(demoPaths) == 0 {
				continue
			}
			outFile := filepath.Join(playerDir, sanitizePathComponent(mapName)+".png")
			jobsList = append(jobsList, heatmapJob{
				steamID:      steamID,
				mapName:      mapName,
				demoPathArg:  strings.Join(demoPaths, ","),
				outputFile:   outFile,
				playerName:   playerName,
				playerFolder: playerDir,
			})
		}
	}

	if len(jobsList) == 0 {
		return nil
	}

	workers := runtime.NumCPU()
	if workers > 4 {
		workers = 4
	}
	if workers < 1 {
		workers = 1
	}

	total := int64(len(jobsList))
	remaining := atomic.Int64{}
	remaining.Store(total)
	log.Printf("csdm heatmaps: queued %d heatmaps with %d workers", total, workers)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	jobs := make(chan heatmapJob)
	var firstErr error
	var firstErrOnce sync.Once

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-jobs:
					if !ok {
						return
					}
					if err := g.generateHeatmap(job.demoPathArg, job.steamID, job.outputFile); err != nil {
						firstErrOnce.Do(func() {
							firstErr = fmt.Errorf("generate heatmap for player %s map %s: %w", job.steamID, job.mapName, err)
							cancel()
						})
						return
					}
					left := remaining.Add(-1)
					log.Printf("csdm heatmaps: %d remaining", left)
				}
			}
		}()
	}

	for _, job := range jobsList {
		select {
		case <-ctx.Done():
			break
		case jobs <- job:
		}
	}
	close(jobs)
	wg.Wait()

	if firstErr != nil {
		return firstErr
	}
	log.Printf("csdm heatmaps: complete")
	return nil
}

func (g *Generator) generateHeatmap(demoPathArg string, steamID string, outputFile string) error {
	args := []string{
		g.CliPath,
		"heatmap",
		"--demo-path",
		demoPathArg,
		"--steamids",
		steamID,
		"--event",
		g.Event,
		"--sides",
		"3",
		"--output",
		outputFile,
	}

	cmd := exec.Command(g.NodePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

var invalidPathChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)

func sanitizePathComponent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}
	s = invalidPathChars.ReplaceAllString(s, "_")
	s = strings.Trim(s, ". ")
	if s == "" {
		return "unknown"
	}
	return s
}

func uniqueNonEmpty(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, it := range items {
		it = strings.TrimSpace(it)
		if it == "" {
			continue
		}
		if _, ok := seen[it]; ok {
			continue
		}
		seen[it] = struct{}{}
		out = append(out, it)
	}
	return out
}
