package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestCol(t *testing.T) {
	for i, want := range map[int]string{0: "A", 25: "Z", 26: "AA", 27: "AB", 51: "AZ", 52: "BA", 701: "ZZ", 702: "AAA"} {
		if got := col(i); got != want {
			t.Errorf("col(%d) = %q, want %q", i, got, want)
		}
	}
}

func TestSheetName(t *testing.T) {
	used := map[string]bool{}
	if got := sheetName("Ekim/Kasım [2026]?", 1, used); got != "Ekim-Kasım -2026--" {
		t.Errorf("got %q", got)
	}
	long := strings.Repeat("ş", 40)
	a := sheetName(long, 2, used)
	b := sheetName(long, 3, used)
	if len([]rune(a)) != 31 || len([]rune(b)) != 31 || a == b {
		t.Errorf("long names %q / %q", a, b)
	}
}

// TestWorkbookIsWellFormed unzips a workbook and parses every part as XML.
func TestWorkbookIsWellFormed(t *testing.T) {
	var buf bytes.Buffer
	err := Write(&buf, []Sheet{
		{Name: "Özet", Widths: []float64{20, 0, 12}, Freeze: 1, Rows: [][]Cell{
			{{V: "Başlık <&> \"x\"", Style: Header}, {V: 1.5, Style: Percent}, {V: 1234567.891, Style: Money}},
			{S(" boşluklu "), N(3, Plain), {}, {Style: Total}, {V: 7}, {V: "kontrol\x01karakter"}},
		}},
		{Name: "Özet", Rows: [][]Cell{{S("aynı isim")}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	z, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{
		"[Content_Types].xml": false, "_rels/.rels": false, "xl/workbook.xml": false,
		"xl/_rels/workbook.xml.rels": false, "xl/styles.xml": false,
		"xl/worksheets/sheet1.xml": false, "xl/worksheets/sheet2.xml": false,
	}
	var sheet1, workbook string
	for _, f := range z.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(rc)
		rc.Close()
		dec := xml.NewDecoder(bytes.NewReader(body))
		for {
			if _, err := dec.Token(); err == io.EOF {
				break
			} else if err != nil {
				t.Fatalf("%s: %v", f.Name, err)
			}
		}
		want[f.Name] = true
		switch f.Name {
		case "xl/worksheets/sheet1.xml":
			sheet1 = string(body)
		case "xl/workbook.xml":
			workbook = string(body)
		}
	}
	for name, seen := range want {
		if !seen {
			t.Errorf("missing part %s", name)
		}
	}
	for _, frag := range []string{
		`<c r="A1" s="1" t="inlineStr"><is><t>Başlık &lt;&amp;&gt; &quot;x&quot;</t></is></c>`,
		`<c r="B1" s="3"><v>1.5</v></c>`,
		`<c r="C1" s="2"><v>1234567.891</v></c>`,
		`<t xml:space="preserve"> boşluklu </t>`,
		`<c r="D2" s="4"/>`,
		`<t>kontrolkarakter</t>`,
		`<pane ySplit="1" topLeftCell="A2" activePane="bottomLeft" state="frozen"/>`,
	} {
		if !strings.Contains(sheet1, frag) {
			t.Errorf("sheet1 lacks %s", frag)
		}
	}
	if strings.Contains(sheet1, `r="C2"`) {
		t.Error("an empty unstyled cell was written")
	}
	if !strings.Contains(workbook, `name="Özet"`) || !strings.Contains(workbook, `name="Özet (2)"`) {
		t.Errorf("sheet names not de-duplicated: %s", workbook)
	}
}
