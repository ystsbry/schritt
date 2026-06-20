package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ystsbry/schritt/internal/implement"
	"github.com/ystsbry/schritt/internal/model"
	"github.com/ystsbry/schritt/internal/store"
)

func newImplementCmd() *cobra.Command {
	var (
		pbi    int
		repos  []string
		engine string
		demo   bool
		mdl    string
		bin    string
		stepN  int
	)
	cmd := &cobra.Command{
		Use:   "implement [refinement-dir]",
		Short: "リファインメント済みの実装計画を、実装ステップごとに実装する",
		Long: "リファインメント結果ディレクトリ（または --pbi <番号> で最新）を読み込み、\n" +
			"実装ステップごとに implement-step skill を起動してコードを実装し、各ステップの\n" +
			"レポート（実装内容・書いた単体テスト）を reports/ 配下に出力します。\n\n" +
			"対象リポジトリは --repo で指定します（省略時は refinement.yml の repo_paths）。",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := store.Home()
			if err != nil {
				return err
			}

			// Resolve the refinement directory.
			dir := ""
			if len(args) == 1 {
				dir = args[0]
			} else if pbi > 0 {
				dir, err = store.LatestRefinementDir(home, pbi)
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("リファインメントディレクトリか --pbi <番号> を指定してください")
			}

			ref, err := store.Load(dir)
			if err != nil {
				return err
			}
			pbiBody, err := store.ReadPBI(dir)
			if err != nil {
				return err
			}

			steps := implementationSteps(ref)
			if len(steps) == 0 {
				return fmt.Errorf("%s に実装ステップがありません", dir)
			}

			// Resolve target repositories.
			repoPaths := repos
			if len(repoPaths) == 0 {
				repoPaths = ref.RepoPaths
			}

			impl, label, err := buildImplementer(engine, demo, mdl, bin, cmd)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "PBI #%d を %s で実装します（%d ステップ）。\n", ref.PBI.Number, label, len(steps))

			for i, st := range steps {
				if stepN > 0 && stepN != i+1 {
					continue
				}
				fmt.Fprintf(out, "\n=== ステップ %d/%d: %s ===\n", i+1, len(steps), st.Title)
				res, err := impl.Implement(cmd.Context(), implement.Input{
					StepTitle: st.Title,
					StepBody:  st.Body,
					PBIBody:   pbiBody,
					RepoPaths: repoPaths,
				})
				if err != nil {
					return fmt.Errorf("ステップ %d (%s): %w", i+1, st.Title, err)
				}
				path, err := store.SaveReport(dir, store.ReportName(st.BodyFile), res.Report)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "レポートを保存: %s\n", path)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&pbi, "pbi", 0, "PBI番号（最新のリファインメントを対象にする）")
	cmd.Flags().StringSliceVar(&repos, "repo", nil, "実装対象リポジトリ（複数可。省略時は refinement.yml の repo_paths）")
	cmd.Flags().StringVar(&engine, "engine", "claude", "使用するAIエンジン (claude | codex)")
	cmd.Flags().BoolVar(&demo, "demo", false, "AIを呼ばずにサンプルレポートを出力する")
	cmd.Flags().StringVar(&mdl, "model", "", "AIに渡すモデル名 (省略時は各CLIの既定)")
	cmd.Flags().StringVar(&bin, "bin", "", "AI実行ファイルのパス (省略時はPATHの claude / codex)")
	cmd.Flags().IntVar(&stepN, "step", 0, "特定の実装ステップ番号だけを実装する（1始まり。0は全ステップ）")
	return cmd
}

// implementationSteps returns the steps of the implementation section, in order.
func implementationSteps(ref *model.Refinement) []model.Step {
	for _, s := range ref.Sections {
		if s.ID == model.SectionImplementation {
			return s.Steps
		}
	}
	return nil
}

// buildImplementer selects an Implementer from the engine/demo flags and
// returns a human label for messages.
func buildImplementer(engine string, demo bool, mdl, bin string, cmd *cobra.Command) (implement.Implementer, string, error) {
	switch {
	case demo:
		return implement.DemoImplementer{}, "demo", nil
	case engine == "claude":
		return &implement.ClaudeImplementer{Model: mdl, Bin: bin, Stream: cmd.OutOrStdout()}, "claude", nil
	case engine == "codex":
		return &implement.CodexImplementer{Model: mdl, Bin: bin, Stream: cmd.OutOrStdout()}, "codex", nil
	default:
		return nil, "", fmt.Errorf("unknown --engine %q (want claude or codex)", engine)
	}
}
