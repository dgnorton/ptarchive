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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: runLs,
}

func init() {
	rootCmd.AddCommand(lsCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// lsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// lsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runLs(cmd *cobra.Command, args []string) {
	archives := getArchiveList()

	for _, a := range archives {
		fmt.Printf("%s\t%d\n", a.Filename, a.Filesize)
	}
}

func getArchiveList() ArchiveInfos {
	tok := mustPapertrailAPIToken()

	url := "https://papertrailapp.com/api/v1/archives.json"
	req, err := http.NewRequest("GET", url, nil)
	check(err)
	req.Header.Set("X-Papertrail-Token", tok)

	c := &http.Client{}
	resp, err := c.Do(req)
	check(err)
	checkHTTP(resp, req)
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	check(err)

	archives := ArchiveInfos{}
	err = json.Unmarshal(b, &archives)
	check(err)

	return archives
}

func mustPapertrailAPIToken() string {
	tok := os.Getenv("PAPERTRAIL_API_TOK")
	if tok == "" {
		check(fmt.Errorf("PAPERTRAIL_API_TOK env var required"))
	}
	return tok
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func checkHTTP(resp *http.Response, req *http.Request) {
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("http status: %v", resp.StatusCode)
		check(err)
	}
}

type ArchiveDownload struct {
	Href string `json:"href"`
}

type ArchiveLinks struct {
	Download ArchiveDownload `json:"download"`
}

type ArchiveInfo struct {
	Links             *ArchiveLinks `json:"_links"`
	DurationFormatted string        `json:"duration_formatted"`
	End               time.Time     `json:"end"`
	Filename          string        `json:"filename"`
	Filesize          int           `json:"filesize"`
	Start             time.Time     `json:"start"`
	StartFormatted    string        `json:"start_formatted"`
}

func (ai *ArchiveInfo) Overlaps(start, end time.Time) bool {
	as, ae := ai.Start.UnixNano(), ai.End.UnixNano()
	bs, be := start.UnixNano(), end.UnixNano()

	return as <= be && ae >= bs
}

type ArchiveInfos []*ArchiveInfo

type ArchiveInfoMatchesFn func(*ArchiveInfo) bool

func (a ArchiveInfos) Matches(fn ArchiveInfoMatchesFn) ArchiveInfos {
	archives := ArchiveInfos{}
	for _, ai := range a {
		if fn(ai) {
			archives = append(archives, ai)
		}
	}
	return archives
}

func (a ArchiveInfos) Len() int {
	return len(a)
}

func (a ArchiveInfos) Size() int {
	n := 0
	for _, ai := range a {
		n += ai.Filesize
	}
	return n
}

func ArchivesOverlap(start, end time.Time) ArchiveInfoMatchesFn {
	return func(ai *ArchiveInfo) bool {
		return ai.Overlaps(start, end)
	}
}
