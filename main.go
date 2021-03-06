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
	"compress/zlib"
	"encoding/ascii85"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
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

// PdfObjectWriter is an interface that all objects implement to allow us to treat the PDF as a list of objects
// and easily write it out.
type PdfObjectWriter interface {
	setID(id int)
	setDocument(*PdfDocument)
	bytes() []byte
}

// PdfObject is the base object that has an id and a reference to the containing document.
// It implements PdfObjectWriter
type PdfObject struct {
	id       int
	document *PdfDocument
}

func (o *PdfObject) setID(id int) {
	o.id = id
}

func (o *PdfObject) setDocument(d *PdfDocument) {
	o.document = d
}

func (o PdfObject) objectRef() string {
	return fmt.Sprintf("%v 0 R", o.id)
}

func (o PdfObject) bytes() []byte {
	panic(fmt.Sprintf("TODO - write bytes method for %T", o))
}

// PdfFont stores the details of one of the 14 base fonts
type PdfFont struct {
	PdfObject
	name     string
	baseFont string
	subtype  string
	encoding string
}

// NewFont creates one of the 14 base fonts
func NewFont(name string, font int) PdfFont {
	var result PdfFont
	switch font {
	case Courier:
		result = PdfFont{name: name, baseFont: "Courier", subtype: "Type1", encoding: "WinAnsiEncoding"}
	case CourierBold:
		result = PdfFont{name: name, baseFont: "Courier-Bold", subtype: "Type1", encoding: "WinAnsiEncoding"}
	case CourierBoldOblique:
		result = PdfFont{name: name, baseFont: "Courier-BoldOblique", subtype: "Type1", encoding: "WinAnsiEncoding"}
	case CourierOblique:
		result = PdfFont{name: name, baseFont: "Courier-Oblique", subtype: "Type1", encoding: "WinAnsiEncoding"}
	case Helvetica:
		result = PdfFont{name: name, baseFont: "Helvetica", subtype: "Type1", encoding: "WinAnsiEncoding"}
	case HelveticaBold:
		result = PdfFont{name: name, baseFont: "Helvetica-Bold", subtype: "Type1", encoding: "WinAnsiEncoding"}
	case HelveticaBoldOblique:
		result = PdfFont{name: name, baseFont: "Helvetica-BoldOblique", subtype: "Type1", encoding: "WinAnsiEncoding"}
	case HelveticaOblique:
		result = PdfFont{name: name, baseFont: "Helvetica-Oblique", subtype: "Type1", encoding: "WinAnsiEncoding"}
	case TimesRoman:
		result = PdfFont{name: name, baseFont: "Times-Roman", subtype: "Type1", encoding: "WinAnsiEncoding"}
	case TimesBold:
		result = PdfFont{name: name, baseFont: "Times-Bold", subtype: "Type1", encoding: "WinAnsiEncoding"}
	case TimesBoldItalic:
		result = PdfFont{name: name, baseFont: "Times-BoldItalic", subtype: "Type1", encoding: "WinAnsiEncoding"}
	case TimesItalic:
		result = PdfFont{name: name, baseFont: "Times-Italic", subtype: "Type1", encoding: "WinAnsiEncoding"}
	case Symbol:
		result = PdfFont{name: name, baseFont: "Symbol", subtype: "Type1", encoding: "StandardEncoding"}
	case ZapfDingbats:
		result = PdfFont{name: name, baseFont: "ZapfDingbats", subtype: "Type1", encoding: "StandardEncoding"}
	default:
		panic("Invalid font " + string(font))
	}
	return result
}

func (f PdfFont) bytes() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v 0 obj\r\n", f.id)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Type /Font \r\n")
	fmt.Fprintf(&buf, "/Subtype /%v \r\n", f.subtype)
	fmt.Fprintf(&buf, "/Name /%v \r\n", f.name)
	fmt.Fprintf(&buf, "/BaseFont /%v \r\n", f.baseFont)
	if f.encoding != "StandardEncoding" {
		fmt.Fprintf(&buf, "/Encoding /%v\r\n", f.encoding)
	}
	fmt.Fprintf(&buf, ">>\r\n")
	fmt.Fprintf(&buf, "endobj\r\n")
	return buf.Bytes()
}

// PdfImage represents an image resource
type PdfImage struct {
	PdfObject
	name        string
	width       int
	height      int
	ascii85data []byte
}

func (pi *PdfImage) loadImage(name string, filename string) {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	image, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	bounds := image.Bounds()
	pi.name = name
	pi.width = bounds.Size().X
	pi.height = bounds.Size().Y
	rgbdata := make([]byte, 0, pi.height*pi.width*3)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := image.At(x, y).RGBA()
			rgbdata = append(rgbdata, byte(r>>8))
			rgbdata = append(rgbdata, byte(g>>8))
			rgbdata = append(rgbdata, byte(b>>8))
		}
	}
	var compressed bytes.Buffer
	fw := zlib.NewWriter(&compressed)
	fw.Write(rgbdata)
	fw.Close()
	var ascii bytes.Buffer
	encoder := ascii85.NewEncoder(&ascii)
	io.Copy(encoder, bytes.NewReader(compressed.Bytes()))
	encoder.Close()
	pi.ascii85data = ascii.Bytes()

}

func (pi PdfImage) bytes() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v 0 obj\r\n", pi.id)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Type /XObject\r\n")
	fmt.Fprintf(&buf, "/Subtype /Image\r\n")
	fmt.Fprintf(&buf, "/Name /%v\r\n", pi.name)
	fmt.Fprintf(&buf, "/Width %v\r\n", pi.width)
	fmt.Fprintf(&buf, "/Height %v\r\n", pi.height)
	fmt.Fprintf(&buf, "/BitsPerComponent 8\r\n")
	fmt.Fprintf(&buf, "/ColorSpace /DeviceRGB\r\n")
	fmt.Fprintf(&buf, "/Filter [ /ASCII85Decode /FlateDecode ]\r\n")
	fmt.Fprintf(&buf, "/Predictor 1\r\n")
	fmt.Fprintf(&buf, "/Length %v\r\n", len(pi.ascii85data))
	fmt.Fprintf(&buf, ">>\r\n")
	fmt.Fprintf(&buf, "stream\r\n")
	fmt.Fprintf(&buf, "%s", string(pi.ascii85data))
	fmt.Fprintf(&buf, "endstream\r\n")
	fmt.Fprintf(&buf, "endobj\r\n")
	return buf.Bytes()
}

// PdfPageContent represents the contents of a page.
type PdfPageContent struct {
	PdfObject
	text, lines, graphics string
}

func (c *PdfPageContent) bytes() []byte {
	var buf bytes.Buffer
	stream := "BT\r\n" + c.text + "\r\nET\r\n" + c.lines + "S\r\n" + c.graphics
	fmt.Fprintf(&buf, "%v 0 obj\r\n", c.id)
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
	font                    *PdfFont
	fontSize                int
	height, width           int
	x, y                    int
	leftMargin, rightMargin int
	topMargin, bottomMargin int
}

func (p *PdfPage) setFont(name string) {
	for _, f := range p.document.resources.fonts {
		if f.name == name {
			p.font = f
		}
	}
	p.content.text += fmt.Sprintf("/%v %v Tf\r\n", p.font.name, p.fontSize)
}

func (p *PdfPage) setFontSize(size int) {
	p.fontSize = size
	p.content.text += fmt.Sprintf("/%v %v Tf\r\n", p.font.name, p.fontSize)
}

func (p *PdfPage) outputText(text string) {
	var sb strings.Builder
	for i := range text {
		b := text[i]
		if b == '(' {
			sb.WriteString(`\(`)
		} else if b == ')' {
			sb.WriteString(`\)`)
		} else if b == '\\' {
			sb.WriteString(`\\`)
		} else {
			sb.WriteByte(b)
		}
	}
	p.content.text += fmt.Sprintf("1 0 0 1 %v %v Tm\r\n", p.x, p.y)
	p.content.text += fmt.Sprintf("(%s) Tj\r\n", sb.String())
}

func (p *PdfPage) print(text string) {
	p.outputText(text)
	p.x += len(text) * p.fontSize
}

func (p *PdfPage) println(text string) {
	p.outputText(text)
	p.x = p.leftMargin
	p.y -= p.fontSize
}

func (p *PdfPage) drawImage(name string, x, y int) {
	var i *PdfImage
	for _, image := range p.document.resources.images {
		if image.name == name {
			i = image
		}
	}
	w := i.width
	h := i.height

	p.content.graphics += fmt.Sprintf("q\r\n")
	p.content.graphics += fmt.Sprintf("%v 0 0 %v %v %v cm\r\n", w, h, x, y)
	p.content.graphics += fmt.Sprintf("/%v Do\r\n", name)
	p.content.graphics += fmt.Sprintf("Q\r\n")

}

func (p *PdfPage) drawBox(x, y, w, h int) {
	p.content.lines += fmt.Sprintf("%v %v %v %v re\r\n", x, y, w, h)
}

func (p *PdfPage) drawLine(x1, y1, x2, y2 int) {
	p.content.lines += fmt.Sprintf("%v %v m\r\n%v %v l\r\n", x1, y1, x2, y2)
}

func (p *PdfPage) setColour(red, green, blue int) {
	p.content.text += fmt.Sprintf("%v %v %v rg\r\n", red, green, blue)
}

func (p PdfPage) bytes() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v 0 obj\r\n", p.id)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Type /Page\r\n")
	fmt.Fprintf(&buf, "/Parent %v\r\n", p.parent.objectRef())
	fmt.Fprintf(&buf, "/Resources %v\r\n", p.document.resources.objectRef())
	fmt.Fprintf(&buf, "/Contents %v\r\n", p.content.objectRef())
	fmt.Fprintf(&buf, ">>\r\n")
	fmt.Fprintf(&buf, "endobj\r\n")
	return buf.Bytes()
}

// PdfPages represents the list of pages
type PdfPages struct {
	PdfObject
	pages []*PdfPage
}

func (p PdfPages) bytes() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v 0 obj\r\n", p.id)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Type /Pages\r\n")
	fmt.Fprintf(&buf, "/MediaBox [ 0 0 595 842 ]\r\n")
	fmt.Fprintf(&buf, "/Count %v\r\n", len(p.pages))
	fmt.Fprintf(&buf, "/Kids [ ")
	for _, page := range p.pages {
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

func (o PdfOutlines) bytes() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v 0 obj\r\n", o.id)
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

func (c PdfCatalog) bytes() []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%v 0 obj\r\n", c.id)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Type /Catalog \r\n")
	fmt.Fprintf(&buf, "/Outlines %v\r\n", c.outlines.objectRef())
	fmt.Fprintf(&buf, "/Pages %v\r\n", c.pdfPages.objectRef())
	fmt.Fprintf(&buf, ">>\r\n")
	fmt.Fprintf(&buf, "endobj\r\n")
	return buf.Bytes()
}

// PdfResources represents the images and fonts for the document
type PdfResources struct {
	PdfObject
	fonts  []*PdfFont
	images []*PdfImage
}

func (r PdfResources) bytes() []byte {
	var buf bytes.Buffer
	procset := "[ /PDF "
	if len(r.fonts) > 0 {
		procset += "/Text "
	}
	if len(r.images) > 0 {
		procset += "/ImageB "
	}
	procset += "]"

	fmt.Fprintf(&buf, "%v 0 obj\r\n", r.id)
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Procset %v\r\n", procset)

	if len(r.fonts) > 0 {
		fmt.Fprintf(&buf, "/Font << ")
		for _, font := range r.fonts {
			fmt.Fprintf(&buf, "/%v %v ", font.name, font.objectRef())
		}
		fmt.Fprintf(&buf, ">>\r\n")
	}

	if len(r.images) > 0 {
		fmt.Fprintf(&buf, "/XObject << ")
		for _, image := range r.images {
			fmt.Fprintf(&buf, "/%v %v ", image.name, image.objectRef())
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
	objects     []PdfObjectWriter
	currentPage *PdfPage
}

func (d *PdfDocument) addObject(o PdfObjectWriter) {
	o.setID(len(d.objects) + 1)
	o.setDocument(d)
	d.objects = append(d.objects, o)
}

// NewPdfDocument creates a new single page document
func NewPdfDocument() PdfDocument {
	d := PdfDocument{}
	d.catalog = new(PdfCatalog)
	d.addObject(d.catalog)
	d.catalog.pdfPages = new(PdfPages)
	d.addObject(d.catalog.pdfPages)
	d.catalog.outlines = new(PdfOutlines)
	d.addObject(d.catalog.outlines)
	d.resources = new(PdfResources)
	d.addObject(d.resources)
	d.addPage()
	return d
}

func (d *PdfDocument) addPage() PdfPage {
	// measurements are in points
	p := PdfPage{
		height:       842,
		width:        595,
		leftMargin:   72,
		rightMargin:  72,
		topMargin:    72,
		bottomMargin: 72,
		fontSize:     10,
	}
	p.parent = d.catalog.pdfPages
	p.document = d
	p.x = p.leftMargin
	p.y = p.height - p.topMargin - p.fontSize
	p.content = new(PdfPageContent)
	p.content.text = "/F1 10 Tf\r\n1 0 0 1 72 -29 Tm\r\n10 TL\r\n"
	p.content.graphics = "0.5 w\r\n"
	d.currentPage = &p
	d.catalog.pdfPages.pages = append(d.catalog.pdfPages.pages, &p)
	d.addObject(&p)
	d.addObject(p.content)
	return p
}

func (d *PdfDocument) addFont(name string, id int) PdfFont {
	font := NewFont(name, id)
	d.addObject(&font)
	d.resources.fonts = append(d.resources.fonts, &font)
	return font
}

func (d *PdfDocument) addImage(name string, filename string) PdfImage {
	i := PdfImage{name: name}
	i.loadImage(name, filename)
	d.addObject(&i)
	d.resources.images = append(d.resources.images, &i)
	return i
}

// Bytes returns the byte representation of the PdfDocument
func (d PdfDocument) Bytes() []byte {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "%%PDF-1.2\r\n")
	fmt.Fprintf(&buf, "%%\u00e2\u00e3\u00cf\u00d3\r\n")

	xref := make([]int, len(d.objects))

	for i, obj := range d.objects {
		xref[i] = buf.Len()
		fmt.Fprintf(&buf, "%s", obj.bytes())
	}

	startxref := buf.Len()

	fmt.Fprintf(&buf, "xref\r\n")
	fmt.Fprintf(&buf, "0 %v \r\n", len(d.objects)+1)
	fmt.Fprintf(&buf, "0000000000 65535 f\r\n")
	for i := range xref {
		fmt.Fprintf(&buf, "%010d 00000 n\r\n", xref[i])
	}
	fmt.Fprintf(&buf, "trailer\r\n")
	fmt.Fprintf(&buf, "<<\r\n")
	fmt.Fprintf(&buf, "/Size %v\r\n", len(xref))
	fmt.Fprintf(&buf, "/Root %v\r\n", d.catalog.objectRef())
	fmt.Fprintf(&buf, ">> \r\n")
	fmt.Fprintf(&buf, "startxref\r\n")
	fmt.Fprintf(&buf, "%v\r\n", startxref)
	fmt.Fprintf(&buf, "%%%%EOF\r\n")

	return buf.Bytes()
}

// Test
func main() {
	var charset [256]byte
	for i := range charset {
		charset[i] = byte(i)
	}
	document := NewPdfDocument()
	page := document.currentPage
	document.addFont("Courier", Courier)
	document.addFont("CourierBold", CourierBold)
	document.addFont("TimesRoman", TimesRoman)
	document.addFont("TimesBold", TimesBold)
	document.addFont("Symbol", Symbol)
	document.addFont("Dingbats", ZapfDingbats)
	document.addImage("gopher", "gopher.jpg")

	page.setFont("CourierBold")
	page.println("Courier")
	page.setFont("Courier")
	for i := 0; i < len(charset); i += 16 {
		s := fmt.Sprintf("%2X %s\r\n", i, string(charset[i:i+16]))
		page.println(s)
	}
	page.println("")

	page.setFont("TimesBold")
	page.println("Times Roman")
	page.setFont("TimesRoman")
	for i := 0; i < len(charset); i += 16 {
		s := fmt.Sprintf("%2X %s\r\n", i, string(charset[i:i+16]))
		page.println(s)
	}
	page.println("")

	page.setFont("TimesBold")
	page.println("Symbol")
	page.setFont("Symbol")
	for i := 0; i < len(charset); i += 16 {
		s := fmt.Sprintf("%2X %s\r\n", i, string(charset[i:i+16]))
		page.println(s)
	}
	page.println("")

	page.setFont("TimesBold")
	page.println("Dingbats")
	page.setFont("Dingbats")
	for i := 0; i < len(charset); i += 16 {
		s := fmt.Sprintf("%2X %s\r\n", i, string(charset[i:i+16]))
		page.println(s)
	}
	page.println("")

	page.drawImage("gopher", 250, 550)
	page.drawBox(250, 500, 300, 20)
	page.drawLine(250, 480, 550, 480)

	page.setFont("TimesBold")
	page.setFontSize(18)
	page.setColour(255, 0, 0)
	page.x = 300
	page.y = 440
	page.print("Red")
	page.setColour(0, 255, 0)
	page.x = 300
	page.y = 420
	page.print("Green")
	page.setColour(0, 0, 255)
	page.x = 300
	page.y = 400
	page.print("Blue")

	fmt.Printf("%v\n", string(document.Bytes()))
}
