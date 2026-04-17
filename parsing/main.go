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

// outputZipPath define o caminho do arquivo ZIP de saída gerado ao final do processamento.
const outputZipPath = "temp/output.zip"

var (
	errColor      = color.New(color.FgRed, color.Bold)
	progressColor = color.New(color.FgCyan)
	successColor  = color.New(color.FgGreen, color.Bold)
)

func main() {
	start := time.Now()

	// Caminho raiz onde estão os pacotes extraídos do D365 (ex: ApplicationSuite, SCMControls...)
	rootPath := filepath.Join("temp", "PackagesLocalDirectory")

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		errColor.Fprintf(os.Stderr, "erro ao ler diretório %s: %v\n", rootPath, err)
		return
	}

	// Cria o arquivo ZIP que irá conter todos os JSONs gerados
	zipFile, err := os.Create(outputZipPath)
	if err != nil {
		errColor.Fprintf(os.Stderr, "erro ao criar arquivo ZIP %s: %v\n", outputZipPath, err)
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	var zipMutex sync.Mutex

	// Faz uma varredura prévia para saber o total de arquivos AxTable a processar (para exibir progresso)
	total := countTotalAxTableFiles(rootPath, entries)

	var processed atomic.Int64

	// Inicia goroutine para exibir progresso periodicamente
	stopProgress := startProgressReporter(&processed, total)

	// Para cada pacote encontrado no diretório raiz (ex: ApplicationSuite)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		packageDir := entry.Name()
		descriptorPath := filepath.Join(rootPath, packageDir, "Descriptor")

		// Só processa se existir a pasta Descriptor (pacote válido)
		if _, err := os.Stat(descriptorPath); os.IsNotExist(err) {
			continue
		}

		// Lê os descritores e processa as pastas AxTable, gravando JSONs no ZIP
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

	// Exibe resumo final do processamento
	successColor.Fprintf(os.Stderr, "Concluído: %d de %d arquivos lidos em %s\n", n, total, time.Since(start))
	successColor.Fprintf(os.Stderr, "ZIP gerado em: %s\n", outputZipPath)
}

// countTotalAxTableFiles faz uma varredura prévia para saber quantos arquivos XML
// existem nas pastas AxTable (usado para exibir o progresso total).
//
// Parâmetros:
//   - rootPath: caminho raiz dos pacotes extraídos
//   - entries: lista de diretórios/pacotes
//
// Retorna:
//   - total de arquivos AxTable.xml encontrados
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

// startProgressReporter inicia uma goroutine que exibe o progresso do processamento a cada segundo.
//
// Parâmetros:
//   - processed: ponteiro para contador atômico de arquivos processados
//   - total: total de arquivos a processar
//
// Retorna:
//   - função que, ao ser chamada, encerra o progresso
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
				progressColor.Fprintf(os.Stderr, "\rArquivos lidos: %d de %d", n, total)
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

// processDescriptorFolder processa todos os arquivos XML da pasta Descriptor de um pacote.
// Para cada arquivo XML encontrado:
//   - extrai o modelFolder (nome do arquivo sem extensão)
//   - extrai o modelName (campo DisplayName dentro de AxModelInfo)
//   - localiza a pasta modelFolder dentro do pacote e processa a subpasta AxTable
//   - gera um arquivo JSON por tabela e o grava no ZIP
//
// Parâmetros:
//   - descriptorPath: caminho da pasta Descriptor
//   - packagePath: caminho da raiz do pacote (ex: .../ApplicationSuite)
//   - packageDir: nome do diretório do pacote (ex: "ApplicationSuite")
//   - processed: ponteiro para contador atômico de progresso
//   - zipWriter: escritor do arquivo ZIP de saída
//   - zipMutex: mutex para proteger escritas concorrentes no ZIP
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

		// modelFolder → nome do arquivo descriptor sem extensão (ex: "Foundation")
		modelFolder := strings.TrimSuffix(xmlFile.Name(), ".xml")

		descriptor, err := readDescriptorXML(xmlFilePath, modelFolder)
		if err != nil {
			errColor.Fprintf(os.Stderr, "\nerro ao ler descriptor %s: %v\n", xmlFile.Name(), err)
			continue
		}

		// descriptor.DisplayName → modelName legível (ex: "Application Suite")
		_ = descriptor.DisplayName

		axTablePath := filepath.Join(packagePath, descriptor.ModelFolder, "AxTable")

		// Processa a pasta AxTable somente se ela existir
		if _, err := os.Stat(axTablePath); os.IsNotExist(err) {
			continue
		}

		processAxTableFolder(axTablePath, packageDir, descriptor.ModelFolder, processed, zipWriter, zipMutex)
	}
}

// readDescriptorXML faz o Unmarshal do XML do descriptor e preenche o campo ModelFolder.
//
// Parâmetros:
//   - filePath: caminho do arquivo descriptor XML
//   - modelFolder: nome derivado do arquivo (sem extensão), atribuído a Descriptor.ModelFolder
//
// Retorna:
//   - Descriptor com DisplayName e ModelFolder preenchidos
//   - erro, se houver
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

// processAxTableFolder processa todos os arquivos AxTable.xml de uma pasta em paralelo (até maxWorkers).
// Para cada arquivo, faz o parse do XML, serializa para JSON e grava no ZIP.
//
// Parâmetros:
//   - axTablePath: caminho da pasta AxTable
//   - packageDir: nome do pacote pai (ex: "ApplicationSuite")
//   - modelFolder: nome da pasta do modelo (ex: "Foundation")
//   - processed: ponteiro para contador atômico de progresso
//   - zipWriter: escritor do arquivo ZIP de saída
//   - zipMutex: mutex para proteger escritas concorrentes no ZIP
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

	const maxWorkers = 8 // Limita concorrência para não sobrecarregar o sistema

	var group errgroup.Group

	group.SetLimit(maxWorkers)

	for _, xmlFile := range xmlFiles {
		if !strings.HasSuffix(xmlFile.Name(), ".xml") {
			continue
		}

		// Cria uma cópia local para evitar problemas de closure na goroutine (corrigido com go 1.22+)
		xmlFile := xmlFile

		group.Go(func() error {
			xmlFilePath := filepath.Join(axTablePath, xmlFile.Name())

			// Faz o parse do XML da tabela
			table, err := readAxTableXML(xmlFilePath)
			if err != nil {
				return fmt.Errorf("erro ao ler %s: %w", xmlFile.Name(), err)
			}

			// Serializa para JSON e grava no arquivo ZIP
			if err := writeTableJSON(zipWriter, zipMutex, packageDir, modelFolder, table); err != nil {
				return fmt.Errorf("erro ao gravar JSON de %s: %w", table.Name, err)
			}

			processed.Add(1)
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		errColor.Fprintf(os.Stderr, "\n%v\n", err)
	}
}

// readAxTableXML faz o Unmarshal de um arquivo AxTable.xml e retorna a estrutura AxTable.
//
// Parâmetros:
//   - filePath: caminho do arquivo AxTable.xml
//
// Retorna:
//   - struct AxTable com todos os campos lidos
//   - erro, se houver
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

// writeTableJSON serializa um AxTable para JSON indentado e o grava como entrada no arquivo ZIP.
// O caminho da entrada dentro do ZIP segue o padrão: packageDir/modelFolder/NomeDaTabela.json
//
// Parâmetros:
//   - zipWriter: escritor do arquivo ZIP de saída
//   - mutex: mutex para garantir acesso exclusivo ao zipWriter
//   - packageDir: nome do pacote pai (ex: "ApplicationSuite")
//   - modelFolder: nome da pasta do modelo (ex: "Foundation")
//   - table: struct AxTable a ser serializada
//
// Retorna:
//   - erro, se houver
func writeTableJSON(
	zipWriter *zip.Writer,
	mutex *sync.Mutex,
	packageDir, modelFolder string,
	table entity.AxTable,
) error {

	jsonData, err := json.MarshalIndent(table, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar JSON: %w", err)
	}

	// Caminho dentro do ZIP: pacote/modelFolder/NomeDaTabela.json
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
