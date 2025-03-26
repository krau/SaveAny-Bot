package core

import (
	"reflect"
	"testing"

	"github.com/celestix/telegraph-go/v2"
)

func TestGetImgSrcs(t *testing.T) {
	complexStructure := telegraph.NodeElement{
		Tag: "div",
		Children: []telegraph.Node{
			telegraph.NodeElement{
				Tag: "figure",
				Children: []telegraph.Node{
					telegraph.NodeElement{
						Tag: "img",
						Attrs: map[string]string{
							"src": "https://example.com/image1.png",
						},
					},
					telegraph.NodeElement{
						Tag: "p",
						Children: []telegraph.Node{
							"A text node",
						},
					},
					telegraph.NodeElement{
						Tag: "figure",
						Children: []telegraph.Node{
							telegraph.NodeElement{
								Tag: "img",
								Attrs: map[string]string{
									"src": "https://example.com/image2.png",
								},
							},
						},
					},
				},
			},
			telegraph.NodeElement{
				Tag: "img",
				Attrs: map[string]string{
					"src": "https://example.com/image3.png",
				},
			},
			"text node",
			telegraph.NodeElement{
				Tag: "div",
				Children: []telegraph.Node{
					telegraph.NodeElement{
						Tag: "span",
						Children: []telegraph.Node{
							telegraph.NodeElement{
								Tag: "img",
								Attrs: map[string]string{
									"src": "https://example.com/image4.png",
								},
							},
						},
					},
				},
			},
		},
	}

	expected := []string{
		"https://example.com/image1.png",
		"https://example.com/image2.png",
		"https://example.com/image3.png",
		"https://example.com/image4.png",
	}

	got := getNodeImages(complexStructure)

	if !reflect.DeepEqual(expected, got) {
		t.Errorf("expected %v，got %v", expected, got)
	}
}
