package main

import (
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/marquesfelip/d365-fo-db-diagram/entity"
	"golang.org/x/sync/errgroup"
)

var (
	errColor      = color.New(color.FgRed, color.Bold)
	progressColor = color.New(color.FgCyan)
	successColor  = color.New(color.FgGreen, color.Bold)
)

func main() {
	start := time.Now()

	rootPath := filepath.Join("temp", "PackagesLocalDirectory")
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		errColor.Fprintf(os.Stderr, "error reading directory %s: %v\n", rootPath, err)
		return
	}

	total := countTotalAxTableFiles(rootPath, entries)
	var processed atomic.Int64

	stopProgress := startProgressReporter(&processed, total)

	// Prepare HTTP request with GZIP streaming
	pr, pw := io.Pipe()
	gzipWriter := gzip.NewWriter(pw)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		client := &http.Client{}
		req, err := http.NewRequest("POST", "http://localhost:8080/api/ingest", pr)
		if err != nil {
			errColor.Fprintf(os.Stderr, "error creating HTTP request: %v\n", err)
			pr.CloseWithError(err)
			return
		}
		req.Header.Set("Content-Type", "application/x-ndjson")
		req.Header.Set("Content-Encoding", "gzip")

		resp, err := client.Do(req)
		if err != nil {
			errColor.Fprintf(os.Stderr, "error sending HTTP request: %v\n", err)
			pr.CloseWithError(err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			errColor.Fprintf(os.Stderr, "server returned status %d: %s\n", resp.StatusCode, string(body))
		}
	}()

	var gzipMutex sync.Mutex

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		packageDir := entry.Name()
		descriptorPath := filepath.Join(rootPath, packageDir, "Descriptor")

		if _, err := os.Stat(descriptorPath); os.IsNotExist(err) {
			continue
		}

		processDescriptorFolderNDJSON(
			descriptorPath,
			filepath.Join(rootPath, packageDir),
			packageDir,
			&processed,
			gzipWriter,
			&gzipMutex,
		)
	}

	// Finalize writers
	gzipWriter.Close()
	pw.Close()
	wg.Wait()

	stopProgress()
	n := processed.Load()
	fmt.Fprint(os.Stderr, "\r")

	successColor.Fprintf(os.Stderr, "Completed: %d of %d files read in %s\n", n, total, time.Since(start))
}

// countTotalAxTableFiles performs a pre-scan to determine how many XML files
// exist in the AxTable folders (used to display total progress).
//
// Parameters:
//   - rootPath: root path of the extracted packages
//   - entries: list of directories/packages
//
// Returns:
//   - total number of AxTable.xml files found
func countTotalAxTableFiles(rootPath string, entries []os.DirEntry) int64 {
	var total int64

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		packageDir := entry.Name()
		descriptorPath := filepath.Join(rootPath, packageDir, "Descriptor")
		xmlFiles, err := os.ReadDir(descriptorPath)
		if err != nil {
			continue
		}
		for _, xmlFile := range xmlFiles {
			if !strings.HasSuffix(xmlFile.Name(), ".xml") {
				continue
			}
			folderName := strings.TrimSuffix(xmlFile.Name(), ".xml")
			tableFolderPath := filepath.Join(rootPath, packageDir, folderName, "AxTable")
			tableFiles, err := os.ReadDir(tableFolderPath)
			if err != nil {
				continue
			}
			for _, f := range tableFiles {
				if strings.HasSuffix(f.Name(), ".xml") {
					total++
				}
			}
		}
	}
	return total
}

// startProgressReporter starts a goroutine that displays processing progress every second.
//
// Parameters:
//   - processed: pointer to atomic counter of processed files
//   - total: total number of files to process
//
// Returns:
//   - function that, when called, stops the progress display
func startProgressReporter(processed *atomic.Int64, total int64) func() {
	stop := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n := processed.Load()
				progressColor.Fprintf(os.Stderr, "\rFiles read: %d of %d", n, total)
			case <-stop:
				return
			}
		}
	}()

	return func() {
		close(stop)
		<-done
	}
}

// processDescriptorFolderNDJSON processes all XML files in the Descriptor folder of a package and streams NDJSON to a gzip.Writer.
func processDescriptorFolderNDJSON(
	descriptorPath, packagePath, packageDir string,
	processed *atomic.Int64,
	gzipWriter *gzip.Writer,
	gzipMutex *sync.Mutex,
) {
	xmlFiles, err := os.ReadDir(descriptorPath)
	if err != nil {
		errColor.Fprintf(os.Stderr, "\nerro ao ler pasta Descriptor: %v\n", err)
		return
	}

	for _, xmlFile := range xmlFiles {
		if !strings.HasSuffix(xmlFile.Name(), ".xml") {
			continue
		}

		xmlFilePath := filepath.Join(descriptorPath, xmlFile.Name())
		modelFolder := strings.TrimSuffix(xmlFile.Name(), ".xml")

		descriptor, err := readDescriptorXML(xmlFilePath, modelFolder)
		if err != nil {
			errColor.Fprintf(os.Stderr, "\nerro ao ler descriptor %s: %v\n", xmlFile.Name(), err)
			continue
		}

		axTablePath := filepath.Join(packagePath, descriptor.ModelFolder, "AxTable")
		if _, err := os.Stat(axTablePath); os.IsNotExist(err) {
			continue
		}

		processAxTableFolderNDJSON(axTablePath, packageDir, descriptor.ModelFolder, descriptor.DisplayName, processed, gzipWriter, gzipMutex)
	}
}

// readDescriptorXML unmarshals the descriptor XML and fills the ModelFolder field.
//
// Parameters:
//   - filePath: path to the descriptor XML file
//   - modelFolder: name derived from the file (without extension), assigned to Descriptor.ModelFolder
//
// Returns:
//   - Descriptor with DisplayName and ModelFolder filled
//   - error, if any
func readDescriptorXML(filePath, modelFolder string) (entity.Descriptor, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return entity.Descriptor{}, err
	}

	var descriptor entity.Descriptor
	if err := xml.Unmarshal(data, &descriptor); err != nil {
		return entity.Descriptor{}, err
	}

	descriptor.ModelFolder = modelFolder

	return descriptor, nil
}

// processAxTableFolderNDJSON processes all AxTable.xml files in a folder in parallel and streams NDJSON to a gzip.Writer.
func processAxTableFolderNDJSON(
	axTablePath, packageDir, modelFolder, modelDisplayName string,
	processed *atomic.Int64,
	gzipWriter *gzip.Writer,
	gzipMutex *sync.Mutex,
) {
	xmlFiles, err := os.ReadDir(axTablePath)
	if err != nil {
		errColor.Fprintf(os.Stderr, "\nerro ao ler pasta AxTable: %v\n", err)
		return
	}

	maxWorkers := runtime.NumCPU() * 2
	var group errgroup.Group
	group.SetLimit(maxWorkers)

	for _, xmlFile := range xmlFiles {
		if !strings.HasSuffix(xmlFile.Name(), ".xml") {
			continue
		}
		xmlFile := xmlFile
		group.Go(func() error {
			xmlFilePath := filepath.Join(axTablePath, xmlFile.Name())
			table, err := readAxTableXML(xmlFilePath, modelDisplayName)
			if err != nil {
				return fmt.Errorf("error reading %s: %w", xmlFile.Name(), err)
			}

			// Serialize to JSON and write as NDJSON line
			line, err := json.Marshal(table)
			if err != nil {
				return fmt.Errorf("error serializing JSON for %s: %w", table.Name, err)
			}

			gzipMutex.Lock()
			_, err = gzipWriter.Write(append(line, '\n'))
			gzipMutex.Unlock()
			if err != nil {
				return fmt.Errorf("error writing NDJSON for %s: %w", table.Name, err)
			}

			processed.Add(1)
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		errColor.Fprintf(os.Stderr, "\n%v\n", err)
	}
}

// readAxTableXML unmarshals an AxTable.xml file and returns the AxTable structure.
//
// Parameters:
//   - filePath: path to the AxTable.xml file
//   - modelName: the DisplayName of the model to assign to AxTable.Model
//
// Returns:
//   - AxTable struct with all fields read and Model set
//   - error, if any
func readAxTableXML(filePath string, modelName string) (entity.AxTable, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return entity.AxTable{}, err
	}

	var axTable entity.AxTable
	if err := xml.Unmarshal(data, &axTable); err != nil {
		return entity.AxTable{}, err
	}
	axTable.Model = modelName
	return axTable, nil
}

// writeTableJSON is no longer needed in NDJSON streaming mode.
