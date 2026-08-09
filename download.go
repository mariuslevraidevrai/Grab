package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/schollz/progressbar/v3"
)

func download(url string, filePath string) {
	startTime := time.Now()

	respond, err := httpClient.Get(url)
	if err != nil {
		fmt.Println(redText("[!]") + " Request error:", err)
		return
	}
	defer respond.Body.Close()

	if respond.StatusCode != http.StatusOK {
		fmt.Printf("%s HTTP Error: %s\n", redText("[!]"), respond.Status)
		return
	}

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Println(redText("[!]") + " File creation error:", err)
		return
	}
	defer file.Close()

	bar := progressbar.DefaultBytes(
		respond.ContentLength,
		cyanText("[1/1]")+" Downloading",
	)

	writer := io.MultiWriter(file, bar)
	fileSize, err := io.Copy(writer, respond.Body)
	if err != nil {
		fmt.Println(redText("[!]") + " Write error:", err)
		return
	}

	if respond.ContentLength > 0 && fileSize != respond.ContentLength {
		fmt.Println(redText("[!]") + " Download incomplete, file may be corrupted")
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

	printSummary(filePath, fileSize, duration, speed)
}

func downloadByChunk(url string, start, end int64, file *os.File, bar *progressbar.ProgressBar, wg *sync.WaitGroup, completedChunks *int32, totalChunks int, errCh chan<- error, barMutex *sync.Mutex) {
	defer wg.Done()

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		errCh <- err
		return
	}
	request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	respond, err := httpClient.Do(request)
	if err != nil {
		errCh <- err
		return
	}
	defer respond.Body.Close()

	if respond.StatusCode != http.StatusPartialContent {
		errCh <- fmt.Errorf("chunk %d-%d failed: %s", start, end, respond.Status)
		return
	}

	expectedBytes := end - start + 1
	buffer := make([]byte, 256*1024)
	currentOffset := start

	for {
		n, readErr := respond.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := file.WriteAt(buffer[:n], currentOffset); writeErr != nil {
				errCh <- writeErr
				return
			}
			currentOffset += int64(n)
			_ = bar.Add(n)
		}
		if readErr == io.EOF {
			if currentOffset-start != expectedBytes {
				errCh <- fmt.Errorf("chunk %d-%d incomplete: got %d of %d bytes", start, end, currentOffset-start, expectedBytes)
				return
			}
			done := atomic.AddInt32(completedChunks, 1)
			barMutex.Lock()
			bar.Describe(fmt.Sprintf("%s Downloading", cyanText(fmt.Sprintf("[%d/%d]", done, totalChunks))))
			barMutex.Unlock()
			errCh <- nil
			return
		}
		if readErr != nil {
			errCh <- readErr
			return
		}
	}
}