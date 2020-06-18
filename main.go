package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/cloudflare/cloudflare-go"
	"gopkg.in/ini.v1"
)

type bucketObjectList struct {
	Contents []bucketObject
}

type bucketObject struct {
	Key  string
	ETag string
}

var (
	showVersion bool
	version     = "dev"
	dryRun      bool
	baseURL     string
	profile     string
	urls        = make(map[string]struct{})
)

func init() {
	flag.BoolVar(&showVersion, "version", false, "print version number")
	flag.BoolVar(&dryRun, "dryrun", false, "run command without making changes")
	flag.StringVar(&baseURL, "baseurl", os.Getenv("S3CF_CF_BASE_URL"), "used to build URLS to delete from Cloudflare's cache (eg https://example.com)")
	flag.StringVar(&profile, "profile", os.Getenv("AWS_PROFILE"), "name of the AWS profile used to authenticate with the API")
	flag.Usage = usage
}

func main() {
	flag.Parse()

	if profile == "" {
		profile = "default"
	}

	// overwrite cloudflare credentials with values from environment
	cfKey := os.Getenv("S3CF_CF_API_KEY")
	cfEmail := os.Getenv("S3CF_CF_API_EMAIL")
	cfZone := os.Getenv("S3CF_CF_API_ZONE")

	// attempt to load cloudflare credentials from disk
	sharedCredentialsPath := os.Getenv("HOME") + "/.aws/credentials"
	cfg, err := ini.Load(sharedCredentialsPath)
	if err == nil {
		if cfKey == "" {
			cfKey = cfg.Section(profile).Key("cf_api_key").String()
		}
		if cfEmail == "" {
			cfEmail = cfg.Section(profile).Key("cf_api_email").String()
		}
		if cfZone == "" {
			cfZone = cfg.Section(profile).Key("cf_api_zone").String()
		}
		if baseURL == "" {
			baseURL = cfg.Section(profile).Key("cf_base_url").String()
		}
	} else {
		fmt.Println("Could not load credentials from " + sharedCredentialsPath)
	}

	if showVersion {
		fmt.Printf("%s %s (runtime: %s)\n", os.Args[0], version, runtime.Version())
		os.Exit(0)
	}

	args := flag.Args()
	fmt.Println(args)
	if len(args) != 2 {
		flag.Usage()
		os.Exit(2)
	}

	var localPath, bucketPath string

	if _, err := os.Stat(args[0]); !os.IsNotExist(err) {
		localPath = args[0]
	} else {
		fmt.Println("the first argument is not a valid path")
		os.Exit(2)
	}

	if strings.Contains(args[1], "s3://") {
		bucketPath = args[1]
	} else {
		fmt.Println("the second argument is not a valid <S3Uri>, should start with s3://")
		os.Exit(2)
	}

	sync(localPath, bucketPath)

	if cfKey != "" && cfEmail != "" && cfZone != "" {
		purge(cfKey, cfEmail, cfZone, bucketPath)
	} else {
		fmt.Println("Skipped Cloudflare cache purge because the environment variables are not set")
	}
}

func usage() {
	fmt.Println("s3cf: error: the following arguments are required: paths")
	fmt.Println("usage: s3cf [OPTIONS] <LocalPath> <S3Uri>")
	fmt.Fprintln(os.Stderr, "\nOPTIONS:")
	flag.PrintDefaults()
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "ENVIRONMENT:")
	fmt.Fprintln(os.Stderr, "  AWS_PROFILE              the name of an AWS profile to use")
	fmt.Fprintln(os.Stderr, "  AWS_ACCESS_KEY_ID        the AWS Key ID Key with S3 Sync permissions")
	fmt.Fprintln(os.Stderr, "  AWS_SECRET_ACCESS_KEY    the AWS Secret Key with S3 Sync permissions")

	fmt.Fprintln(os.Stderr, "  S3CF_CF_API_KEY          the Cloudflare API Key used to authenticate when purging cache")
	fmt.Fprintln(os.Stderr, "  S3CF_CF_API_EMAIL        the Cloudflare Email used to authenticate when purging cache")
	fmt.Fprintln(os.Stderr, "  S3CF_CF_API_ZONE         the Cloudflare Zone ID used to authenticate when purging cache")
	fmt.Fprintln(os.Stderr, "  S3CF_CF_BASE_URL         used to build URLS to delete from Cloudflare's cache (eg https://example.com)")
}

func sync(localPath, bucketPath string) {
	fmt.Println("Syncing files in folder: '" + localPath + "' with files in S3 bucket: '" + bucketPath + "'")

	args := []string{"s3", "sync", localPath, bucketPath, "--delete", "--size-only", "--exclude=*.DS_Store", "--profile", profile}
	if dryRun {
		args = append(args, "--dryrun")
	}
	cmd := exec.Command("aws", args...)
	process(cmd)

	bucketObjects := list(bucketPath)
	for _, obj := range bucketObjects.Contents {
		fileHash, err := hashFileMD5(localPath + "/" + obj.Key)
		if err != nil {
			log.Fatal(err)
		}
		// if MD5 hash of local file does not match
		// copy the local file to the S3 bucket
		remoteHash := trimQuotes(obj.ETag)
		if fileHash != remoteHash {
			filePath := localPath + "/" + obj.Key
			remoteFilePath := bucketPath + "/" + obj.Key
			// no need to copy if it was already copied with previous sync
			if _, ok := urls[remoteFilePath]; ok {
				continue
			}
			args := []string{"s3", "cp", filePath, remoteFilePath, "--profile", profile}
			if dryRun {
				args = append(args, "--dryrun")
			}
			cmd = exec.Command("aws", args...)
			process(cmd)
		}
	}
}

func process(cmd *exec.Cmd) {
	// sync images files
	// Get a pipe to read from standard out
	r, _ := cmd.StdoutPipe()
	// Use the same pipe for standard error
	cmd.Stderr = cmd.Stdout
	// Make a new channel which will be used to ensure we get all output
	done := make(chan struct{})
	// Create a scanner which scans r in a line-by-line fashion
	scanner := bufio.NewScanner(r)
	// Use the scanner to scan the output line by line and log it
	// It's running in a goroutine so that it doesn't block
	go func() {
		// Read line by line and process it
		for scanner.Scan() {
			line := scanner.Text()
			s3Uri := extractS3Uri(line)
			if s3Uri != "" {
				urls[s3Uri] = struct{}{}
			}
			fmt.Println(line)
		}
		// We're all done, unblock the channel
		done <- struct{}{}
	}()
	// Start the command and check for errors
	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	// Wait for all output to be processed
	<-done
	// Wait for the command to finish
	err = cmd.Wait()
	if err != nil {
		log.Fatal(err)
	}
}

func list(bucketPath string) bucketObjectList {
	bucket := bucketName(bucketPath)
	cmd := exec.Command("aws", "s3api", "list-objects-v2", "--bucket", bucket)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = cmd.Stdout
	err := cmd.Run()
	if err != nil {
		fmt.Println(out.String())
		log.Fatal(err)
	}
	// parse into
	var objList bucketObjectList
	err = json.Unmarshal([]byte(out.String()), &objList)
	if err != nil {
		log.Fatal(err)
	}
	return objList
}

func purge(apiKey, apiEmail, zoneID, bucketPath string) {
	fmt.Println("Purging URLs from CloudFlare Cache")
	api, err := cloudflare.New(apiKey, apiEmail)
	if err != nil {
		log.Fatal(err)
	}
	// purge URLS in batches of 30
	chunkSize := 30
	batchKeys := make([]string, 0, chunkSize)
	process := func() {
		pcr := cloudflare.PurgeCacheRequest{Files: batchKeys}
		r, err := api.PurgeCache(zoneID, pcr)
		if err != nil {
			log.Fatal(err)
		}
		if r.Success {
			fmt.Println("Purged:", batchKeys)
		} else {
			fmt.Println("Purge Failed:", batchKeys)
		}
		batchKeys = batchKeys[:0]
	}

	for k := range urls {
		url := strings.Replace(k, bucketPath, baseURL, 1)
		url = strings.Replace(url, "index.html", "", 1)
		batchKeys = append(batchKeys, url)
		if len(batchKeys) == chunkSize {
			process()
		}
	}
	// Process last, potentially incomplete batch
	if len(batchKeys) > 0 {
		process()
	}
}

func bucketName(s3Uri string) string {
	if strings.HasPrefix(s3Uri, "s3://") {
		return strings.TrimPrefix(s3Uri, "s3://")
	}
	return s3Uri
}

func extractS3Uri(line string) string {
	split := strings.Split(line, "s3://")
	if len(split) > 1 {
		return "s3://" + split[1]
	}
	return ""
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func hashFileMD5(filePath string) (string, error) {
	var returnMD5String string

	file, err := os.Open(filePath)
	if err != nil {
		return returnMD5String, err
	}

	defer file.Close()
	hash := md5.New()

	if _, err := io.Copy(hash, file); err != nil {
		return returnMD5String, err
	}

	hashInBytes := hash.Sum(nil)[:16]
	returnMD5String = hex.EncodeToString(hashInBytes)
	return returnMD5String, nil
}
