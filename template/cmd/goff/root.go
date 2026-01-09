package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"goff/template/internal/config"
	"goff/template/internal/download"
	"goff/template/internal/puzzle"
	"goff/template/internal/puzzletext"
	"goff/template/internal/setup"
	"goff/template/internal/summary"
	"goff/template/internal/ui"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "goff",
	Short:         "Helper CLI for FlipFlop Codes",
	SilenceUsage:  true,
	SilenceErrors: true,
}

var yearFlag int

func init() {
	// Load default year from config if available, otherwise use current year
	defaultYear := time.Now().Year()
	if cfg, err := config.Load(); err == nil && cfg.CurrentYear > 0 {
		defaultYear = cfg.CurrentYear
	}

	rootCmd.PersistentFlags().IntVarP(&yearFlag, "year", "y", defaultYear, "puzzle year")

	rootCmd.AddCommand(newPrepareCmd())
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newBenchCmd())
	rootCmd.AddCommand(newGetCmd())
	rootCmd.AddCommand(newSummaryCmd())
	rootCmd.AddCommand(newDownloadCmd())
	rootCmd.AddCommand(newSessionCmd())
	rootCmd.AddCommand(newSetupCmd())
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ui.Error(os.Stderr, "goff error", err.Error())
		os.Exit(1)
	}
}

func newPrepareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "prepare <puzzle> | prepare y <year> <puzzle>",
		Aliases: []string{"p"},
		Short:   "Prepare a puzzle directory",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			year := yearFlag
			dayArg := args[0]
			if len(args) == 3 && (args[0] == "y" || args[0] == "year") {
				parsedYear, err := parseYearArg(args[1])
				if err != nil {
					return err
				}
				year = parsedYear
				dayArg = args[2]
			} else if len(args) != 1 {
				return fmt.Errorf("invalid arguments: %s", strings.Join(args, " "))
			}

			day, err := parsePuzzleArg(dayArg)
			if err != nil {
				return err
			}

			if err := puzzle.Prepare(year, day); err != nil {
				return err
			}
			ui.Success(os.Stdout, "Prepared puzzle", fmt.Sprintf("%d puzzle %02d", year, day))

			if ok, err := download.InputIfAvailable(year, day); err != nil {
				return err
			} else if ok {
				ui.Success(os.Stdout, "Input saved", fmt.Sprintf("%d puzzle %02d (input.txt)", year, day))
			}

			return nil
		},
	}

	return cmd
}

func newRunCmd() *cobra.Command {
	var useTest bool
	var useInput bool

	cmd := &cobra.Command{
		Use:     "run [puzzle] [i|t]",
		Aliases: []string{"r"},
		Short:   "Run a puzzle solution",
		Args:    cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			year, day, useTestArg, err := parseRunArgs(args, useTest, useInput)
			if err != nil {
				return err
			}

			label := "input.txt"
			if useTestArg {
				label = "input_test.txt"
			}

			ui.Info(os.Stdout, "Running puzzle", fmt.Sprintf("%d puzzle %02d (%s)", year, day, label))
			return puzzle.Run(year, day, useTestArg)
		},
	}

	cmd.Flags().BoolVarP(&useTest, "test", "t", false, "use test input")
	cmd.Flags().BoolVarP(&useInput, "input", "i", false, "use real input")

	return cmd
}

func newBenchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "bench [puzzle]",
		Aliases: []string{"b"},
		Short:   "Benchmark a puzzle solution",
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			year, day, err := parsePuzzleFromArgsOrDir(args)
			if err != nil {
				return err
			}

			ui.Info(os.Stdout, "Benchmarking puzzle", fmt.Sprintf("%d puzzle %02d (input.txt)", year, day))
			return puzzle.Bench(year, day)
		},
	}

	return cmd
}

func newGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get [part]",
		Aliases: []string{"g"},
		Short:   "Print the puzzle text for a part",
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			year, day, err := parsePuzzleFromArgsOrDir(nil)
			if err != nil {
				return err
			}

			part := 1
			if len(args) == 1 {
				parsedPart, err := parsePartArg(args[0])
				if err != nil {
					return err
				}
				part = parsedPart
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			text, err := puzzletext.FetchPart(year, day, part, cfg.PHPSESSID)
			if err != nil {
				return err
			}

			fmt.Fprintln(os.Stdout, text)
			return nil
		},
	}

	return cmd
}

func newDownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "download [puzzle]",
		Aliases: []string{"d"},
		Short:   "Download puzzle input",
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			year, day, err := parsePuzzleFromArgsOrDir(args)
			if err != nil {
				return err
			}

			ui.Info(os.Stdout, "Downloading input", fmt.Sprintf("%d puzzle %02d", year, day))
			if err := download.Input(year, day); err != nil {
				return err
			}
			ui.Success(os.Stdout, "Input saved", fmt.Sprintf("%d puzzle %02d (input.txt)", year, day))
			return nil
		},
	}

	return cmd
}

func newSessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "session <token>",
		Aliases: []string{"s"},
		Short:   "Store PHPSESSID session token",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.Config{PHPSESSID: args[0]}
			if err := config.Save(cfg); err != nil {
				return err
			}
			ui.Success(os.Stdout, "Stored session token", fmt.Sprintf("PHPSESSID saved to %s", config.FileName))
			return nil
		},
	}

	return cmd
}

func newSummaryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "summary",
		Aliases: []string{"score", "points", "pointers"},
		Short:   "Update README pointers and benchmarks",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			root, err := config.RepoRoot()
			if err != nil {
				return err
			}
			years, err := summary.Years(root)
			if err != nil {
				return err
			}
			if len(years) == 0 {
				return fmt.Errorf("no year directories found")
			}

			var latest summary.YearSummary
			for _, year := range years {
				yearSummary, err := summary.Build(year, cfg.PHPSESSID, root)
				if err != nil {
					return err
				}
				yearPath := filepath.Join(root, fmt.Sprintf("%d", year), "README.md")
				if err := summary.UpdateReadme(yearPath, yearSummary); err != nil {
					return err
				}
				latest = yearSummary
			}

			rootReadme := filepath.Join(root, "README.md")
			if err := summary.UpdateReadme(rootReadme, latest); err != nil {
				return err
			}

			ui.Success(os.Stdout, "Pointers updated", fmt.Sprintf("%d/%d", latest.Score, latest.Total))
			return nil
		},
	}

	return cmd
}

func parsePuzzleArg(value string) (int, error) {
	day, err := strconv.Atoi(value)
	if err != nil || day < 1 {
		return 0, fmt.Errorf("invalid puzzle number: %s", value)
	}
	return day, nil
}

func parseYearArg(value string) (int, error) {
	year, err := strconv.Atoi(value)
	if err != nil || year < 1000 {
		return 0, fmt.Errorf("invalid year: %s", value)
	}
	return year, nil
}

func parsePartArg(value string) (int, error) {
	part, err := strconv.Atoi(value)
	if err != nil || part < 1 || part > 3 {
		return 0, fmt.Errorf("invalid part: %s", value)
	}
	return part, nil
}

func parsePuzzleFromArgsOrDir(args []string) (int, int, error) {
	if len(args) == 1 {
		day, err := parsePuzzleArg(args[0])
		if err != nil {
			return 0, 0, err
		}
		return yearFlag, day, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return 0, 0, fmt.Errorf("get cwd: %w", err)
	}
	return puzzle.InferFromDir(cwd)
}

func parseRunArgs(args []string, useTestFlag, useInputFlag bool) (int, int, bool, error) {
	useTest := useTestFlag
	useInput := useInputFlag
	var puzzleArg string
	var inputArg string

	switch len(args) {
	case 0:
	case 1:
		if args[0] == "i" || args[0] == "t" {
			inputArg = args[0]
		} else {
			puzzleArg = args[0]
		}
	case 2:
		puzzleArg = args[0]
		inputArg = args[1]
	default:
		return 0, 0, false, fmt.Errorf("invalid arguments: %s", strings.Join(args, " "))
	}

	if inputArg != "" {
		switch inputArg {
		case "i":
			useInput = true
		case "t":
			useTest = true
		default:
			return 0, 0, false, fmt.Errorf("invalid input selector: %s", inputArg)
		}
	}

	if useTest && useInput {
		return 0, 0, false, fmt.Errorf("choose either test or input")
	}
	if !useTest && !useInput {
		return 0, 0, false, fmt.Errorf("choose input: use 'i' or 't'")
	}

	if puzzleArg != "" {
		day, err := parsePuzzleArg(puzzleArg)
		if err != nil {
			return 0, 0, false, err
		}
		return yearFlag, day, useTest, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return 0, 0, false, fmt.Errorf("get cwd: %w", err)
	}
	year, day, err := puzzle.InferFromDir(cwd)
	if err != nil {
		return 0, 0, false, err
	}
	return year, day, useTest, nil
}

func newSetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "setup [year]",
		Aliases: []string{"init"},
		Short:   "Set up a new year directory and configuration",
		Args:    cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			year := time.Now().Year()
			if len(args) == 1 {
				parsedYear, err := parseYearArg(args[0])
				if err != nil {
					return err
				}
				year = parsedYear
			}

			if err := setup.SetupYear(year); err != nil {
				return err
			}

			// Update config with the new current year
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.CurrentYear = year
			if err := config.Save(cfg); err != nil {
				return err
			}

			ui.Success(os.Stdout, "Year setup complete", fmt.Sprintf("%d directory created and configured", year))
			return nil
		},
	}

	return cmd
}
