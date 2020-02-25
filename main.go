package main

// PDF Structure
// =============
// 	PdfDocument
//		PdfResources
//			PdfFont
//			PdfImage
//		PdfCatalog
//			PdfOutlines
//			PdfPages
//				PdfPage
//					PdfPageContent

import (
	"bytes"
	"fmt"
	"strings"
)

// 14 core fonts
const (
	Courier = iota + 1
	CourierBold
	CourierBoldOblique
	CourierOblique
	Helvetica
	HelveticaBold
	HelveticaBoldOblique
	HelveticaOblique
	TimesRoman
	TimesBold
	TimesItalic
	TimesBoldItalic
	Symbol
	ZapfDingbats
)

var objectIDCounter = 0

func nextObjectID() int {
	objectIDCounter++
	return objectIDCounter
}

// PdfObject is the base object that has an id and a reference to the containing document.
type PdfObject struct {
	objectID int
	document *PdfDocument
}

func (pdfObject PdfObject) objectRef() string {
	return fmt.Sprintf("%v 0 R", pdfObject.objectID)
}

func (pdfObject PdfObject) bytes() []byte {
	var buf bytes.Buffer
	return buf.Bytes()
}

// PdfFont stores the details of one of the 14 base fonts
type PdfFont struct {
	PdfObject
	name     string
	baseFont string
	subtype  string
}

// NewFont creates one of the 14 base fonts
func NewFont(name string, font int) PdfFont {
	var result PdfFont
	switch font {
	case Courier:
		result = PdfFont{name: "/" + name, baseFont: "/Courier", subtype: "/Type1"}
	case CourierBold:
		result = PdfFont{name: "/" + name, baseFont: "/Courier-Bold", subtype: "/Type1"}
	case CourierBoldOblique:
		result = PdfFont{name: "/" + name, baseFont: "/Courier-BoldOblique", subtype: "/Type1"}
	case CourierOblique:
		result = PdfFont{name: "/" + name, baseFont: "/Courier-Oblique", subtype: "/Type1"}
	case Helvetica:
		result = PdfFont{name: "/" + name, baseFont: "/Helvetica", subtype: "/Type1"}
	case HelveticaBold:
		result = PdfFont{name: "/" + name, baseFont: "/Helvetica-Bold", subtype: "/Type1"}
	case HelveticaBoldOblique:
		result = PdfFont{name: "/" + name, baseFont: "/Helvetica-BoldOblique", subtype: "/Type1"}
	case HelveticaOblique:
		result = PdfFont{name: "/" + name, baseFont: "/Helvetica-Oblique", subtype: "/Type1"}
	case TimesRoman:
		result = PdfFont{name: "/" + name, baseFont: "/Times-Roman", subtype: "/Type1"}
	case TimesBold:
		result = PdfFont{name: "/" + name, baseFont: "/Times-Bold", subtype: "/Type1"}
	case TimesBoldItalic:
		result = PdfFont{name: "/" + name, baseFont: "/Times-BoldItalic", subtype: "/Type1"}
	case TimesItalic:
		result = PdfFont{name: "/" + name, baseFont: "/Times-Italic", subtype: "/Type1"}
	default:
		panic("Invalid font " + string(font))
	}
	result.objectID = nextObjectID()
	return result
}

func (pdfFont PdfFont) bytes() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v 0 obj\r\n", pdfFont.objectID)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Type /Font \r\n")
	fmt.Fprintf(&buf, "/Subtype %v \r\n", pdfFont.subtype)
	fmt.Fprintf(&buf, "/Name %v \r\n", pdfFont.name)
	fmt.Fprintf(&buf, "/BaseFont %v \r\n", pdfFont.baseFont)
	fmt.Fprintf(&buf, ">>\r\n")
	fmt.Fprintf(&buf, "endobj\r\n")
	return buf.Bytes()
}

// PdfImage represents an image resource
type PdfImage struct {
	PdfObject
	name   string
	width  int
	height int
}

func (pdfImage PdfImage) bytes() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v 0 obj\r\n", pdfImage.objectID)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Type /XObject\r\n")
	fmt.Fprintf(&buf, "/Subtype /Image\r\n")
	fmt.Fprintf(&buf, "/Name %v\r\n", pdfImage.name)
	fmt.Fprintf(&buf, "/Width %v\r\n", pdfImage.width)
	fmt.Fprintf(&buf, "/Height %v\r\n", pdfImage.height)
	fmt.Fprintf(&buf, "/BitsPerComponent 8\r\n")
	fmt.Fprintf(&buf, "/ColorSpace /DeviceRGB\r\n")
	fmt.Fprintf(&buf, "/Filter [/FlateDecode]\r\n")
	fmt.Fprintf(&buf, "/Predictor 1\r\n")
	fmt.Fprintf(&buf, "/Length 0\r\n")
	fmt.Fprintf(&buf, ">>\r\n")
	fmt.Fprintf(&buf, "stream\r\n")
	// TODO
	fmt.Fprintf(&buf, "endstream\r\n")
	fmt.Fprintf(&buf, "endobj\r\n")
	return buf.Bytes()
}

// PdfPageContent represents the contents of a page.
type PdfPageContent struct {
	PdfObject
	text, lines, graphics string
}

func (content *PdfPageContent) bytes() []byte {
	var buf bytes.Buffer
	stream := "BT\r\n" + content.text + "\r\nET\r\n" + content.lines + "S\r\n" + content.graphics
	fmt.Fprintf(&buf, "%v 0 obj\r\n", content.objectID)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Length %v\r\n", len(stream))
	fmt.Fprintf(&buf, ">>\r\n")
	fmt.Fprintf(&buf, "stream\r\n")
	fmt.Fprint(&buf, stream)
	fmt.Fprintf(&buf, "endstream\r\n")
	fmt.Fprintf(&buf, "endobj\r\n")
	return buf.Bytes()
}

// PdfPage represents a single page
type PdfPage struct {
	PdfObject
	parent                  *PdfPages
	content                 *PdfPageContent
	height, width           int
	x, y                    int
	fontSize                int
	leftMargin, rightMargin int
	topMargin, bottomMargin int
}

func (page *PdfPage) outputText(text string) {
	var sb strings.Builder
	for _, c := range text {
		if c == '(' {
			sb.WriteString("\\(")
		} else if c == ')' {
			sb.WriteString("\\)")
		} else if c == '\\' {
			sb.WriteString("\\\\)")
		} else {
			sb.WriteRune(c)
		}
	}
	page.content.text += fmt.Sprintf("1 0 0 1 %v %v Tm\r\n", page.leftMargin+page.x, page.height-page.topMargin-page.y-page.fontSize)
	page.content.text += fmt.Sprintf("(%s) Tj\r\n", sb.String())
}

func (pdfPage PdfPage) bytes() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v 0 obj\r\n", pdfPage.objectID)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Type /Page\r\n")
	fmt.Fprintf(&buf, "/Parent %v\r\n", pdfPage.parent.objectRef())
	fmt.Fprintf(&buf, "/Resources %v\r\n", pdfPage.document.resources.objectRef())
	fmt.Fprintf(&buf, "/Contents %v\r\n", pdfPage.content.objectRef())
	fmt.Fprintf(&buf, ">>\r\n")
	fmt.Fprintf(&buf, "endobj\r\n")
	return buf.Bytes()
}

// PdfPages represents the list of pages
type PdfPages struct {
	PdfObject
	pages []PdfPage
}

func (pdfPages PdfPages) bytes() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v 0 obj\r\n", pdfPages.objectID)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Type /Pages\r\n")
	fmt.Fprintf(&buf, "/MediaBox [ 0 0 595 842 ]\r\n")
	fmt.Fprintf(&buf, "/Count %v\r\n", len(pdfPages.pages))
	fmt.Fprintf(&buf, "/Kids [ ")
	for _, page := range pdfPages.pages {
		fmt.Fprintf(&buf, page.objectRef()+" ")
	}
	fmt.Fprintf(&buf, "]\r\n")
	fmt.Fprintf(&buf, ">>\r\n")
	fmt.Fprintf(&buf, "endobj\r\n")
	return buf.Bytes()
}

// PdfOutlines ...
type PdfOutlines struct {
	PdfObject
}

func (pdfOutlines PdfOutlines) bytes() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v 0 obj\r\n", pdfOutlines.objectID)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Type /Outlines\r\n")
	fmt.Fprintf(&buf, "/Count 0\r\n") // TODO : Add outlines
	fmt.Fprintf(&buf, ">>\r\n")
	fmt.Fprintf(&buf, "endobj\r\n")
	return buf.Bytes()
}

// PdfCatalog ...
type PdfCatalog struct {
	PdfObject
	outlines *PdfOutlines
	pdfPages *PdfPages
}

func (pdfCatalog PdfCatalog) bytes() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v 0 obj\r\n", pdfCatalog.objectID)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Type /Catalog \r\n")
	fmt.Fprintf(&buf, "/Outlines %v\r\n", pdfCatalog.outlines.objectRef())
	fmt.Fprintf(&buf, "/Pages %v\r\n", pdfCatalog.pdfPages.objectRef())
	fmt.Fprintf(&buf, ">>\r\n")
	fmt.Fprintf(&buf, "endobj\r\n")
	return buf.Bytes()
}

// PdfResources represents the images and fonts for the document
type PdfResources struct {
	PdfObject
	fonts  []PdfFont
	images []PdfImage
}

func (pdfResources PdfResources) bytes() []byte {
	var buf bytes.Buffer
	procset := "[ /PDF "
	if len(pdfResources.fonts) > 0 {
		procset += "/Text "
	}
	if len(pdfResources.images) > 0 {
		procset += "/ImageB "
	}
	procset += "]"

	fmt.Fprintf(&buf, "%v 0 obj\r\n", pdfResources.objectID)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Procset %v\r\n", procset)

	if len(pdfResources.fonts) > 0 {
		fmt.Fprintf(&buf, "/Font << ")
		for _, font := range pdfResources.fonts {
			fmt.Fprintf(&buf, "%v %v ", font.name, font.objectRef())
		}
		fmt.Fprintf(&buf, ">>\r\n")
	}

	if len(pdfResources.images) > 0 {
		fmt.Fprintf(&buf, "/XObject << ")
		for _, image := range pdfResources.images {
			fmt.Fprintf(&buf, "%v %v ", image.name, image.objectRef())
		}
		fmt.Fprintf(&buf, ">>\r\n")
	}

	fmt.Fprintf(&buf, ">>\r\n")
	fmt.Fprintf(&buf, "endobj\r\n")
	return buf.Bytes()

}

// PdfDocument represents the top level document
type PdfDocument struct {
	PdfObject
	resources   *PdfResources
	catalog     *PdfCatalog
	objects     []interface{}
	currentPage *PdfPage
}

// NewPdfDocument creates a new single page document
func NewPdfDocument() PdfDocument {
	document := PdfDocument{}
	document.catalog = new(PdfCatalog)
	document.catalog.objectID = nextObjectID()
	document.catalog.pdfPages = new(PdfPages)
	document.catalog.pdfPages.objectID = nextObjectID()
	document.catalog.outlines = new(PdfOutlines)
	document.catalog.outlines.objectID = nextObjectID()
	document.resources = new(PdfResources)
	document.resources.objectID = nextObjectID()
	document.objects = append(document.objects, document.catalog)
	document.objects = append(document.objects, document.catalog.pdfPages)
	document.objects = append(document.objects, document.catalog.outlines)
	document.objects = append(document.objects, document.resources)
	document.addPage()
	document.addFont("F1", 1)
	return document
}

func (document *PdfDocument) addPage() PdfPage {
	page := PdfPage{
		height:       842,
		width:        595,
		leftMargin:   72,
		rightMargin:  72,
		topMargin:    72,
		bottomMargin: 72,
		fontSize:     10,
	}
	page.objectID = nextObjectID()
	page.parent = document.catalog.pdfPages
	page.document = document
	page.content = new(PdfPageContent)
	page.content.text = "/F1 10 Tf\r\n1 0 0 1 72 -29 Tm\r\n10 TL\r\n"
	page.content.graphics = "0.5 w\r\n"
	page.content.objectID = nextObjectID()
	document.currentPage = &page
	document.catalog.pdfPages.pages = append(document.catalog.pdfPages.pages, page)
	document.objects = append(document.objects, &page)
	document.objects = append(document.objects, page.content)
	return page
}

func (document *PdfDocument) addFont(name string, id int) PdfFont {
	font := NewFont(name, id)
	document.resources.fonts = append(document.resources.fonts, font)
	document.objects = append(document.objects, &font)
	return font
}

// Bytes returns the byte representation of the PdfDocument
func (document PdfDocument) Bytes() []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "%%PDF-1.2\r\n")
	fmt.Fprintf(&buf, "%%\u00e2\u00e3\u00cf\u00d3\r\n")

	xref := make([]int, len(document.objects))

	for i, obj := range document.objects {
		xref[i] = buf.Len()
		font, ok := obj.(*PdfFont)
		if ok {
			fmt.Fprintf(&buf, "%s", string(font.bytes()))
		}
		pages, ok := obj.(*PdfPages)
		if ok {
			fmt.Fprintf(&buf, "%s", string(pages.bytes()))
		}
		page, ok := obj.(*PdfPage)
		if ok {
			fmt.Fprintf(&buf, "%s", string(page.bytes()))
		}
		content, ok := obj.(*PdfPageContent)
		if ok {
			fmt.Fprintf(&buf, "%s", string(content.bytes()))
		}
		catalog, ok := obj.(*PdfCatalog)
		if ok {
			fmt.Fprintf(&buf, "%s", string(catalog.bytes()))
		}
		resources, ok := obj.(*PdfResources)
		if ok {
			fmt.Fprintf(&buf, "%s", string(resources.bytes()))
		}
		outlines, ok := obj.(*PdfOutlines)
		if ok {
			fmt.Fprintf(&buf, "%s", string(outlines.bytes()))
		}
	}

	startxref := buf.Len()

	fmt.Fprintf(&buf, "xref\r\n")
	fmt.Fprintf(&buf, "0 %v \r\n", len(document.objects)+1)
	fmt.Fprintf(&buf, "0000000000 65535 f\r\n")
	for i := range xref {
		fmt.Fprintf(&buf, "%010d 00000 n\r\n", xref[i])
	}
	fmt.Fprintf(&buf, "trailer\r\n")
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Size %v\r\n", len(xref))
	fmt.Fprintf(&buf, "/Root %v\r\n", document.catalog.objectRef())
	fmt.Fprintf(&buf, ">> \r\n")
	fmt.Fprintf(&buf, "startxref\r\n")
	fmt.Fprintf(&buf, "%v\r\n", startxref)
	fmt.Fprintf(&buf, "%%%%EOF\r\n")

	return buf.Bytes()
}

// Test
func main() {
	document := NewPdfDocument()
	document.currentPage.outputText("Hello World")
	fmt.Printf("%v\n", string(document.Bytes()))

}
