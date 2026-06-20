package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ystsbry/schritt/internal/model"
	"github.com/ystsbry/schritt/internal/store"
	"github.com/ystsbry/schritt/internal/verify"
)

func newVerifyCmd() *cobra.Command {
	var (
		pbi    int
		url    string
		repos  []string
		engine string
		demo   bool
		mdl    string
		bin    string
		stepN  int
	)
	cmd := &cobra.Command{
		Use:   "verify [refinement-dir]",
		Short: "E2Eシナリオに沿って CDP経由でChromeを操作し、動作確認する",
		Long: "リファインメント結果のE2Eシナリオ（統合テスト）ごとに verify-e2e skill を起動し、\n" +
			"CDP経由でChromeを操作して動作確認します。各シナリオの合否レポートとスクリーンショットを\n" +
			"verification/ 配下に出力します。\n\n" +
			"検証対象アプリは --url で指定します（起動済みであること）。",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !demo && url == "" {
				return fmt.Errorf("--url を指定してください（検証対象アプリのURL）")
			}
			home, err := store.Home()
			if err != nil {
				return err
			}

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

			scenarios := integrationScenarios(ref)
			if len(scenarios) == 0 {
				return fmt.Errorf("%s にE2Eシナリオ（統合テスト）がありません", dir)
			}

			repoPaths := repos
			if len(repoPaths) == 0 {
				repoPaths = ref.RepoPaths
			}

			vfr, label, err := buildVerifier(engine, demo, mdl, bin, cmd)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "PBI #%d を %s で動作確認します（%d シナリオ）。\n", ref.PBI.Number, label, len(scenarios))

			for i, sc := range scenarios {
				if stepN > 0 && stepN != i+1 {
					continue
				}
				fmt.Fprintf(out, "\n=== シナリオ %d/%d: %s ===\n", i+1, len(scenarios), sc.Title)
				res, err := vfr.Verify(cmd.Context(), verify.Input{
					ScenarioTitle: sc.Title,
					ScenarioBody:  sc.Body,
					PBIBody:       pbiBody,
					AppURL:        url,
					RepoPaths:     repoPaths,
				})
				if err != nil {
					return fmt.Errorf("シナリオ %d (%s): %w", i+1, sc.Title, err)
				}
				shots := make([]store.Screenshot, 0, len(res.Screenshots))
				for _, s := range res.Screenshots {
					shots = append(shots, store.Screenshot{Name: s.Name, Data: s.Data})
				}
				path, err := store.SaveVerification(dir, store.ReportName(sc.BodyFile), res.Report, shots)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "レポートを保存: %s（スクショ %d枚）\n", path, len(shots))
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&pbi, "pbi", 0, "PBI番号（最新のリファインメントを対象にする）")
	cmd.Flags().StringVar(&url, "url", "", "検証対象アプリのURL（例: http://localhost:3000）")
	cmd.Flags().StringSliceVar(&repos, "repo", nil, "参照するリポジトリ（複数可。省略時は refinement.yml の repo_paths）")
	cmd.Flags().StringVar(&engine, "engine", "claude", "使用するAIエンジン (claude | codex)")
	cmd.Flags().BoolVar(&demo, "demo", false, "ブラウザ/AIを使わずサンプルレポートを出力する")
	cmd.Flags().StringVar(&mdl, "model", "", "AIに渡すモデル名 (省略時は各CLIの既定)")
	cmd.Flags().StringVar(&bin, "bin", "", "AI実行ファイルのパス (省略時はPATHの claude / codex)")
	cmd.Flags().IntVar(&stepN, "step", 0, "特定のシナリオ番号だけを検証する（1始まり。0は全シナリオ）")
	return cmd
}

// integrationScenarios returns the E2E scenario steps of the integration
// section, in order.
func integrationScenarios(ref *model.Refinement) []model.Step {
	for _, s := range ref.Sections {
		if s.ID == model.SectionIntegrationTests {
			return s.Steps
		}
	}
	return nil
}

// buildVerifier selects a Verifier from the engine/demo flags.
func buildVerifier(engine string, demo bool, mdl, bin string, cmd *cobra.Command) (verify.Verifier, string, error) {
	switch {
	case demo:
		return verify.DemoVerifier{}, "demo", nil
	case engine == "claude":
		return &verify.ClaudeVerifier{Model: mdl, Bin: bin, Stream: cmd.OutOrStdout()}, "claude", nil
	case engine == "codex":
		return &verify.CodexVerifier{Model: mdl, Bin: bin, Stream: cmd.OutOrStdout()}, "codex", nil
	default:
		return nil, "", fmt.Errorf("unknown --engine %q (want claude or codex)", engine)
	}
}
