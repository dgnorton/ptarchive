// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"
)

// cpCmd represents the cp command
var cpCmd = &cobra.Command{
	Use:   "cp",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: runCp,
}

var (
	outDir    string
	startTime string
	endTime   string
	dry       bool
	filter    string
)

func init() {
	rootCmd.AddCommand(cpCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// cpCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// cpCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	startTime = time.Now().Add(time.Hour * -1).UTC().Format(time.RFC3339)
	endTime = time.Now().UTC().Format(time.RFC3339)

	cpCmd.Flags().StringVarP(&outDir, "outdir", "o", "", "Output directory")
	cpCmd.Flags().StringVarP(&startTime, "start", "s", startTime, "Start of time range")
	cpCmd.Flags().StringVarP(&endTime, "end", "e", endTime, "End of time range")
	cpCmd.Flags().BoolVarP(&dry, "dry", "d", false, "Dry run lists archives that would be downloaded")
}

func runCp(cmd *cobra.Command, args []string) {
	start := mustParseTime(time.RFC3339, startTime)
	end := mustParseTime(time.RFC3339, endTime)

	archives := getArchiveList()
	archives = archives.Matches(ArchivesOverlap(start, end))

	if dry {
		fmt.Println("Dry run - these files would be downloaded...")
	}

	if outDir == "" {
		dir, err := ioutil.TempDir("", "ptarchive")
		check(err)
		outDir = dir
	}

	for _, a := range archives {
		if dry {
			fmt.Printf("%s\t%d\n", a.Filename, a.Filesize)
			continue
		}

		outPath := path.Join(outDir, a.Filename)
		fmt.Printf("%s: ", outPath)
		result := "success"
		if err := getArchive(a, outDir); err != nil {
			result = err.Error()
		}
		fmt.Printf("%s\n", result)
	}
}

func getArchive(a *ArchiveInfo, outDir string) error {
	tok := mustPapertrailAPIToken()

	url := a.Links.Download.Href
	req, err := http.NewRequest("GET", url, nil)
	check(err)
	req.Header.Set("X-Papertrail-Token", tok)

	c := &http.Client{}
	resp, err := c.Do(req)
	check(err)
	checkHTTP(resp, req)
	defer resp.Body.Close()

	f, err := os.Create(path.Join(outDir, a.Filename))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)

	return err
}

func mustParseTime(layout, value string) time.Time {
	t, err := time.Parse(layout, value)
	check(err)
	return t
}
