# ptarchive
Command line tool for downloading, unzipping, and filtering papertrail archives

### Features
* Control which archives are downloaded based on date range (`--start` & `--end`)
* Automatically unzip downloaded archives (if either `--substr` or `--pattern`)
* Filter content of archives by sub-string search (faster) or regexp (slower)
  * Archives are filtered as they're streamed to reduce disk usage
* Dry run shows details of what would be performed without actually doing it (`--dry`)
  * Summary, including number of files to be downloaded and total size in bytes
  * List of files to be downloaded
  * Output directory
  * etc.
* Concurrent archive download and processing (`--concurrent`)

### Install
You'll need the Go tools (compiler) installed. Then run:
```
go install github.com/dgnorton/ptarchive@latest
```

### Usage
`ptarchive` requires a Papertrail API token.
```
export PAPERTRAIL_API_TOK=your_token_here
```
#### Command line help:
```
ptarchive -h
```
```
ptarchive <command> -h
```

#### List archives available:
```
ptarchive ls
```
#### Download / copy archive files:
Usually, before downloading, you'll want to do a "dry run" to see what archives will be downloaded and processed. Make sure the specified command line options are going to do what is expected before running this potentially lengthy operation.
```
ptarchive cp -d

fetching list of archives between 2019-02-10T01:10:01Z and 2019-02-10T02:10:01Z...
found 0 matching archive files totalling 0 bytes
```
The `-d` or `--dry` flags will make it a dry run. Running it with no other parameters is an easy way to get a reminder of the date format, as shown in the example above.

Dry run example with concurrency bumped up to 8, a start and end date range, and sub-string filtering:
```
ptarchive cp --concurrent 8 -s 2019-01-21T00:00:00Z -e 2019-01-30T00:00:00Z --substr "prod-abc123-eu-west-1-data" -d

fetching list of archives between 2019-01-21T00:00:00Z and 2019-01-30T00:00:00Z...
found 217 matching archive files totalling 62831924861 bytes
output directory: /tmp/ptarchive455907957
files will contain only lines matching substring: prod-abc123-eu-west-1-data
files will be automatically unzipped
8 files will be downloaded concurrently
Dry run - these files would be downloaded...
2019-01-30-00.tsv.gz    311489677
2019-01-29-23.tsv.gz    274523994
2019-01-29-22.tsv.gz    279747796
2019-01-29-21.tsv.gz    294226547
2019-01-29-20.tsv.gz    296502765
2019-01-29-19.tsv.gz    281413200
<snip>
```
Once the dry run output looks good, re-run the same command without the `-d`.

Progress will be displayed as downloads start and finish.
