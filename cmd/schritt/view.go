package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ystsbry/schritt/internal/store"
	"github.com/ystsbry/schritt/internal/tui"
)

func newViewCmd() *cobra.Command {
	var pbi int
	cmd := &cobra.Command{
		Use:   "view [refinement-dir]",
		Short: "リファインメント結果と各レポートをTUIで閲覧する",
		Long: "既存のリファインメント結果ディレクトリ（または --pbi <番号> で最新）を読み込み、\n" +
			"リファインメント内容に加えて、実装レポート（reports/）・検証レポート（verification/）が\n" +
			"あれば一覧に並べてTUIで閲覧します。",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
			return tui.Run(tui.Config{Refinement: ref})
		},
	}
	cmd.Flags().IntVar(&pbi, "pbi", 0, "PBI番号（最新のリファインメントを開く）")
	return cmd
}
