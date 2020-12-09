package cmd

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"github.com/tidwall/transform"
)

// cpCmd represents the cp command
var cpCmd = &cobra.Command{
	Use:   "cp",
	Short: "downloads archives from papertrail",
	Long: `Concurrently downloads archives from papertrail for a specified range of time
and optionally filters the log files for a given string or regex pattern.`,
	Run: runCp,
}

var (
	outDir     string
	startTime  string
	endTime    string
	dry        bool
	unzip      bool
	pattern    string
	substr     string
	concurrent int
	cpuprofile string
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
	cpCmd.Flags().StringVarP(&pattern, "pattern", "p", "", "Regexp to filter logs. Will automatically unzip files.")
	cpCmd.Flags().StringVar(&substr, "substr", "", "Substring search pattern to filter logs. Will automatically unzip files.")
	cpCmd.Flags().StringVar(&cpuprofile, "cpuprofile", "", "Path to write CPU performance profile")
	cpCmd.Flags().BoolVarP(&dry, "dry", "d", false, "Dry run lists archives that would be downloaded")
	cpCmd.Flags().IntVar(&concurrent, "concurrent", 4, "Number of concurrent downloads")

}

func runCp(cmd *cobra.Command, args []string) {
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	start := mustParseTime(time.RFC3339, startTime)
	end := mustParseTime(time.RFC3339, endTime)

	fmt.Printf("fetching list of archives between %s and %s...\n", start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339))

	archives := getArchiveList()
	archives = archives.Matches(ArchivesOverlap(start, end))

	sz, err := archives.Size()
	checkm("getting archives' size", err)
	fmt.Printf("found %d matching archive files totalling %d bytes\n", archives.Len(), sz)

	if archives.Len() == 0 {
		return
	}

	if outDir == "" {
		dir, err := ioutil.TempDir("", "ptarchive")
		check(err)
		outDir = dir
	}
	fmt.Printf("output directory: %s\n", outDir)

	if pattern != "" {
		fmt.Printf("files will contain only lines matching regexp: %s\n", pattern)
		unzip = true
	}

	if substr != "" {
		fmt.Printf("files will contain only lines matching substring: %s\n", substr)
		unzip = true
	}

	if unzip {
		fmt.Println("files will be automatically unzipped")
	}

	if concurrent > len(archives) {
		concurrent = len(archives)
	}
	fmt.Printf("%d files will be downloaded concurrently\n", concurrent)

	if dry {
		fmt.Println("Dry run - these files would be downloaded...")
		for _, a := range archives {
			fmt.Printf("%s\t%s\n", a.Filename, a.Filesize)
		}
		return

	}

	queue := make(chan *ArchiveInfo)
	wg := &sync.WaitGroup{}

	for n := 0; n < concurrent; n++ {
		wg.Add(1)
		go procQueue(queue, wg)
	}

	for _, a := range archives {
		queue <- a
	}

	close(queue)
	wg.Wait()
}

func procQueue(queue chan *ArchiveInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	for a := range queue {
		outPath := path.Join(outDir, a.Filename)
		// If we're unzipping the archive, remove the .gz extension.
		if unzip {
			outPath = strings.TrimSuffix(outPath, path.Ext(a.Filename))
		}
		fmt.Printf("%s: started\n", outPath)
		if err := getArchive(a, pattern, substr, outPath); err != nil {
			fmt.Printf("%s: %s\n", outPath, err)
			continue
		}
		fmt.Printf("%s: finished\n", outPath)
	}
}

func getArchive(a *ArchiveInfo, pattern, substr, outPath string) error {
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

	r := io.Reader(resp.Body)

	if unzip {
		r, err = gzip.NewReader(r)
		if err != nil {
			return err
		}
	}

	if substr != "" {
		r = substrFilter(r, substr)
	}

	if pattern != "" {
		r = regexpFilter(r, pattern)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, r)

	return err
}

func regexpFilter(r io.Reader, pattern string) io.Reader {
	re := regexp.MustCompile(pattern)
	br := bufio.NewReader(r)
	return transform.NewTransformer(func() ([]byte, error) {
		for {
			line, err := br.ReadBytes('\n')
			if matched := re.Match(line); matched {
				return line, err
			}
			if err != nil {
				return nil, err
			}
		}
	})
}

func substrFilter(r io.Reader, substr string) io.Reader {
	br := bufio.NewReader(r)
	return transform.NewTransformer(func() ([]byte, error) {
		for {
			line, err := br.ReadBytes('\n')
			if strings.Contains(string(line), substr) {
				return line, err
			}
			if err != nil {
				return nil, err
			}
		}
	})
}

func mustParseTime(layout, value string) time.Time {
	t, err := time.Parse(layout, value)
	check(err)
	return t
}
