package converter

import "encoding/xml"

type Dr struct {
	XMLName xml.Name `xml:"dr"`
	Text    string   `xml:",chardata"`
	Xmlns   string   `xml:"xmlns,attr"`
	Version string   `xml:"version,attr"`
	Rosto   struct {
		Text     string `xml:",chardata"`
		Quadro01 struct {
			Text string `xml:",chardata"`
			F1   string `xml:"f1"`
		} `xml:"quadro01"`
		Quadro02 struct {
			Text string `xml:",chardata"`
			F1   string `xml:"f1"`
		} `xml:"quadro02"`
		Quadro03 struct {
			Text string `xml:",chardata"`
			F1   string `xml:"f1"`
			F2   string `xml:"f2"`
		} `xml:"quadro03"`
		Quadro0405 struct {
			Text  string `xml:",chardata"`
			F10   string `xml:"f10"`
			F17   string `xml:"f17"`
			F18   string `xml:"f18"`
			F19   string `xml:"f19"`
			Table struct {
				Text       string      `xml:",chardata"`
				TableItems []TableItem `xml:"tableItem"`
			} `xml:"table"`
		} `xml:"quadro0405"`
		Quadro06 struct {
			Text string `xml:",chardata"`
			F1   string `xml:"f1"`
		} `xml:"quadro06"`
		Quadro07 struct {
			Text  string `xml:",chardata"`
			Table string `xml:"table"`
		} `xml:"quadro07"`
	} `xml:"rosto"`
}

type TableItem struct {
	Text string `xml:",chardata"`
	F2   string `xml:"f2"`
	F3   string `xml:"f3"`
	F4   string `xml:"f4"`
	F5   string `xml:"f5"`
}
