package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/schollz/progressbar/v3"
)

const maxChunkRetries = 3

func download(url string, filePath string) {
	startTime := time.Now()

	var fileSize int64
	var err error

	for attempt := 1; attempt <= maxChunkRetries; attempt++ {
		fileSize, err = attemptDownload(url, filePath)
		if err == nil {
			break
		}
		fmt.Println(redText("[!]") + " " + err.Error())
		if attempt < maxChunkRetries {
			fmt.Println("[" + cyanText("i") + "] Retrying...")
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
	}

	if err != nil {
		_ = os.Remove(filePath)
		fmt.Println(redText("[!]") + " Download failed after multiple attempts")
		return
	}

	duration := time.Since(startTime)
	seconds := duration.Seconds()
	if seconds <= 0 {
		seconds = 0.001
	}
	speed := float64(fileSize) / seconds / (1000 * 1000)

	printSummary(filePath, fileSize, duration, speed)
}

func attemptDownload(url string, filePath string) (int64, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", userAgent)

	respond, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer respond.Body.Close()

	if respond.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("HTTP Error: %s", respond.Status)
	}

	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	bar := progressbar.DefaultBytes(
		respond.ContentLength,
		cyanText("[1/1]")+" Downloading",
	)

	writer := io.MultiWriter(file, bar)
	fileSize, err := io.Copy(writer, respond.Body)
	if err != nil {
		return fileSize, err
	}

	if respond.ContentLength > 0 && fileSize != respond.ContentLength {
		return fileSize, fmt.Errorf("download incomplete, file may be corrupted")
	}

	_ = bar.Finish()
	fmt.Fprintln(os.Stderr)

	return fileSize, nil
}

func downloadByChunk(ctx context.Context, url string, start, end int64, file *os.File, bar *progressbar.ProgressBar, wg *sync.WaitGroup, completedChunks *int32, totalChunks int, errCh chan<- error, barMutex *sync.Mutex) {
	defer wg.Done()

	expectedBytes := end - start + 1
	var lastErr error

	for attempt := 1; attempt <= maxChunkRetries; attempt++ {
		if ctx.Err() != nil {
			errCh <- ctx.Err()
			return
		}

		written, err := attemptChunkDownload(ctx, url, start, end, file, bar, barMutex)

		if err == nil && written == expectedBytes {
			done := atomic.AddInt32(completedChunks, 1)
			barMutex.Lock()
			bar.Describe(fmt.Sprintf("%s Downloading", cyanText(fmt.Sprintf("[%d/%d]", done, totalChunks))))
			barMutex.Unlock()
			errCh <- nil
			return
		}

		if written > 0 {
			barMutex.Lock()
			_ = bar.Add(-int(written))
			barMutex.Unlock()
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("chunk %d-%d incomplete: got %d of %d bytes", start, end, written, expectedBytes)
		}

		if ctx.Err() != nil {
			errCh <- ctx.Err()
			return
		}

		if attempt < maxChunkRetries {
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
		}
	}

	errCh <- fmt.Errorf("chunk %d-%d failed after %d attempts: %w", start, end, maxChunkRetries, lastErr)
}

func attemptChunkDownload(ctx context.Context, url string, start, end int64, file *os.File, bar *progressbar.ProgressBar, barMutex *sync.Mutex) (int64, error) {
	request, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("User-Agent", userAgent)
	request.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	respond, err := httpClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer respond.Body.Close()

	if respond.StatusCode != http.StatusPartialContent {
		_, _ = io.Copy(io.Discard, io.LimitReader(respond.Body, 1<<20))
		return 0, fmt.Errorf("chunk %d-%d failed: %s", start, end, respond.Status)
	}

	buffer := make([]byte, 512*1024)
	currentOffset := start

	for {
		n, readErr := respond.Body.Read(buffer)
		if n > 0 {
			if _, writeErr := file.WriteAt(buffer[:n], currentOffset); writeErr != nil {
				return currentOffset - start, writeErr
			}
			currentOffset += int64(n)

			barMutex.Lock()
			_ = bar.Add(n)
			barMutex.Unlock()
		}
		if readErr == io.EOF {
			return currentOffset - start, nil
		}
		if readErr != nil {
			return currentOffset - start, readErr
		}
	}
}