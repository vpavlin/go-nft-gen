package nftgen

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/vpavlin/go-nft-gen/internal/config"
	"github.com/vpavlin/go-nft-gen/internal/generate"
)

var runCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the NFT generation",
	PreRun: toggleDebug,
	Run: func(cmd *cobra.Command, args []string) {
		configFile, err := cmd.Flags().GetString("config")
		if err != nil {
			logrus.Fatal(err)
		}

		c, err := config.Load(configFile)
		if err != nil {
			logrus.Fatal(err)
		}

		output, err := cmd.Flags().GetString("output")
		if err != nil {
			logrus.Fatal(err)
		}

		if output != "" {
			c.OutputDir = output
		}

		g, err := generate.NewGenerate(c)
		if err != nil {
			logrus.Fatal(err)
		}

		n, err := cmd.Flags().GetUint("amount")
		if err != nil {
			logrus.Fatal(err)
		}

		if n > 0 {
			c.N = n
		}

		logrus.Infof("Producing %d NFTS", n)
		err = g.GenerateN(n)
		if err != nil {
			logrus.Fatal(err)
		}

		r, err := cmd.Flags().GetBool("rarities")
		if err != nil {
			logrus.Fatal(err)
		}

		if r {
			err = g.WriteRarities()
			if err != nil {
				logrus.Fatal(err)
			}
		}
	},
}

func init() {
	runCmd.Flags().String("output", "", "Override output dir from config file")
	runCmd.Flags().UintP("amount", "n", 0, "Number of NFTs to generate")
	runCmd.Flags().BoolP("rarities", "r", true, "Calculate trait rarities")

	rootCmd.AddCommand(runCmd)

}
