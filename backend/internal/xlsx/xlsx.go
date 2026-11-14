// Package xlsx writes small .xlsx workbooks with the standard library alone:
// a zip of a handful of SpreadsheetML parts, strings inline, a fixed set of
// cell styles. Enough for a report people open in Excel, nothing more.
package xlsx

import (
	"archive/zip"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Style picks one of the fixed cell formats in styles.xml below.
type Style int

const (
	Plain   Style = iota
	Header        // bold on a light fill — column headings
	Money         // #,##0.00
	Percent       // 1.5 shows as 1,50%  (the value is already in percent)
	Total         // bold money on the header fill
	Title         // large bold
	Bold
)

// Cell is a string, a number (any int or float), or nil for an empty cell.
type Cell struct {
	V     any
	Style Style
}

func S(v string) Cell           { return Cell{V: v} }
func N(v float64, s Style) Cell { return Cell{V: v, Style: s} }

type Sheet struct {
	Name   string
	Widths []float64 // column widths in characters; 0 leaves the default
	Freeze int       // rows frozen at the top
	Rows   [][]Cell
}

// Write streams the workbook to w.
func Write(w io.Writer, sheets []Sheet) error {
	z := zip.NewWriter(w)
	part := func(name, body string) error {
		f, err := z.Create(name)
		if err != nil {
			return err
		}
		_, err = io.WriteString(f, body)
		return err
	}

	var types, sheetList, rels strings.Builder
	names := map[string]bool{}
	for i, sh := range sheets {
		n := i + 1
		name := sheetName(sh.Name, n, names)
		fmt.Fprintf(&types, `<Override PartName="/xl/worksheets/sheet%d.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/>`, n)
		fmt.Fprintf(&sheetList, `<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, esc(name), n, n)
		fmt.Fprintf(&rels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, n, n)
		if err := part(fmt.Sprintf("xl/worksheets/sheet%d.xml", n), sheetXML(sh)); err != nil {
			return err
		}
	}
	fmt.Fprintf(&rels, `<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`, len(sheets)+1)

	parts := []struct{ name, body string }{
		{"[Content_Types].xml", xmlHead + `<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">` +
			`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>` +
			`<Default Extension="xml" ContentType="application/xml"/>` +
			`<Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>` +
			`<Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/>` +
			types.String() + `</Types>`},
		{"_rels/.rels", xmlHead + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>` +
			`</Relationships>`},
		{"xl/workbook.xml", xmlHead + `<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">` +
			`<sheets>` + sheetList.String() + `</sheets></workbook>`},
		{"xl/_rels/workbook.xml.rels", xmlHead + `<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">` +
			rels.String() + `</Relationships>`},
		{"xl/styles.xml", stylesXML},
	}
	for _, p := range parts {
		if err := part(p.name, p.body); err != nil {
			return err
		}
	}
	return z.Close()
}

const xmlHead = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>` + "\n"

// The cellXfs order is the Style constants' order.
const stylesXML = xmlHead + `<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">` +
	`<numFmts count="2"><numFmt numFmtId="164" formatCode="#,##0.00"/><numFmt numFmtId="165" formatCode="0.00&quot;%&quot;"/></numFmts>` +
	`<fonts count="3"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="14"/><name val="Calibri"/></font></fonts>` +
	`<fills count="3"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="gray125"/></fill>` +
	`<fill><patternFill patternType="solid"><fgColor rgb="FFE8ECF4"/><bgColor indexed="64"/></patternFill></fill></fills>` +
	`<borders count="1"><border><left/><right/><top/><bottom/><diagonal/></border></borders>` +
	`<cellStyleXfs count="1"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/></cellStyleXfs>` +
	`<cellXfs count="7">` +
	`<xf numFmtId="0" fontId="0" fillId="0" borderId="0" xfId="0"/>` +
	`<xf numFmtId="0" fontId="1" fillId="2" borderId="0" xfId="0" applyFont="1" applyFill="1"/>` +
	`<xf numFmtId="164" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/>` +
	`<xf numFmtId="165" fontId="0" fillId="0" borderId="0" xfId="0" applyNumberFormat="1"/>` +
	`<xf numFmtId="164" fontId="1" fillId="2" borderId="0" xfId="0" applyNumberFormat="1" applyFont="1" applyFill="1"/>` +
	`<xf numFmtId="0" fontId="2" fillId="0" borderId="0" xfId="0" applyFont="1"/>` +
	`<xf numFmtId="0" fontId="1" fillId="0" borderId="0" xfId="0" applyFont="1"/>` +
	`</cellXfs>` +
	`<cellStyles count="1"><cellStyle name="Normal" xfId="0" builtinId="0"/></cellStyles>` +
	`</styleSheet>`

func sheetXML(sh Sheet) string {
	var b strings.Builder
	b.WriteString(xmlHead)
	b.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">`)
	if sh.Freeze > 0 {
		fmt.Fprintf(&b, `<sheetViews><sheetView workbookViewId="0"><pane ySplit="%d" topLeftCell="A%d" activePane="bottomLeft" state="frozen"/></sheetView></sheetViews>`, sh.Freeze, sh.Freeze+1)
	}
	if len(sh.Widths) > 0 {
		b.WriteString(`<cols>`)
		for i, wd := range sh.Widths {
			if wd > 0 {
				fmt.Fprintf(&b, `<col min="%d" max="%d" width="%g" customWidth="1"/>`, i+1, i+1, wd)
			}
		}
		b.WriteString(`</cols>`)
	}
	b.WriteString(`<sheetData>`)
	for r, row := range sh.Rows {
		fmt.Fprintf(&b, `<row r="%d">`, r+1)
		for c, cell := range row {
			ref := col(c) + strconv.Itoa(r+1)
			style := ""
			if cell.Style != Plain {
				style = fmt.Sprintf(` s="%d"`, cell.Style)
			}
			switch v := cell.V.(type) {
			case nil:
				if style != "" {
					fmt.Fprintf(&b, `<c r="%s"%s/>`, ref, style)
				}
			case string:
				sp := ""
				if strings.TrimSpace(v) != v {
					sp = ` xml:space="preserve"`
				}
				fmt.Fprintf(&b, `<c r="%s"%s t="inlineStr"><is><t%s>%s</t></is></c>`, ref, style, sp, esc(v))
			default:
				f, ok := number(v)
				if !ok {
					fmt.Fprintf(&b, `<c r="%s"%s t="inlineStr"><is><t>%s</t></is></c>`, ref, style, esc(fmt.Sprint(v)))
					continue
				}
				fmt.Fprintf(&b, `<c r="%s"%s><v>%s</v></c>`, ref, style, strconv.FormatFloat(f, 'f', -1, 64))
			}
		}
		b.WriteString(`</row>`)
	}
	b.WriteString(`</sheetData></worksheet>`)
	return b.String()
}

func number(v any) (float64, bool) {
	var f float64
	switch x := v.(type) {
	case float64:
		f = x
	case float32:
		f = float64(x)
	case int:
		f = float64(x)
	case int64:
		f = float64(x)
	default:
		return 0, false
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

// col turns a 0-based column index into its letters: 0 → A, 26 → AA.
func col(i int) string {
	s := ""
	for i++; i > 0; i = (i - 1) / 26 {
		s = string(rune('A'+(i-1)%26)) + s
	}
	return s
}

// sheetName makes a name Excel accepts: no []:*?/\, at most 31 characters,
// unique in the workbook.
func sheetName(name string, n int, used map[string]bool) string {
	name = strings.Map(func(r rune) rune {
		if strings.ContainsRune(`[]:*?/\`, r) {
			return '-'
		}
		return r
	}, strings.TrimSpace(name))
	if name == "" {
		name = fmt.Sprintf("Sayfa%d", n)
	}
	for utf8.RuneCountInString(name) > 31 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	base := name
	for i := 2; used[strings.ToLower(name)]; i++ {
		suffix := fmt.Sprintf(" (%d)", i)
		name = base
		for utf8.RuneCountInString(name)+len(suffix) > 31 {
			_, size := utf8.DecodeLastRuneInString(name)
			name = name[:len(name)-size]
		}
		name += suffix
	}
	used[strings.ToLower(name)] = true
	return name
}

// esc escapes text for XML and drops the control characters XML 1.0 forbids.
func esc(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '&':
			b.WriteString("&amp;")
		case r == '<':
			b.WriteString("&lt;")
		case r == '>':
			b.WriteString("&gt;")
		case r == '"':
			b.WriteString("&quot;")
		case r == '\t' || r == '\n' || r == '\r':
			b.WriteRune(r)
		case r < 0x20 || r == 0xFFFE || r == 0xFFFF || r == utf8.RuneError:
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
