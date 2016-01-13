package main

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	//"io/ioutil"
	"log"
	"os"
)

var oneTabs string = "\t"
var twoTabs string = "\t\t"
var threeTabs string = "\t\t\t"
var fourTabs string = "\t\t\t\t"

func writeStaticPart(inputFile string, dstFile *os.File) {
	hdr, err := os.Open(inputFile)
	if err != nil {
		log.Fatal(err)
	}
	defer hdr.Close()

	scanner := bufio.NewScanner(hdr)
	for scanner.Scan() {
		line := scanner.Text()
		dstFile.WriteString(line + "\n")
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}

func writeResourceHdr(strName string, dstFile *os.File) {
	dstFile.WriteString(twoTabs + "\"/" + strName + "\": { \n")
	dstFile.WriteString(twoTabs + "\"post\": { " + "\n")
	dstFile.WriteString(threeTabs + "\"tags\": [ " + "\n")
	dstFile.WriteString(fourTabs + "\"" + strName + "\"" + "\n")
	dstFile.WriteString(threeTabs + "]," + "\n")
	dstFile.WriteString(twoTabs + "\"summary\": \"Create New" + strName + "\"," + "\n")
	dstFile.WriteString(twoTabs + "\"description\":" + "\"" + strName + "\"," + "\n")
	//dstFile.WriteString(twoTabs + "\"operationId\": \"add" + strName + ",\n")
	dstFile.WriteString(twoTabs + "\"consumes\": [" + "\n")
	dstFile.WriteString(threeTabs + "\"application/json\"," + "\n")
	dstFile.WriteString(twoTabs + "]," + "\n")
	dstFile.WriteString(twoTabs + "\"produces\": [ " + "\n")
	dstFile.WriteString(threeTabs + "\"application/json\"," + "\n")
	dstFile.WriteString(twoTabs + "]," + "\n")
	dstFile.WriteString(twoTabs + "\"parameters\": [ " + "\n")
}

func writeEpilogueForStruct(strName string, dstFile *os.File) {
	dstFile.WriteString(twoTabs + "\"responses\": { " + "\n")
	dstFile.WriteString(threeTabs + "\"405\": {" + "\n")
	dstFile.WriteString(fourTabs + "\"description\": \"Invalid input\"" + "\n")
	dstFile.WriteString(threeTabs + " }" + "\n")
	dstFile.WriteString(twoTabs + " }" + "\n")
	dstFile.WriteString(twoTabs + " } " + "\n")
}

func writeAttributeJson(attrName string, attrType string, dstFile *os.File) {
	var attrTypeVal string
	dstFile.WriteString(fourTabs + "{" + "\n")
	dstFile.WriteString(fourTabs + "\"in\": \"formData\"," + "\n")
	dstFile.WriteString(fourTabs + "\"name\":" + "\"" + attrName + "\"" + "," + "\n")
	switch attrType {
	case "string":
		attrTypeVal = "string"
	case "int32", "uint32":
		attrTypeVal = "integer"
	default:
		attrTypeVal = "string"
	}
	dstFile.WriteString(fourTabs + "\"type\":" + "\"" + attrTypeVal + "\"" + "," + "\n")
	dstFile.WriteString(fourTabs + "\"description\":" + "\"" + attrName + "\"" + "," + "\n")
	dstFile.WriteString(fourTabs + "\"required\":" + "true," + "\n")
	dstFile.WriteString(fourTabs + "}," + "\n")
}

func writePathCompletion(dstFile *os.File) {
	dstFile.WriteString(twoTabs + " }, " + "\n")
}
func main() {
	fset := token.NewFileSet() // positions are relative to fset
	outFileName := "flexApis.json"

	inputFile := "../../models/objects.go"

	docJsFile, err := os.Create(outFileName)
	if err != nil {
		fmt.Println("Failed to open the file")
		return
	}
	defer docJsFile.Close()

	// Write Header by copying each line from header file
	writeStaticPart("part1.txt", docJsFile)
	docJsFile.Sync()

	// Parse the object file.
	f, err := parser.ParseFile(fset,
		inputFile,
		nil,
		parser.ParseComments)

	if err != nil {
		fmt.Println("Failed to parse input file ", inputFile, err)
		return
	}

	for _, dec := range f.Decls {
		tk, ok := dec.(*ast.GenDecl)
		if ok {
			for _, spec := range tk.Specs {
				switch spec.(type) {
				case *ast.TypeSpec:
					typ := spec.(*ast.TypeSpec)
					str, ok := typ.Type.(*ast.StructType)
					switch typ.Name.Name {
					case "BGPNeighborConfig", "IPv4Intf", "Vlan", "PortIntfConfig":
						fmt.Printf("%s \n", typ.Name.Name)
						if ok {
							writeResourceHdr(typ.Name.Name, docJsFile)
							for _, fld := range str.Fields.List {
								if fld.Names != nil {
									switch fld.Type.(type) {
									case *ast.Ident:
										fmt.Printf("-- %s \n", fld.Names[0])
										idnt := fld.Type.(*ast.Ident)
										writeAttributeJson(fld.Names[0].Name, idnt.String(), docJsFile)
									}
								}
							}
							docJsFile.WriteString(twoTabs + " ], " + "\n")
							writeEpilogueForStruct(typ.Name.Name, docJsFile)
							writePathCompletion(docJsFile)
						}

					}

				}

			}
		}
	}
	docJsFile.WriteString(twoTabs + " } " + "\n")
	docJsFile.WriteString(twoTabs + " }; " + "\n")
	writeStaticPart("part2.txt", docJsFile)
}
