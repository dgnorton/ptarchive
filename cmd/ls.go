package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
)

// lsCmd represents the ls command
var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "lists archives available for download from papertrail",
	Long:  ``,
	Run:   runLs,
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
		fmt.Printf("%s\t%s\n", a.Filename, a.Filesize)
	}
}

func getArchiveList() ArchiveInfos {
	tok := mustPapertrailAPIToken()

	url := "https://papertrailapp.com/api/v1/archives.json"
	req, err := http.NewRequest("GET", url, nil)
	checkm("getting archive list: new request", err)
	req.Header.Set("X-Papertrail-Token", tok)

	c := &http.Client{}
	resp, err := c.Do(req)
	checkm("getting archive list: making HTTP request", err)
	checkHTTP(resp, req)
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	checkm("getting archive list: reading HTTP response", err)

	archives := ArchiveInfos{}
	err = json.Unmarshal(b, &archives)
	//fmt.Println(string(b))
	checkm("getting archive list: unmarshaling response JSON", err)

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

func checkm(msg string, err error) {
	if err != nil {
		fmt.Printf("%s: %s:\n", msg, err)
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
	Filesize          string        `json:"filesize"`
	fsize             int
	Start             time.Time `json:"start"`
	StartFormatted    string    `json:"start_formatted"`
}

func (ai *ArchiveInfo) Overlaps(start, end time.Time) bool {
	as, ae := ai.Start.UnixNano(), ai.End.UnixNano()
	bs, be := start.UnixNano(), end.UnixNano()

	return as <= be && ae >= bs
}

func (ai *ArchiveInfo) Size() (int, error) {
	if ai.fsize > 0 {
		return ai.fsize, nil
	}

	v, err := strconv.Atoi(ai.Filesize)
	if err == nil {
		ai.fsize = v
	}

	return v, err
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

func (a ArchiveInfos) Size() (int, error) {
	n := 0
	for _, ai := range a {
		sz, err := ai.Size()
		if err != nil {
			return 0, err
		}
		n += sz
	}
	return n, nil
}

func ArchivesOverlap(start, end time.Time) ArchiveInfoMatchesFn {
	return func(ai *ArchiveInfo) bool {
		return ai.Overlaps(start, end)
	}
}
