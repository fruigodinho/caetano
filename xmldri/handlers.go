package xmldri

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fruigodinho/caetano/xmldri/converter"
	"github.com/fruigodinho/caetano/xmldri/library/utilities"
)

func csvHandler(c *gin.Context) {
	c.HTML(http.StatusOK, TemplateUpload, gin.H{
		"Layout": "authenticated",
		"Title":  "Conversor CSV → XML",
		"Nav":    []NavLink{},
		"year":   utilities.Year(),
	})
}

func csvUploadHandler(c *gin.Context) {
	file, err := c.FormFile("csvfile")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	xmlBytes, status, message := converter.ProcessUploadedCsvAndGenerateXml(file)
	if status != http.StatusOK {
		c.JSON(status, gin.H{"error": message})
		return
	}

	fileName := utilities.DateNow() + "-" + strings.ReplaceAll(file.Filename, ".csv", ".xml")

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/octet-stream")
	c.Writer.Write(xmlBytes)
}
