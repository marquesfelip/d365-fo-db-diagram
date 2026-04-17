package main

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fatih/color"
	"github.com/marquesfelip/d365-fo-db-diagram/entity"
	"golang.org/x/sync/errgroup"
)

// outputZipPath defines the path of the output ZIP file generated at the end of processing.
const outputZipPath = "temp/output.zip"

var (
	errColor      = color.New(color.FgRed, color.Bold)
	progressColor = color.New(color.FgCyan)
	successColor  = color.New(color.FgGreen, color.Bold)
)

func main() {
	start := time.Now()

	// Root path where the extracted D365 packages are located (e.g., ApplicationSuite, SCMControls...)
	rootPath := filepath.Join("temp", "PackagesLocalDirectory")

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		errColor.Fprintf(os.Stderr, "error reading directory %s: %v\n", rootPath, err)
		return
	}

	// Create the ZIP file that will contain all generated JSONs
	zipFile, err := os.Create(outputZipPath)
	if err != nil {
		errColor.Fprintf(os.Stderr, "error creating ZIP file %s: %v\n", outputZipPath, err)
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	var zipMutex sync.Mutex

	// Pre-scan to know the total number of AxTable files to process (for progress display)
	total := countTotalAxTableFiles(rootPath, entries)

	var processed atomic.Int64

	// Start goroutine to periodically display progress
	stopProgress := startProgressReporter(&processed, total)

	// For each package found in the root directory (e.g., ApplicationSuite)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		packageDir := entry.Name()
		descriptorPath := filepath.Join(rootPath, packageDir, "Descriptor")

		// Only process if the Descriptor folder exists (valid package)
		if _, err := os.Stat(descriptorPath); os.IsNotExist(err) {
			continue
		}

		// Read descriptors and process AxTable folders, writing JSONs to the ZIP
		processDescriptorFolder(
			descriptorPath,
			filepath.Join(rootPath, packageDir),
			packageDir,
			&processed,
			zipWriter,
			&zipMutex,
		)
	}

	stopProgress()

	n := processed.Load()
	fmt.Fprint(os.Stderr, "\r")

	// Display final processing summary
	successColor.Fprintf(os.Stderr, "Completed: %d of %d files read in %s\n", n, total, time.Since(start))
	successColor.Fprintf(os.Stderr, "ZIP generated at: %s\n", outputZipPath)
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

// processDescriptorFolder processes all XML files in the Descriptor folder of a package.
// For each XML file found:
//   - extracts modelFolder (filename without extension)
//   - extracts modelName (DisplayName field inside AxModelInfo)
//   - locates the modelFolder inside the package and processes the AxTable subfolder
//   - generates a JSON file per table and writes it to the ZIP
//
// Parameters:
//   - descriptorPath: path to the Descriptor folder
//   - packagePath: root path of the package (e.g., .../ApplicationSuite)
//   - packageDir: name of the package directory (e.g., "ApplicationSuite")
//   - processed: pointer to atomic progress counter
//   - zipWriter: output ZIP file writer
//   - zipMutex: mutex to protect concurrent writes to the ZIP
func processDescriptorFolder(
	descriptorPath, packagePath, packageDir string,
	processed *atomic.Int64,
	zipWriter *zip.Writer,
	zipMutex *sync.Mutex,
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

		// modelFolder → descriptor filename without extension (e.g., "Foundation")
		modelFolder := strings.TrimSuffix(xmlFile.Name(), ".xml")

		descriptor, err := readDescriptorXML(xmlFilePath, modelFolder)
		if err != nil {
			errColor.Fprintf(os.Stderr, "\nerro ao ler descriptor %s: %v\n", xmlFile.Name(), err)
			continue
		}

		// descriptor.DisplayName → readable modelName (e.g., "Application Suite")
		_ = descriptor.DisplayName

		axTablePath := filepath.Join(packagePath, descriptor.ModelFolder, "AxTable")

		// Only process the AxTable folder if it exists
		if _, err := os.Stat(axTablePath); os.IsNotExist(err) {
			continue
		}

		processAxTableFolder(axTablePath, packageDir, descriptor.ModelFolder, processed, zipWriter, zipMutex)
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

// processAxTableFolder processes all AxTable.xml files in a folder in parallel (up to maxWorkers).
// For each file, parses the XML, serializes to JSON, and writes to the ZIP.
//
// Parameters:
//   - axTablePath: path to the AxTable folder
//   - packageDir: name of the parent package (e.g., "ApplicationSuite")
//   - modelFolder: name of the model folder (e.g., "Foundation")
//   - processed: pointer to atomic progress counter
//   - zipWriter: output ZIP file writer
//   - zipMutex: mutex to protect concurrent writes to the ZIP
func processAxTableFolder(
	axTablePath, packageDir, modelFolder string,
	processed *atomic.Int64,
	zipWriter *zip.Writer,
	zipMutex *sync.Mutex,
) {
	xmlFiles, err := os.ReadDir(axTablePath)
	if err != nil {
		errColor.Fprintf(os.Stderr, "\nerro ao ler pasta AxTable: %v\n", err)
		return
	}

	const maxWorkers = 8 // Limits concurrency to avoid overloading the system

	var group errgroup.Group

	group.SetLimit(maxWorkers)

	for _, xmlFile := range xmlFiles {
		if !strings.HasSuffix(xmlFile.Name(), ".xml") {
			continue
		}

		// Create a local copy to avoid closure issues in the goroutine (fixed in Go 1.22+)
		xmlFile := xmlFile

		group.Go(func() error {
			xmlFilePath := filepath.Join(axTablePath, xmlFile.Name())

			// Parse the table XML
			table, err := readAxTableXML(xmlFilePath)
			if err != nil {
				return fmt.Errorf("error reading %s: %w", xmlFile.Name(), err)
			}

			// Serialize to JSON and write to the ZIP file
			if err := writeTableJSON(zipWriter, zipMutex, packageDir, modelFolder, table); err != nil {
				return fmt.Errorf("error writing JSON for %s: %w", table.Name, err)
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
//
// Returns:
//   - AxTable struct with all fields read
//   - error, if any
func readAxTableXML(filePath string) (entity.AxTable, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return entity.AxTable{}, err
	}

	var axTable entity.AxTable
	if err := xml.Unmarshal(data, &axTable); err != nil {
		return entity.AxTable{}, err
	}

	return axTable, nil
}

// writeTableJSON serializes an AxTable to indented JSON and writes it as an entry in the ZIP file.
// The entry path inside the ZIP follows the pattern: packageDir/modelFolder/TableName.json
//
// Parameters:
//   - zipWriter: output ZIP file writer
//   - mutex: mutex to ensure exclusive access to zipWriter
//   - packageDir: name of the parent package (e.g., "ApplicationSuite")
//   - modelFolder: name of the model folder (e.g., "Foundation")
//   - table: AxTable struct to be serialized
//
// Returns:
//   - error, if any
func writeTableJSON(
	zipWriter *zip.Writer,
	mutex *sync.Mutex,
	packageDir, modelFolder string,
	table entity.AxTable,
) error {

	jsonData, err := json.MarshalIndent(table, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializing JSON: %w", err)
	}

	// Path inside the ZIP: package/modelFolder/TableName.json
	entryName := fmt.Sprintf("%s/%s/%s.json", packageDir, modelFolder, table.Name)

	mutex.Lock()
	defer mutex.Unlock()

	w, err := zipWriter.Create(entryName)
	if err != nil {
		return fmt.Errorf("erro ao criar entrada ZIP %s: %w", entryName, err)
	}

	_, err = w.Write(jsonData)
	return err
}
