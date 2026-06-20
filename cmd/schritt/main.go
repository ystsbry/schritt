package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ystsbry/schritt/internal/refine"
	"github.com/ystsbry/schritt/internal/store"
	"github.com/ystsbry/schritt/internal/tui"
)

// These are overwritten at release time via -ldflags.
var (
	version = "0.1.0-dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schritt",
		Short: "PBIからリファインメント→実装→PRまでを段階的に進めるツール",
		Long: "schritt は PBI を起点に、リファインメント・実装計画レビュー・実装・検証・PR作成を\n" +
			"段階的に進めるためのツールです。現在はリファインメント段階を提供します。",
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newRefinementCmd())
	cmd.AddCommand(newImplementCmd())
	return cmd
}

func newRefinementCmd() *cobra.Command {
	var (
		demo   bool
		engine string
		mdl    string
		bin    string
	)
	cmd := &cobra.Command{
		Use:   "refinement",
		Short: "PBIを貼り付けてリファインメントを実行し、結果をTUIで確認する",
		Long: "起動するとPBIのマークダウンを貼り付ける画面が開きます。PBI番号と本文を入力し\n" +
			"Ctrl+R でAIによるリファインメントを実行すると、POへの確認事項・実装内容・\n" +
			"単体/統合テストのテストケースをTUIで確認できます。\n\n" +
			"AIエンジンは --engine で claude / codex を選べます。\n" +
			"結果は ~/.schritt/pbi-{番号}/{日時}/ に refinement.yml + markdown として保存されます。",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := store.Home()
			if err != nil {
				return err
			}

			var refiner refine.Refiner
			model := mdl
			switch {
			case demo:
				refiner = refine.DemoRefiner{}
				model = "demo"
			case engine == "claude":
				refiner = &refine.ClaudeRefiner{Model: mdl, Bin: bin}
			case engine == "codex":
				refiner = &refine.CodexRefiner{Model: mdl, Bin: bin}
			default:
				return fmt.Errorf("unknown --engine %q (want claude or codex)", engine)
			}

			return tui.Run(tui.Config{
				Refiner: refiner,
				Home:    home,
				Model:   model,
			})
		},
	}
	cmd.Flags().BoolVar(&demo, "demo", false, "AIを呼ばずにサンプル結果でTUIを試す")
	cmd.Flags().StringVar(&engine, "engine", "claude", "使用するAIエンジン (claude | codex)")
	cmd.Flags().StringVar(&mdl, "model", "", "AIに渡すモデル名 (省略時は各CLIの既定)")
	cmd.Flags().StringVar(&bin, "bin", "", "AI実行ファイルのパス (省略時はPATHの claude / codex)")
	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print schritt version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "schritt %s (commit %s, built %s)\n", version, commit, date)
			return nil
		},
	}
}
