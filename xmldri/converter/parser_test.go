package converter

import (
	"encoding/csv"
	"fmt"
	"os"
	"testing"
)

func TestProcessCsvAndGenerateXml(t *testing.T) {
	// Open the CSV file
	file, err := os.Open("../data.csv")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	processCsvAndGenerateXml(reader)

	// assert.Equal(t, http.StatusForbidden, w.Code)
	// assert.Equal(t, "Forbidden: Route Not Available.", w.Body.String())
}
