package hconf

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/hcl/hcl/parser"
	"github.com/hashicorp/hcl/hcl/printer"
	"github.com/hashicorp/hcl/hcl/token"
)

// EditAndSave open's existing file, edits section/key value, saves back as a formatted HCL. (kitchen sink method)
func (hc *HC) EditAndSave(filename string, section string, key string, value interface{}) error {
	err := hc.editAndSave(filename, section, key, value)
	if err != nil {
		switch xerr := err.(type) {
		case *parser.PosError:
			if xerr.Pos.Filename == "" {
				xerr.Pos.Filename = filename
			}
		}
		return err
	}
	return nil
}

func (hc *HC) editAndSave(filename string, section string, key string, value interface{}) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	tree, err := hcl.ParseBytes(data)
	if err != nil {
		return err
	}

	root, ok := tree.Node.(*ast.ObjectList)
	if !ok {
		return &parser.PosError{
			Pos: tree.Pos(),
			Err: fmt.Errorf("invalid config: missing root objects: %#v", tree.Node),
		}
	}

	sectionFound := false

	setObjKey := &ast.ObjectKey{
		Token: token.Token{
			Type: token.IDENT,
			Text: key,
		},
	}

	var setNode ast.Node
	switch v := value.(type) {
	case string:
		setNode = &ast.LiteralType{
			Token: token.Token{
				Type: token.STRING,
				Text: strconv.Quote(v),
			},
		}
	case int32:
		setNode = &ast.LiteralType{
			Token: token.Token{
				Type: token.NUMBER,
				Text: fmt.Sprintf("%d", v),
			},
		}
	case int64:
		setNode = &ast.LiteralType{
			Token: token.Token{
				Type: token.NUMBER,
				Text: fmt.Sprintf("%d", v),
			},
		}
	case bool:
		setNode = &ast.LiteralType{
			Token: token.Token{
				Type: token.BOOL,
				Text: fmt.Sprintf("%t", v),
			},
		}
	case []string:
		lt := &ast.ListType{
			List: make([]ast.Node, 0, len(v)),
		}
		for _, ent := range v {
			entNode := &ast.LiteralType{
				Token: token.Token{
					Type: token.STRING,
					Text: strconv.Quote(ent),
				},
			}
			lt.List = append(lt.List, entNode)
		}
		setNode = lt
	default:
		return &parser.PosError{
			Pos: tree.Pos(),
			Err: fmt.Errorf("invalid set: unknown type %T trying to set %s.%s = %#v", v, key, section, v),
		}
	}

	setObjItem := &ast.ObjectItem{
		Keys: []*ast.ObjectKey{
			setObjKey,
		},
		Assign: token.Pos{Line: 1},
		Val:    setNode,
	}

	setSection := &ast.ObjectItem{
		Keys: []*ast.ObjectKey{
			&ast.ObjectKey{
				Token: token.Token{
					Type: token.IDENT,
					Text: "section",
				},
			},
			&ast.ObjectKey{
				Token: token.Token{
					Type: token.STRING,
					Text: strconv.Quote(section),
				},
			},
		},
		Val: &ast.ObjectType{
			List: &ast.ObjectList{
				Items: []*ast.ObjectItem{
					setObjItem,
				},
			},
		},
	}

	for _, item := range root.Items {
		if len(item.Keys) == 2 {
			typeOfSection := item.Keys[0].Token.Text
			switch typeOfSection {
			case "section":
				sectionName, err := getKeyAsString(item.Keys[1])
				if err != nil {
					return err
				}

				if sectionName != section {
					continue
				}

				sectionFound = true

				keyFound := false
				obj := item.Val.(*ast.ObjectType)
				for _, item := range obj.List.Items {
					if len(item.Keys) != 1 {
						return &parser.PosError{
							Pos: item.Pos(),
							Err: fmt.Errorf("expected flat keys under section %s", sectionName),
						}
					}

					keyName, err := getKeyAsString(item.Keys[0])
					if err != nil {
						return err
					}

					if keyName == key {
						keyFound = true
						item.Val = setNode
						break
					}
				}

				if !keyFound {
					obj.List.Add(setObjItem)
				}
			}
		}
	}

	if !sectionFound {
		// append new section
		root.Add(setSection)
	}

	buf := &bytes.Buffer{}

	err = printer.Fprint(buf, tree)
	if err != nil {
		return err
	}

	dirname := filepath.Dir(filename)
	if _, err := os.Stat(dirname); os.IsNotExist(err) {
		os.MkdirAll(dirname, 0755)
	}

	return ioutil.WriteFile(filename, buf.Bytes(), 0600)
}
