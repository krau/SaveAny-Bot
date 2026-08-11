package tgutil

import (
	"testing"

	"github.com/gotd/td/tg"
)

func TestEscapeHTMLTemplateDataDoesNotMutateInput(t *testing.T) {
	input := map[string]any{
		"Text":  `<b>A&B</b>`,
		"Count": 2,
	}
	escaped := EscapeHTMLTemplateData(input)

	if got, want := escaped["Text"], "&lt;b&gt;A&amp;B&lt;/b&gt;"; got != want {
		t.Fatalf("escaped text = %q, want %q", got, want)
	}
	if got := input["Text"]; got != `<b>A&B</b>` {
		t.Fatalf("input was mutated: %q", got)
	}
	if got := escaped["Count"]; got != 2 {
		t.Fatalf("non-string value = %v, want 2", got)
	}
}

func TestRenderHTMLUsesTemplateStylesAndDecodesValues(t *testing.T) {
	data := EscapeHTMLTemplateData(map[string]any{"Name": `<b>A&B</b>.bin`})
	markup := `<blockquote><b>Uploading</b>
<code>` + data["Name"].(string) + `</code></blockquote>`

	text, entities, err := RenderHTML(markup)
	if err != nil {
		t.Fatalf("RenderHTML() failed: %v", err)
	}
	if want := "Uploading\n<b>A&B</b>.bin"; text != want {
		t.Fatalf("rendered text = %q, want %q", text, want)
	}

	var bold, code, blockquote int
	for _, messageEntity := range entities {
		switch messageEntity.(type) {
		case *tg.MessageEntityBold:
			bold++
		case *tg.MessageEntityCode:
			code++
		case *tg.MessageEntityBlockquote:
			blockquote++
		}
	}
	if bold != 1 || code != 1 || blockquote != 1 {
		t.Fatalf("entity counts = bold:%d code:%d blockquote:%d", bold, code, blockquote)
	}
}
