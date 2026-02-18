package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"magento.GO/config"
	productService "magento.GO/service/product"
)

var (
	importFile    string
	importStore   uint16
	importBatch   int
	importAttrSet uint16
	importRawSQL  bool
)

var importCmd = &cobra.Command{
	Use:   "products:import",
	Short: "Import products from CSV into Magento EAV tables",
	Run: func(cmd *cobra.Command, args []string) {
		f, err := os.Open(importFile)
		if err != nil {
			fmt.Printf("Failed to open CSV: %v\n", err)
			return
		}
		defer f.Close()

		db, err := config.NewDB()
		if err != nil {
			fmt.Printf("Database connection failed: %v\n", err)
			return
		}

		res, err := productService.ImportProducts(db, f, productService.ImportOptions{
			StoreID:      importStore,
			BatchSize:    importBatch,
			AttributeSet: importAttrSet,
			RawSQL:       importRawSQL,
		})
		if err != nil {
			fmt.Printf("Import failed: %v\n", err)
			return
		}

		for _, w := range res.Warnings {
			fmt.Printf("  [warn] %s\n", w)
		}

		eavTotal := res.EAVCounts["varchar"] + res.EAVCounts["int"] + res.EAVCounts["decimal"] + res.EAVCounts["text"] + res.EAVCounts["datetime"]
		fmt.Printf(`
=== Import Report ===
CSV rows:       %d
Created:        %d
Updated:        %d
Skipped:        %d
EAV values:     %d (varchar=%d int=%d decimal=%d text=%d datetime=%d)
Stock rows:     %d
Gallery rows:   %d
Price rows:     %d
Mode:           %s
Total time:     %s
  - Processing: %s
  - DB upsert:  %s
=====================
`, res.TotalRows, res.Created, res.Updated, res.Skipped,
			eavTotal, res.EAVCounts["varchar"], res.EAVCounts["int"], res.EAVCounts["decimal"], res.EAVCounts["text"], res.EAVCounts["datetime"],
			res.EAVCounts["stock"], res.EAVCounts["gallery"], res.EAVCounts["price_index"],
			map[bool]string{true: "Raw SQL", false: "GORM ORM"}[importRawSQL],
			res.TotalTime.Round(time.Millisecond),
			res.ProcessTime.Round(time.Millisecond),
			res.DBTime.Round(time.Millisecond))
	},
}

func init() {
	importCmd.Flags().StringVarP(&importFile, "file", "f", "", "CSV file path (required)")
	importCmd.MarkFlagRequired("file")
	importCmd.Flags().Uint16Var(&importStore, "store", 0, "Store ID (default 0)")
	importCmd.Flags().IntVar(&importBatch, "batch-size", 500, "Batch size for DB operations")
	importCmd.Flags().Uint16Var(&importAttrSet, "attribute-set", 4, "Default attribute set ID for new products")
	importCmd.Flags().BoolVar(&importRawSQL, "raw-sql", false, "Use raw SQL for EAV upserts (faster, bypasses GORM ORM)")
	rootCmd.AddCommand(importCmd)
}
