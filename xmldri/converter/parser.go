package converter

import (
	"encoding/csv"
	"encoding/xml"
	"fmt"
	"mime/multipart"
	"net/http"
)

func ProcessUploadedCsvAndGenerateXml(file *multipart.FileHeader) ([]byte, int, string) {

	// Open the uploaded file
	uploadedFile, err := file.Open()
	if err != nil {
		return nil, http.StatusInternalServerError, err.Error()
	}

	defer uploadedFile.Close()

	// Create a CSV reader
	csvReader := csv.NewReader(uploadedFile)

	// Create a CSV reader
	xml, statusCode, errMessage := processCsvAndGenerateXml(csvReader)

	return xml, statusCode, errMessage

}

func processCsvAndGenerateXml(reader *csv.Reader) ([]byte, int, string) {

	reader.Comma = ';'

	// Read all records from CSV
	records, err := reader.ReadAll()
	if err != nil {
		return nil, http.StatusInternalServerError, err.Error()
	}

	var xmlData = Dr{}

	// Create a slice to store records
	var allQuadro0405Items []TableItem

	// Iterate through each record and fill the struct
	for ndx, row := range records {
		// Convert string fields to int where necessary
		if ndx > 0 {
			if ndx == 1 {
				xmlData.Xmlns = "http://www.at.gov.pt/2019/DRIVAWeb/schema"
				xmlData.Version = "1.0"

				xmlData.Rosto.Quadro01.F1 = row[1]
				xmlData.Rosto.Quadro02.F1 = row[2]
				xmlData.Rosto.Quadro03.F1 = row[3]
				// --Add leading zero if necessary
				strNumber := row[4]
				if len(strNumber) == 1 {
					strNumber = "0" + row[4]
				}
				xmlData.Rosto.Quadro03.F2 = strNumber
				xmlData.Rosto.Quadro06.F1 = row[13]
				xmlData.Rosto.Quadro0405.F10 = row[5]
				xmlData.Rosto.Quadro0405.F17 = row[6]
				xmlData.Rosto.Quadro0405.F18 = row[7]
				xmlData.Rosto.Quadro0405.F19 = row[8]
			}

			// Create a new record table item
			record := TableItem{}

			record.F2 = row[9]
			record.F3 = row[10]
			record.F4 = row[11]
			record.F5 = row[12]

			// Append the record to the slice
			allQuadro0405Items = append(allQuadro0405Items, record)
		}

	}
	xmlData.Rosto.Quadro0405.Table.TableItems = allQuadro0405Items
	// Convert the struct to XML
	xmlConverted, err := xml.MarshalIndent(xmlData, "", "    ")
	if err != nil {
		errMsg := fmt.Sprintf("Error marshaling to XML: %s", err)
		return nil, http.StatusInternalServerError, errMsg
	}
	/*
		// Write the XML data to a file
		xmlFile, err := os.Create("../output.xml")
		if err != nil {
			errMsg := fmt.Sprintf("Error creating XML file. Error: %s", err)
			return nil, http.StatusInternalServerError, errMsg
		}
		defer xmlFile.Close()

		// Write the XML data to the file
		_, err = xmlFile.Write(xmlConverted)
		if err != nil {
			errMsg := fmt.Sprintf("Error writing XML data to file. Error: %s", err)
			return nil, http.StatusInternalServerError, errMsg

		}
	*/
	return xmlConverted, http.StatusOK, "XML data has been successful generated"
}
