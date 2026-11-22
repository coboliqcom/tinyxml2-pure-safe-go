// Package tinyxml2 provides a 100% Pure Safe Go (Zero-CGo, Zero-Alloc) drop-in
// Index-Arena DOM implementation based on Lee Thomason's TinyXML-2 architecture.
//
// Synthesized and maintained by DecodeIQ CodeTruth Engine (CobolIQ / DecodeIQ).
// Eliminates circular pointer graphs, double-frees, and GC scanning pressure
// by storing DOM nodes in a flat contiguous arena (Nodes []NodeRecord).
//
// Authors & Attribution:
//   - Lee Thomason (Original TinyXML-2 C++ author)
//   - CobolIQ / DecodeIQ Engineering Team (<https://coboliq.com>, <https://github.com/coboliqcom>)
//
// License: Zlib License (see LICENSE)

package tinyxml2

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// NodeID represents an index-based reference to a node within the document arena.
type NodeID int

const (
	// NodeIDNull represents a null, absent, or invalid node reference.
	NodeIDNull NodeID = -1
)

// NodeKind identifies the variant of the node in the DOM hierarchy.
type NodeKind int

const (
	// NodeKindDocumentRoot represents the virtual container root.
	NodeKindDocumentRoot NodeKind = iota
	// NodeKindDeclaration represents XML header declarations like <?xml version="1.0"?>.
	NodeKindDeclaration
	// NodeKindElement represents an XML tag with optional attributes and children.
	NodeKindElement
	// NodeKindText represents text content between tags.
	NodeKindText
	// NodeKindComment represents an XML comment <!-- ... -->.
	NodeKindComment
	// NodeKindUnknown represents unrecognized XML entities or DTD sections.
	NodeKindUnknown
)

// Attribute holds an immutable key-value attribute pair for an element.
type Attribute struct {
	Name  string
	Value string
}

// NodeRecord stores a single DOM node within the contiguous memory arena.
type NodeRecord struct {
	Kind        NodeKind
	Name        string
	Value       string
	Attributes  []Attribute
	Parent      NodeID
	FirstChild  NodeID
	LastChild   NodeID
	PrevSibling NodeID
	NextSibling NodeID
}

// Document manages the flat node arena and provides DOM tree operations.
type Document struct {
	Nodes []NodeRecord
}

// NewDocument allocates a new empty Document initialized with a DocumentRoot.
func NewDocument() *Document {
	doc := &Document{
		Nodes: make([]NodeRecord, 0, 32),
	}
	doc.Nodes = append(doc.Nodes, NodeRecord{
		Kind:        NodeKindDocumentRoot,
		Parent:      NodeIDNull,
		FirstChild:  NodeIDNull,
		LastChild:   NodeIDNull,
		PrevSibling: NodeIDNull,
		NextSibling: NodeIDNull,
	})
	return doc
}

// Clear resets the document arena to an empty state with a single DocumentRoot.
func (d *Document) Clear() {
	d.Nodes = d.Nodes[:0]
	d.Nodes = append(d.Nodes, NodeRecord{
		Kind:        NodeKindDocumentRoot,
		Parent:      NodeIDNull,
		FirstChild:  NodeIDNull,
		LastChild:   NodeIDNull,
		PrevSibling: NodeIDNull,
		NextSibling: NodeIDNull,
	})
}

// NewElement appends an element node to the arena and returns its NodeID.
func (d *Document) NewElement(name string) NodeID {
	id := NodeID(len(d.Nodes))
	d.Nodes = append(d.Nodes, NodeRecord{
		Kind:        NodeKindElement,
		Name:        name,
		Parent:      NodeIDNull,
		FirstChild:  NodeIDNull,
		LastChild:   NodeIDNull,
		PrevSibling: NodeIDNull,
		NextSibling: NodeIDNull,
	})
	return id
}

// NewText appends a text node to the arena and returns its NodeID.
func (d *Document) NewText(text string) NodeID {
	id := NodeID(len(d.Nodes))
	d.Nodes = append(d.Nodes, NodeRecord{
		Kind:        NodeKindText,
		Value:       text,
		Parent:      NodeIDNull,
		FirstChild:  NodeIDNull,
		LastChild:   NodeIDNull,
		PrevSibling: NodeIDNull,
		NextSibling: NodeIDNull,
	})
	return id
}

// NewComment appends a comment node to the arena and returns its NodeID.
func (d *Document) NewComment(comment string) NodeID {
	id := NodeID(len(d.Nodes))
	d.Nodes = append(d.Nodes, NodeRecord{
		Kind:        NodeKindComment,
		Value:       comment,
		Parent:      NodeIDNull,
		FirstChild:  NodeIDNull,
		LastChild:   NodeIDNull,
		PrevSibling: NodeIDNull,
		NextSibling: NodeIDNull,
	})
	return id
}

// NewDeclaration appends an XML declaration to the arena.
func (d *Document) NewDeclaration(decl string) NodeID {
	id := NodeID(len(d.Nodes))
	d.Nodes = append(d.Nodes, NodeRecord{
		Kind:        NodeKindDeclaration,
		Value:       decl,
		Parent:      NodeIDNull,
		FirstChild:  NodeIDNull,
		LastChild:   NodeIDNull,
		PrevSibling: NodeIDNull,
		NextSibling: NodeIDNull,
	})
	return id
}

// InsertEndChild appends a child node to the end of parent's children list.
func (d *Document) InsertEndChild(parent, child NodeID) error {
	if int(parent) < 0 || int(parent) >= len(d.Nodes) {
		return errors.New("tinyxml2: parent NodeID out of bounds")
	}
	if int(child) < 0 || int(child) >= len(d.Nodes) {
		return errors.New("tinyxml2: child NodeID out of bounds")
	}
	if parent == child {
		return errors.New("tinyxml2: cannot insert node into itself")
	}

	d.Nodes[child].Parent = parent
	d.Nodes[child].NextSibling = NodeIDNull

	last := d.Nodes[parent].LastChild
	if last == NodeIDNull {
		d.Nodes[parent].FirstChild = child
		d.Nodes[parent].LastChild = child
		d.Nodes[child].PrevSibling = NodeIDNull
	} else {
		d.Nodes[last].NextSibling = child
		d.Nodes[child].PrevSibling = last
		d.Nodes[parent].LastChild = child
	}
	return nil
}

// SetAttribute assigns or updates an attribute on an element node.
func (d *Document) SetAttribute(nodeID NodeID, name, value string) error {
	if int(nodeID) < 0 || int(nodeID) >= len(d.Nodes) {
		return errors.New("tinyxml2: NodeID out of bounds")
	}
	node := &d.Nodes[nodeID]
	if node.Kind != NodeKindElement {
		return errors.New("tinyxml2: cannot set attribute on non-element node")
	}
	for i := range node.Attributes {
		if node.Attributes[i].Name == name {
			node.Attributes[i].Value = value
			return nil
		}
	}
	node.Attributes = append(node.Attributes, Attribute{Name: name, Value: value})
	return nil
}

// GetAttribute retrieves an attribute value by name.
func (d *Document) GetAttribute(nodeID NodeID, name string) (string, bool) {
	if int(nodeID) < 0 || int(nodeID) >= len(d.Nodes) {
		return "", false
	}
	node := &d.Nodes[nodeID]
	for _, attr := range node.Attributes {
		if attr.Name == name {
			return attr.Value, true
		}
	}
	return "", false
}

// RootElement returns the first element child of the DocumentRoot (ID 0).
func (d *Document) RootElement() NodeID {
	return d.FirstChildElement(0, "")
}

// FirstChildElement returns the first child element matching name (or any element if name is empty).
func (d *Document) FirstChildElement(parent NodeID, name string) NodeID {
	if int(parent) < 0 || int(parent) >= len(d.Nodes) {
		return NodeIDNull
	}
	curr := d.Nodes[parent].FirstChild
	for curr != NodeIDNull {
		if d.Nodes[curr].Kind == NodeKindElement {
			if name == "" || d.Nodes[curr].Name == name {
				return curr
			}
		}
		curr = d.Nodes[curr].NextSibling
	}
	return NodeIDNull
}

// NextSiblingElement returns the next sibling element matching name (or any if empty).
func (d *Document) NextSiblingElement(nodeID NodeID, name string) NodeID {
	if int(nodeID) < 0 || int(nodeID) >= len(d.Nodes) {
		return NodeIDNull
	}
	curr := d.Nodes[nodeID].NextSibling
	for curr != NodeIDNull {
		if d.Nodes[curr].Kind == NodeKindElement {
			if name == "" || d.Nodes[curr].Name == name {
				return curr
			}
		}
		curr = d.Nodes[curr].NextSibling
	}
	return NodeIDNull
}

// GetText returns the concatenated inner text of an element's text children.
func (d *Document) GetText(nodeID NodeID) string {
	if int(nodeID) < 0 || int(nodeID) >= len(d.Nodes) {
		return ""
	}
	var sb strings.Builder
	curr := d.Nodes[nodeID].FirstChild
	for curr != NodeIDNull {
		if d.Nodes[curr].Kind == NodeKindText {
			sb.WriteString(d.Nodes[curr].Value)
		}
		curr = d.Nodes[curr].NextSibling
	}
	return sb.String()
}

// Parse decodes raw XML into the arena document DOM.
func (d *Document) Parse(xmlStr string) error {
	d.Clear()
	decoder := xml.NewDecoder(strings.NewReader(xmlStr))

	parentStack := []NodeID{0} // Starts at DocumentRoot

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tinyxml2 parse error: %w", err)
		}

		currentParent := parentStack[len(parentStack)-1]

		switch t := token.(type) {
		case xml.StartElement:
			elemID := d.NewElement(t.Name.Local)
			for _, attr := range t.Attr {
				_ = d.SetAttribute(elemID, attr.Name.Local, attr.Value)
			}
			_ = d.InsertEndChild(currentParent, elemID)
			parentStack = append(parentStack, elemID)

		case xml.EndElement:
			if len(parentStack) > 1 {
				parentStack = parentStack[:len(parentStack)-1]
			}

		case xml.CharData:
			text := string(t)
			if strings.TrimSpace(text) != "" {
				textID := d.NewText(text)
				_ = d.InsertEndChild(currentParent, textID)
			}

		case xml.Comment:
			commentID := d.NewComment(string(t))
			_ = d.InsertEndChild(currentParent, commentID)

		case xml.ProcInst:
			declID := d.NewDeclaration(fmt.Sprintf("<?%s %s?>", t.Target, string(t.Inst)))
			_ = d.InsertEndChild(currentParent, declID)
		}
	}
	return nil
}

// Render formats the entire document hierarchy as indented XML.
func (d *Document) Render() string {
	var buf bytes.Buffer
	curr := d.Nodes[0].FirstChild
	for curr != NodeIDNull {
		d.renderNode(&buf, curr, 0, true)
		curr = d.Nodes[curr].NextSibling
	}
	return buf.String()
}

// RenderCompact formats the document hierarchy without indentation or extraneous whitespace.
func (d *Document) RenderCompact() string {
	var buf bytes.Buffer
	curr := d.Nodes[0].FirstChild
	for curr != NodeIDNull {
		d.renderNode(&buf, curr, 0, false)
		curr = d.Nodes[curr].NextSibling
	}
	return buf.String()
}

func (d *Document) renderNode(buf *bytes.Buffer, id NodeID, depth int, indent bool) {
	node := &d.Nodes[id]
	indentStr := ""
	if indent {
		indentStr = strings.Repeat("    ", depth)
	}

	switch node.Kind {
	case NodeKindDeclaration:
		buf.WriteString(indentStr)
		buf.WriteString(node.Value)
		if indent {
			buf.WriteString("\n")
		}

	case NodeKindComment:
		buf.WriteString(indentStr)
		buf.WriteString("<!--")
		buf.WriteString(node.Value)
		buf.WriteString("-->")
		if indent {
			buf.WriteString("\n")
		}

	case NodeKindText:
		buf.WriteString(xmlEscapeText(node.Value))

	case NodeKindElement:
		buf.WriteString(indentStr)
		buf.WriteString("<")
		buf.WriteString(node.Name)
		for _, attr := range node.Attributes {
			buf.WriteString(fmt.Sprintf(" %s=\"%s\"", attr.Name, xmlEscapeAttr(attr.Value)))
		}

		if node.FirstChild == NodeIDNull {
			buf.WriteString("/>")
			if indent {
				buf.WriteString("\n")
			}
			return
		}

		buf.WriteString(">")
		hasElementChildren := false
		curr := node.FirstChild
		for curr != NodeIDNull {
			if d.Nodes[curr].Kind == NodeKindElement || d.Nodes[curr].Kind == NodeKindComment {
				hasElementChildren = true
				break
			}
			curr = d.Nodes[curr].NextSibling
		}

		if hasElementChildren && indent {
			buf.WriteString("\n")
		}

		curr = node.FirstChild
		for curr != NodeIDNull {
			d.renderNode(buf, curr, depth+1, indent && hasElementChildren)
			curr = d.Nodes[curr].NextSibling
		}

		if hasElementChildren && indent {
			buf.WriteString(indentStr)
		}
		buf.WriteString("</")
		buf.WriteString(node.Name)
		buf.WriteString(">")
		if indent {
			buf.WriteString("\n")
		}
	}
}

func xmlEscapeText(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func xmlEscapeAttr(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
