package main

import (
	"context"
	"flag"
	"fmt"
	"net"
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

const (
	userAgent = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	maxChunks = 32
)

var (
	Version   = "v0.0.2"
	redText   = color.New(color.FgRed).SprintFunc()
	greenText = color.New(color.FgGreen).SprintFunc()
	cyanText  = color.New(color.FgCyan).SprintFunc()

	httpClient = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   15 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   15 * time.Second,
			ResponseHeaderTimeout: 20 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       30 * time.Second,
			MaxIdleConnsPerHost:   maxChunks,
			DisableCompression:    true,
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
	if chunks > maxChunks {
		chunks = maxChunks
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

	req, err := http.NewRequest(http.MethodHead, targetURL, nil)
	if err != nil {
		fmt.Println(redText("[!]") + " Request error:", err)
		return
	}
	req.Header.Set("User-Agent", userAgent)

	respond, err := httpClient.Do(req)
	if err != nil {
		fmt.Println(redText("[!]") + " Request error:", err)
		return
	}
	statusCode := respond.StatusCode
	statusText := respond.Status
	contentLength := respond.Header.Get("Content-Length")
	acceptRanges := respond.Header.Get("Accept-Ranges")
	respond.Body.Close()

	if statusCode == http.StatusMethodNotAllowed || statusCode == http.StatusNotImplemented {
		fmt.Println("[" + cyanText("i") + "] HEAD not supported by server. Falling back to single-thread download...")
		download(targetURL, finalPath)
		return
	}

	if statusCode != http.StatusOK {
		fmt.Printf("%s HTTP Error: %s\n", redText("[!]"), statusText)
		return
	}

	fileSize, err := strconv.ParseInt(contentLength, 10, 64)

	if chunks == 1 || err != nil || fileSize <= 0 || acceptRanges != "bytes" {
		if chunks > 1 && (err != nil || fileSize <= 0) {
			fmt.Println("[" + cyanText("i") + "] Content-Length unknown. Falling back to single-thread download...")
		} else if chunks > 1 && acceptRanges != "bytes" {
			fmt.Println("[" + cyanText("i") + "] Server does not support range requests. Falling back to single-thread download...")
		}
		download(targetURL, finalPath)
		return
	}

	if int64(chunks) > fileSize {
		chunks = int(fileSize)
	}
	if chunks <= 1 {
		download(targetURL, finalPath)
		return
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var cancelOnce sync.Once

	startTime := time.Now()

	for i := 0; i < chunks; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1

		if i == chunks-1 {
			end = fileSize - 1
		}

		wg.Add(1)
		go downloadByChunk(ctx, targetURL, start, end, file, bar, &wg, &completedChunks, chunks, errCh, &barMutex)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	failed := false
	for e := range errCh {
		if e != nil {
			fmt.Println(redText("[!]") + " " + e.Error())
			failed = true
			cancelOnce.Do(cancel)
		}
	}

	if failed {
		fmt.Println(redText("[!]") + " Download failed, file may be incomplete or corrupted")
		_ = os.Remove(finalPath)
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