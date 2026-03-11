package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/marquesfelip/d365-fo-db-diagram/entity"
)

func main() {
	rootPath := filepath.Join("temp", "PackageLocalDirectory")

	entries, err := os.ReadDir(rootPath)
	if err != nil {
		fmt.Printf("erro ao ler diretório %s: %v\n", rootPath, err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		packageDir := entry.Name()
		fmt.Printf("\nprocessando pacote %s\n", packageDir)

		descriptorPath := filepath.Join(rootPath, packageDir, "Descriptor")

		if _, err := os.Stat(descriptorPath); os.IsNotExist(err) {
			fmt.Printf("pasta descriptor não encontrada em %s\n", packageDir)
			continue
		}

		processDescriptorFolder(descriptorPath, filepath.Join(rootPath, packageDir))
	}
}

func processDescriptorFolder(descriptorPath, packagePath string) {
	xmlFiles, err := os.ReadDir(descriptorPath)
	if err != nil {
		fmt.Printf("erro ao ler pasta Descriptor: %v\n", err)
		return
	}

	for _, xmlFile := range xmlFiles {
		if !strings.HasSuffix(xmlFile.Name(), ".xml") {
			continue
		}

		xmlFilePath := filepath.Join(descriptorPath, xmlFile.Name())

		displayName, err := readDescriptorXML(xmlFilePath)
		if err != nil {
			fmt.Printf("erro ao ler %s: %v\n", xmlFile.Name(), err)
			continue
		}

		fmt.Printf("encontrato: %s -> DisplayName: %s\n", xmlFile.Name(), displayName)

		folderName := strings.TrimSuffix(xmlFile.Name(), ".xml")

		tableFolderPath := filepath.Join(packagePath, folderName, "AxTable")

		if _, err := os.Stat(tableFolderPath); !os.IsNotExist(err) {
			fmt.Printf("processando AxTable em: %s\n", tableFolderPath)
			processAxTableFolder(tableFolderPath)
		} else {
			fmt.Printf("pasta %s não encontrado\n", folderName)
		}
	}
}

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

func processAxTableFolder(axTablePath string) {
	xmlFiles, err := os.ReadDir(axTablePath)
	if err != nil {
		fmt.Printf("erro ao ler pasta AxTable: %v\n", err)
		return
	}

	for _, xmlFile := range xmlFiles {
		if !strings.HasSuffix(xmlFile.Name(), ".xml") {
			continue
		}

		xmlFilePath := filepath.Join(axTablePath, xmlFile.Name())

		tableInfos, err := readAxTableXML(xmlFilePath)
		if err != nil {
			fmt.Printf("erro ao ler %s: %v\n", xmlFile.Name(), err)
			continue
		}

		fmt.Printf("arquivo %s\n", xmlFile.Name())
		for _, info := range tableInfos {
			fmt.Printf("tabela: %s, campo: %s, ExtendedDataType: %s\n", info.TableName, info.FieldName, info.ExtendedDataType)
		}
	}
}

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
