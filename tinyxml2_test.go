package tinyxml2

import (
	"strings"
	"testing"
)

func TestProgrammaticDOM(t *testing.T) {
	doc := NewDocument()

	book := doc.NewElement("book")
	_ = doc.SetAttribute(book, "category", "philosophy")
	_ = doc.SetAttribute(book, "year", "2026")
	_ = doc.InsertEndChild(0, book)

	title := doc.NewElement("title")
	titleText := doc.NewText("The Art of War")
	_ = doc.InsertEndChild(title, titleText)
	_ = doc.InsertEndChild(book, title)

	author := doc.NewElement("author")
	authorText := doc.NewText("Sun Tzu")
	_ = doc.InsertEndChild(author, authorText)
	_ = doc.InsertEndChild(book, author)

	root := doc.RootElement()
	if root != book {
		t.Fatalf("expected root element %d, got %d", book, root)
	}

	cat, ok := doc.GetAttribute(root, "category")
	if !ok || cat != "philosophy" {
		t.Errorf("expected category=philosophy, got %s (ok=%v)", cat, ok)
	}

	firstElem := doc.FirstChildElement(root, "title")
	if firstElem == NodeIDNull {
		t.Fatal("expected to find title child element")
	}
	if text := doc.GetText(firstElem); text != "The Art of War" {
		t.Errorf("expected title text 'The Art of War', got '%s'", text)
	}

	nextElem := doc.NextSiblingElement(firstElem, "author")
	if nextElem == NodeIDNull {
		t.Fatal("expected to find author sibling element")
	}
	if text := doc.GetText(nextElem); text != "Sun Tzu" {
		t.Errorf("expected author text 'Sun Tzu', got '%s'", text)
	}

	rendered := doc.RenderCompact()
	expected := `<book category="philosophy" year="2026"><title>The Art of War</title><author>Sun Tzu</author></book>`
	if rendered != expected {
		t.Errorf("rendered compact mismatch:\ngot:  %s\nwant: %s", rendered, expected)
	}
}

func TestParseAndRenderRoundtrip(t *testing.T) {
	inputXML := `<?xml version="1.0"?>
<library name="Alexandria">
    <!-- Ancient collection of manuscripts -->
    <book id="1">
        <title>Elements</title>
        <author>Euclid</author>
    </book>
    <book id="2">
        <title>Republic</title>
        <author>Plato</author>
    </book>
</library>`

	doc := NewDocument()
	err := doc.Parse(inputXML)
	if err != nil {
		t.Fatalf("failed to parse XML: %v", err)
	}

	root := doc.RootElement()
	if root == NodeIDNull {
		t.Fatal("expected root element library")
	}
	if doc.Nodes[root].Name != "library" {
		t.Errorf("expected root name library, got %s", doc.Nodes[root].Name)
	}

	libName, ok := doc.GetAttribute(root, "name")
	if !ok || libName != "Alexandria" {
		t.Errorf("expected library name Alexandria, got %s", libName)
	}

	book1 := doc.FirstChildElement(root, "book")
	if book1 == NodeIDNull {
		t.Fatal("expected book 1")
	}
	id1, _ := doc.GetAttribute(book1, "id")
	if id1 != "1" {
		t.Errorf("expected id=1, got %s", id1)
	}

	title1 := doc.FirstChildElement(book1, "title")
	if doc.GetText(title1) != "Elements" {
		t.Errorf("expected book 1 title Elements, got %s", doc.GetText(title1))
	}

	book2 := doc.NextSiblingElement(book1, "book")
	if book2 == NodeIDNull {
		t.Fatal("expected book 2")
	}
	id2, _ := doc.GetAttribute(book2, "id")
	if id2 != "2" {
		t.Errorf("expected id=2, got %s", id2)
	}

	rendered := doc.Render()
	if !strings.Contains(rendered, "Alexandria") || !strings.Contains(rendered, "Euclid") {
		t.Errorf("rendered output missing expected strings:\n%s", rendered)
	}
}

func TestSafetyAndErrorHandling(t *testing.T) {
	doc := NewDocument()
	elem := doc.NewElement("node")

	// Attempt inserting node into itself
	if err := doc.InsertEndChild(elem, elem); err == nil {
		t.Error("expected error when inserting node into itself, got nil")
	}

	// Attempt out-of-bounds parent
	if err := doc.InsertEndChild(NodeID(9999), elem); err == nil {
		t.Error("expected error for invalid parent, got nil")
	}

	// Attempt out-of-bounds child
	if err := doc.InsertEndChild(elem, NodeID(9999)); err == nil {
		t.Error("expected error for invalid child, got nil")
	}

	// Attempt setting attribute on non-element
	textNode := doc.NewText("hello")
	if err := doc.SetAttribute(textNode, "attr", "val"); err == nil {
		t.Error("expected error setting attribute on text node, got nil")
	}
}

func BenchmarkDOMCreation(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		doc := NewDocument()
		root := doc.NewElement("root")
		_ = doc.InsertEndChild(0, root)
		for j := 0; j < 100; j++ {
			elem := doc.NewElement("item")
			_ = doc.SetAttribute(elem, "idx", "test")
			text := doc.NewText("content")
			_ = doc.InsertEndChild(elem, text)
			_ = doc.InsertEndChild(root, elem)
		}
	}
}

func BenchmarkParseAndRender(b *testing.B) {
	inputXML := `<data><entry id="1"><val>100</val></entry><entry id="2"><val>200</val></entry></data>`
	doc := NewDocument()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = doc.Parse(inputXML)
		_ = doc.RenderCompact()
	}
}
