package main

import (
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

var (
	errColor      = color.New(color.FgRed, color.Bold)
	progressColor = color.New(color.FgCyan)
	successColor  = color.New(color.FgGreen, color.Bold)
)

func main() {
	start := time.Now()

	// Caminho raiz onde estão os pacotes extraídos do D365
	rootPath := filepath.Join("temp", "PackageLocalDirectory")

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		errColor.Fprintf(os.Stderr, "erro ao ler diretório %s: %v\n", rootPath, err)
		return
	}

	// Faz uma varredura prévia para saber o total de arquivos AxTable a processar (para exibir progresso)
	total := countTotalAxTableFiles(rootPath, entries)

	var processed atomic.Int64
	// Inicia goroutine para exibir progresso periodicamente
	stopProgress := startProgressReporter(&processed, total)

	// allResults armazena todos os campos de todas as tabelas processadas
	var (
		allMutex   sync.Mutex
		allResults []entity.TableFieldInfo
	)

	// Para cada pacote encontrado no diretório raiz
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

		// Processa a pasta Descriptor e coleta os campos das tabelas
		results := processDescriptorFolder(descriptorPath, filepath.Join(rootPath, packageDir), &processed)
		allMutex.Lock()
		allResults = append(allResults, results...)
		allMutex.Unlock()
	}

	stopProgress()

	n := processed.Load()
	fmt.Fprint(os.Stderr, "\r")
	// Exibe resumo final do processamento
	successColor.Fprintf(os.Stderr, "Concluído: %d de %d arquivos lidos em %s\n", n, total, time.Since(start))

	// Exemplo de uso: imprime todos os campos encontrados
	for _, info := range allResults {
		fmt.Printf("tabela: %s, campo: %s, ExtendedDataType: %s\n",
			info.TableName,
			info.FieldName,
			info.ExtendedDataType,
		)
	}
}

// Faz uma varredura prévia para saber quantos arquivos XML
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

// Inicia uma goroutine que exibe o progresso do processamento a cada segundo.
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

// Processa todos os arquivos XML da pasta Descriptor de um pacote.
// Para cada arquivo, tenta encontrar e processar a pasta AxTable correspondente.
//
// Parâmetros:
//   - descriptorPath: caminho da pasta Descriptor
//   - packagePath: caminho do pacote
//   - processed: ponteiro para contador atômico de progresso
//
// Retorna:
//   - slice com todos os campos das tabelas encontradas
func processDescriptorFolder(descriptorPath, packagePath string, processed *atomic.Int64) []entity.TableFieldInfo {
	xmlFiles, err := os.ReadDir(descriptorPath)
	if err != nil {
		errColor.Fprintf(os.Stderr, "\nerro ao ler pasta Descriptor: %v\n", err)
		return nil
	}

	var results []entity.TableFieldInfo

	for _, xmlFile := range xmlFiles {
		if !strings.HasSuffix(xmlFile.Name(), ".xml") {
			continue
		}

		xmlFilePath := filepath.Join(descriptorPath, xmlFile.Name())

		// Lê o arquivo descriptor apenas para garantir que é válido
		_, err := readDescriptorXML(xmlFilePath)
		if err != nil {
			errColor.Fprintf(os.Stderr, "\nerro ao ler descriptor %s: %v\n", xmlFile.Name(), err)
			continue
		}

		folderName := strings.TrimSuffix(xmlFile.Name(), ".xml")
		tableFolderPath := filepath.Join(packagePath, folderName, "AxTable")

		// Se existir a pasta AxTable correspondente, processa todos os arquivos nela
		if _, err := os.Stat(tableFolderPath); !os.IsNotExist(err) {
			tableResults := processAxTableFolder(tableFolderPath, processed)
			results = append(results, tableResults...)
		}
	}

	return results
}

// readDescriptorXML faz o Unmarshal do XML do descriptor apenas para validar e extrair o DisplayName.
//
// Parâmetros:
//   - filePath: caminho do arquivo descriptor XML
//
// Retorna:
//   - DisplayName extraído do XML
//   - erro, se houver
func readDescriptorXML(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	var descriptor entity.Descriptor
	err = xml.Unmarshal(data, &descriptor)
	if err != nil {
		return "", err
	}

	return descriptor.DisplayName, nil
}

// processAxTableFolder processa todos os arquivos AxTable.xml de uma pasta em paralelo (até maxWorkers).
//
// Parâmetros:
//   - axTablePath: caminho da pasta AxTable
//   - processed: ponteiro para contador atômico de progresso
//
// Retorna:
//   - slice com todos os campos das tabelas encontrados
func processAxTableFolder(axTablePath string, processed *atomic.Int64) []entity.TableFieldInfo {
	xmlFiles, err := os.ReadDir(axTablePath)
	if err != nil {
		errColor.Fprintf(os.Stderr, "\nerro ao ler pasta AxTable: %v\n", err)
		return nil
	}

	const maxWorkers = 8 // Limita concorrência para não sobrecarregar o sistema

	var (
		group   errgroup.Group
		mutex   sync.Mutex
		results []entity.TableFieldInfo
	)

	group.SetLimit(maxWorkers)

	for _, xmlFile := range xmlFiles {
		if !strings.HasSuffix(xmlFile.Name(), ".xml") {
			continue
		}

		xmlFile := xmlFile
		group.Go(func() error {
			xmlFilePath := filepath.Join(axTablePath, xmlFile.Name())

			// Lê e extrai todos os campos da tabela deste arquivo
			tableInfos, err := readAxTableXML(xmlFilePath)
			if err != nil {
				return fmt.Errorf("erro ao ler %s: %w", xmlFile.Name(), err)
			}

			processed.Add(1)

			mutex.Lock()
			results = append(results, tableInfos...)
			mutex.Unlock()
			return nil
		})
	}

	if err := group.Wait(); err != nil {
		errColor.Fprintf(os.Stderr, "\n%v\n", err)
	}

	return results
}

// readAxTableXML faz o Unmarshal do AxTable.xml e retorna todos os campos da tabela.
//
// Parâmetros:
//   - filePath: caminho do arquivo AxTable.xml
//
// Retorna:
//   - slice com TableFieldInfo para cada campo da tabela
//   - erro, se houver
func readAxTableXML(filePath string) ([]entity.TableFieldInfo, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var axTable entity.AxTable
	err = xml.Unmarshal(data, &axTable)
	if err != nil {
		return nil, err
	}

	var results []entity.TableFieldInfo

	// Para cada campo da tabela, cria um TableFieldInfo
	for _, field := range axTable.Fields.AxTableField {
		info := entity.TableFieldInfo{
			TableName:        axTable.Name,
			FieldName:        field.Name,
			ExtendedDataType: field.ExtendedDataType,
		}
		results = append(results, info)
	}

	return results, nil
}
