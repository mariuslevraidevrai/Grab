package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/schollz/progressbar/v3"
)

var (
	Version   = "v0.0.1"
	redText   = color.New(color.FgRed).SprintFunc()
	greenText = color.New(color.FgGreen).SprintFunc()
	cyanText  = color.New(color.FgCyan).SprintFunc()

	httpClient = &http.Client{
		Transport: &http.Transport{
			ResponseHeaderTimeout: 20 * time.Second,
			IdleConnTimeout:       30 * time.Second,
		},
	}
)

func main() {
	output := flag.String("o", ".", "Output directory or path")
	urlFlag := flag.String("u", "", "URL to download")
	chunksFlag := flag.Int("c", 4, "Number of chunks/parts for the download")
	versionFlag := flag.Bool("v", false, "Print current version")
	flag.BoolVar(versionFlag, "version", false, "Print current version")
	flag.Parse()

	banner := `
   _____           _     
  / ____|         | |    
 | |  __ _ __ __ _| |__  
 | | |_ | '__/ _` + "`" + ` | '_ \ 
 | |__| | | | (_| | |_) |
  \_____|_|  \__,_|_.__/ 
`

	if *versionFlag {
		fmt.Println(banner)
		fmt.Printf("Grab %s\n", Version)
		fmt.Println("\nYet another file downloader")
		fmt.Println("For more info visit github.com/mariuslevraidevrai/grab")
		return
	}

	var rawURL string

	if *urlFlag != "" {
		rawURL = *urlFlag
	} else if len(flag.Args()) > 0 {
		rawURL = flag.Args()[0]
	} else {
		fmt.Println("Usage: grab [options] <URL> or grab -u <URL> -o <PATH>")
		fmt.Println("Options:")
		flag.PrintDefaults()
		return
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		fmt.Println(redText("[!]") + " Invalid or unsupported URL")
		return
	}
	targetURL := parsedURL.String()

	chunks := *chunksFlag
	if chunks <= 0 {
		chunks = 4
	}

	defaultFileName := path.Base(parsedURL.Path)

	if defaultFileName == "." || defaultFileName == "/" || defaultFileName == "" {
		fmt.Println(redText("[!]") + " Invalid file name")
		return
	}

	var finalPath string
	info, err := os.Stat(*output)
	if err == nil && info.IsDir() {
		finalPath = filepath.Join(*output, defaultFileName)
	} else {
		finalPath = *output
	}

	if err := os.MkdirAll(filepath.Dir(finalPath), 0755); err != nil {
		fmt.Println(redText("[!]") + " Error creating directory:", err)
		return
	}

	respond, err := httpClient.Head(targetURL)
	if err != nil {
		fmt.Println(redText("[!]") + " Head request error:", err)
		return
	}
	statusCode := respond.StatusCode
	statusText := respond.Status
	acceptRanges := respond.Header.Get("Accept-Ranges")
	contentLength := respond.Header.Get("Content-Length")
	respond.Body.Close()

	if statusCode != http.StatusOK {
		fmt.Printf("%s HTTP Error: %s\n", redText("[!]"), statusText)
		return
	}

	if acceptRanges != "bytes" {
		fmt.Println("[" + cyanText("i") + "] Server does not support parallel downloading. Falling back to single-thread...")
		download(targetURL, finalPath)
		return
	}

	fileSize, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil || fileSize <= 0 {
		fmt.Println(redText("[!]") + " Invalid file size:", err)
		return
	}

	if int64(chunks) > fileSize {
		chunks = 1
	}

	file, err := os.OpenFile(finalPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println(redText("[!]") + " File creation error:", err)
		return
	}
	defer file.Close()

	if err := file.Truncate(fileSize); err != nil {
		fmt.Println(redText("[!]") + " File sizing error:", err)
		return
	}

	var completedChunks int32 = 0

	bar := progressbar.NewOptions64(
		fileSize,
		progressbar.OptionSetDescription(fmt.Sprintf("%s Downloading", cyanText(fmt.Sprintf("[0/%d]", chunks)))),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(15),
		progressbar.OptionThrottle(65),
		progressbar.OptionShowCount(),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
	)

	chunkSize := fileSize / int64(chunks)
	var wg sync.WaitGroup
	var barMutex sync.Mutex
	errCh := make(chan error, chunks)

	startTime := time.Now()

	for i := 0; i < chunks; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1

		if i == chunks-1 {
			end = fileSize - 1
		}

		wg.Add(1)
		go downloadByChunk(targetURL, start, end, file, bar, &wg, &completedChunks, chunks, errCh, &barMutex)
	}

	wg.Wait()
	close(errCh)

	failed := false
	for e := range errCh {
		if e != nil {
			fmt.Println(redText("[!]") + " " + e.Error())
			failed = true
		}
	}

	if failed {
		fmt.Println(redText("[!]") + " Download failed, file may be incomplete or corrupted")
		return
	}

	_ = bar.Finish()
	fmt.Fprintln(os.Stderr)

	duration := time.Since(startTime)
	seconds := duration.Seconds()
	if seconds <= 0 {
		seconds = 0.001
	}
	speed := float64(fileSize) / seconds / (1000 * 1000)

	printSummary(finalPath, fileSize, duration, speed)
}