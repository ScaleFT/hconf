package hconf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"strconv"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/hcl/hcl/parser"
	"github.com/hashicorp/hcl/hcl/token"
)

const tagSection = "hsection"
const tagValue = "hconf"

type HC struct {
	c *Config
}

type Config struct {
}

func New(c *Config) (*HC, error) {
	return &HC{c: c}, nil
}

// Decode HC into an object
func (hc *HC) DecodeFile(out interface{}, filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return hc.Decode(out, filename, data)
}

func (hc *HC) sectionFields(out reflect.Value) (map[string]reflect.Value, error) {
	structType := out.Type()
	fields := make(map[*reflect.StructField]reflect.Value)
	for i := 0; i < structType.NumField(); i++ {
		fieldType := structType.Field(i)
		fields[&fieldType] = out.Field(i)
	}

	valueFields := make(map[string]reflect.Value)

	for fieldType, field := range fields {
		if !field.CanSet() {
			continue
		}

		tag := fieldType.Tag.Get(tagValue)
		if tag != "" {
			valueFields[tag] = field
		}
	}

	return valueFields, nil
}

// node invariants:
//  node.Keys[0] == "section"
//  node.Keys[1] == sectionName
func (hc *HC) handleSection(out reflect.Value, node *ast.ObjectItem) error {
	sectionName, err := getKeyAsString(node.Keys[1])
	if err != nil {
		return err
	}

	valueFields, err := hc.sectionFields(out)
	if err != nil {
		return err
	}

	obj := node.Val.(*ast.ObjectType)
	for _, item := range obj.List.Items {
		if len(item.Keys) != 1 {
			return &parser.PosError{
				Pos: item.Pos(),
				Err: fmt.Errorf("expected flat keys under section %s", sectionName),
			}
		}

		key, err := getKeyAsString(item.Keys[0])
		if err != nil {
			return err
		}

		v, ok := valueFields[key]
		if !ok {
			return &parser.PosError{
				Pos: item.Keys[0].Pos(),
				Err: fmt.Errorf("unknown key: %s.%s", sectionName, key),
			}
		}

		err = hc.decodeInto(key, item.Val, v)
		if err != nil {
			return err
		}
		/*
			println("------------")
			fmt.Printf("section.item: %s.%s\n", sectionName, key)
			println("------------")
		*/
	}

	return nil
}

// node invariants:
//  node.Keys[0] == "when"
//  node.Keys[1] == whenConditional
func (hc *HC) handleWhen(out interface{}, node *ast.ObjectItem) error {
	// fmt.Printf("when: %#v\n", node)
	return nil
}

func getKeyAsString(objkey *ast.ObjectKey) (string, error) {
	key := ""
	keyToken := objkey.Token
	switch keyToken.Type {
	case token.NUMBER:
		key = keyToken.Text
	case token.IDENT:
		key = keyToken.Text
	case token.STRING, token.HEREDOC:
		key = keyToken.Value().(string)
	default:
		return "", &parser.PosError{
			Pos: objkey.Pos(),
			Err: fmt.Errorf("expected string type: %s", keyToken.Type),
		}
	}
	return key, nil
}

func (hc *HC) Decode(out interface{}, filename string, data []byte) error {
	err := hc.decode(out, filename, data)
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

func (hc *HC) fields(obj reflect.Value) (map[string]reflect.Value, map[string]reflect.Value, error) {
	result := obj.Elem()
	structType := result.Type()
	fields := make(map[*reflect.StructField]reflect.Value)
	for i := 0; i < structType.NumField(); i++ {
		fieldType := structType.Field(i)
		fields[&fieldType] = result.Field(i)
	}

	sectionFields := make(map[string]reflect.Value)
	valueFields := make(map[string]reflect.Value)

	for fieldType, field := range fields {
		if !field.CanSet() {
			continue
		}

		tag := fieldType.Tag.Get(tagSection)
		if tag != "" {
			sectionFields[tag] = field
		} else {
			tag = fieldType.Tag.Get(tagValue)
			if tag != "" {
				valueFields[tag] = field
			}
		}
	}

	return sectionFields, valueFields, nil
}

func (hc *HC) decode(out interface{}, filename string, data []byte) error {
	tree, err := hcl.ParseBytes(data)
	if err != nil {
		return err
	}

	val := reflect.ValueOf(out)
	if val.Kind() != reflect.Ptr {
		return errors.New("out must be a pointer")
	}

	sectionFields, valueFields, err := hc.fields(val)
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

	for _, item := range root.Items {
		if len(item.Keys) == 1 {
			// top level key
			key := item.Keys[0].Token.Text

			v, ok := valueFields[key]
			if !ok {
				return &parser.PosError{
					Pos: item.Keys[0].Pos(),
					Err: fmt.Errorf("unknown key: %s", key),
				}
			}

			err = hc.decodeInto(key, item.Val, v)
			if err != nil {
				return err
			}
		} else if len(item.Keys) == 2 {
			typeOfSection := item.Keys[0].Token.Text
			switch typeOfSection {
			case "section":
				key, err := getKeyAsString(item.Keys[1])
				if err != nil {
					return err
				}

				sectionValue, ok := sectionFields[key]
				if !ok {
					return &parser.PosError{
						Pos: item.Keys[1].Pos(),
						Err: fmt.Errorf("unknown section: %s", key),
					}
				}
				err = hc.handleSection(sectionValue, item)
				if err != nil {
					return err
				}
			case "when":
				err = hc.handleWhen(out, item)
				if err != nil {
					return err
				}
			default:
				return &parser.PosError{
					Pos: item.Pos(),
					Err: fmt.Errorf("unkown section type '%s' expected 'section' or 'when'", typeOfSection),
				}
			}
		} else {
			return &parser.PosError{
				Pos: item.Pos(),
				Err: fmt.Errorf("invalid config: expected: section, when, or top level key: %#v", item),
			}
		}
	}

	return nil
}

// Set a specific value from a section/key pair
func (hc *HC) Set(input interface{}, section string, key string, value interface{}) error {
	val := reflect.ValueOf(input)
	if val.Kind() != reflect.Ptr {
		return errors.New("out must be a pointer")
	}

	sectionFields, _, err := hc.fields(val)
	if err != nil {
		return err
	}

	sectionValue, ok := sectionFields[section]
	if !ok {
		return fmt.Errorf("unknown section: %s", section)
	}

	valueFields, err := hc.sectionFields(sectionValue)
	if err != nil {
		return err
	}

	v, ok := valueFields[key]
	if !ok {
		return fmt.Errorf("unknown key: %s in section %s", key, section)
	}

	vif := v.Addr().Interface()

	switch v := value.(type) {
	case string:
		if ss, ok := vif.(stringSetter); ok {
			ss.SetValue(v)
			return nil
		}

		if bs, ok := vif.(boolSetter); ok {
			if v == "true" {
				bs.SetValue(true)
				return nil
			} else if v == "false" {
				bs.SetValue(false)
				return nil
			}
			return fmt.Errorf("key: %s.%s failed to set bool from string: '%s'", key, section, v)
		}

		if bs, ok := vif.(stringSliceSetter); ok {
			ss := value.(string)
			x := []string{}
			err := json.Unmarshal([]byte(ss), &x)
			if err != nil {
				return fmt.Errorf("'%s.%s' must be a list of strings", section, key)
			}
			bs.SetValue(x)
			return nil
		}

		return fmt.Errorf("key: %s.%s failed to set from string", key, section)
	case int32:
		if is, ok := vif.(int64Setter); ok {
			is.SetValue(int64(v))
			return nil
		}
	case int64:
		if is, ok := vif.(int64Setter); ok {
			is.SetValue(v)
			return nil
		}
	case bool:
		if bs, ok := vif.(boolSetter); ok {
			bs.SetValue(v)
			return nil
		}
	case []string:
		if bs, ok := vif.(stringSliceSetter); ok {
			bs.SetValue(v)
			return nil
		}
	}

	return fmt.Errorf("unknown key: %s.%s is %T, not known type, failed to set to from '%v'", key, section, vif, vif)
}

// Get a specific value from a section/key pair
func (hc *HC) Get(input interface{}, section string, key string) (interface{}, token.Pos, error) {
	pos := token.Pos{}
	val := reflect.ValueOf(input)
	if val.Kind() != reflect.Ptr {
		return nil, pos, errors.New("out must be a pointer")
	}

	sectionFields, _, err := hc.fields(val)
	if err != nil {
		return nil, pos, err
	}

	sectionValue, ok := sectionFields[section]
	if !ok {
		return nil, pos, fmt.Errorf("unknown section: %s", section)
	}

	valueFields, err := hc.sectionFields(sectionValue)
	if err != nil {
		return nil, pos, err
	}

	v, ok := valueFields[key]
	if !ok {
		return nil, pos, fmt.Errorf("unknown key: %s in section %s", key, section)
	}

	vif := v.Addr().Interface()
	if sg, ok := vif.(sourceGetter); ok {
		return vif, sg.Source(), nil
	}
	return vif, pos, nil
}

func (hc *HC) decodeInto(name string, node ast.Node, result reflect.Value) error {
	var err error
	switch result.Kind() {
	case reflect.Bool:
		err = hc.decodeBool(name, node, result)
	case reflect.Float64:
		err = hc.decodeFloat(name, node, result)
	case reflect.Int:
		err = hc.decodeInt(name, node, result)
	case reflect.Ptr:
		err = hc.decodePtr(name, node, result)
	case reflect.String:
		err = hc.decodeString(name, node, result)
	case reflect.Struct:
		if ss, ok := result.Addr().Interface().(stringSetter); ok {
			var v string
			rv := reflect.Indirect(reflect.ValueOf(&v))
			err = hc.decodeString(name, node, rv)
			if err != nil {
				return err
			}
			ss.SetValue(v)
		} else if is, ok := result.Addr().Interface().(int64Setter); ok {
			var i int64
			rv := reflect.Indirect(reflect.ValueOf(&i))
			err = hc.decodeInt(name, node, rv)
			if err != nil {
				return err
			}
			is.SetValue(i)
		} else if bs, ok := result.Addr().Interface().(boolSetter); ok {
			var b bool
			rv := reflect.Indirect(reflect.ValueOf(&b))
			err = hc.decodeBool(name, node, rv)
			if err != nil {
				return err
			}
			bs.SetValue(b)
		} else if bs, ok := result.Addr().Interface().(stringSliceSetter); ok {
			var out []string
			rv := reflect.Indirect(reflect.ValueOf(&out))
			err = hc.decodeStringSlice(name, node, rv)
			if err != nil {
				return err
			}
			bs.SetValue(out)
		}

		if ss, ok := result.Addr().Interface().(sourceSetter); ok {
			ss.SetSource(node.Pos())
		}
	default:
		return &parser.PosError{
			Pos: node.Pos(),
			Err: fmt.Errorf("%s: unknown kind to decode into: %s", name, result.Kind()),
		}
	}

	if err != nil {
		return err
	}

	return nil
}

func (hc *HC) decodeBool(name string, node ast.Node, result reflect.Value) error {
	switch n := node.(type) {
	case *ast.LiteralType:
		if n.Token.Type == token.BOOL || n.Token.Type == token.STRING {
			v, err := strconv.ParseBool(n.Token.Text)
			if err != nil {
				return err
			}

			result.Set(reflect.ValueOf(v))
			return nil
		}
		if n.Token.Type == token.STRING {
			v, err := strconv.ParseBool(n.Token.Value().(string))
			if err != nil {
				return err
			}

			result.Set(reflect.ValueOf(v))
			return nil
		}
	}

	return &parser.PosError{
		Pos: node.Pos(),
		Err: fmt.Errorf("%s: unknown type %T", name, node),
	}
}

func (hc *HC) decodeFloat(name string, node ast.Node, result reflect.Value) error {
	switch n := node.(type) {
	case *ast.LiteralType:
		if n.Token.Type == token.FLOAT {
			v, err := strconv.ParseFloat(n.Token.Text, 64)
			if err != nil {
				return err
			}

			result.Set(reflect.ValueOf(v))
			return nil
		}
	}

	return &parser.PosError{
		Pos: node.Pos(),
		Err: fmt.Errorf("%s: unknown type %T", name, node),
	}
}

func (hc *HC) decodeInt(name string, node ast.Node, result reflect.Value) error {
	switch n := node.(type) {
	case *ast.LiteralType:
		switch n.Token.Type {
		case token.NUMBER:
			v, err := strconv.ParseInt(n.Token.Text, 0, 0)
			if err != nil {
				return err
			}

			result.Set(reflect.ValueOf(int64(v)))
			return nil
		case token.STRING:
			v, err := strconv.ParseInt(n.Token.Value().(string), 0, 0)
			if err != nil {
				return err
			}

			result.Set(reflect.ValueOf(int64(v)))
			return nil
		}
	}

	return &parser.PosError{
		Pos: node.Pos(),
		Err: fmt.Errorf("%s: unknown type %T", name, node),
	}
}

func (hc *HC) decodeString(name string, node ast.Node, result reflect.Value) error {
	switch n := node.(type) {
	case *ast.LiteralType:
		switch n.Token.Type {
		case token.NUMBER:
			result.Set(reflect.ValueOf(n.Token.Text).Convert(result.Type()))
			return nil
		case token.STRING, token.HEREDOC:
			result.Set(reflect.ValueOf(n.Token.Value()).Convert(result.Type()))
			return nil
		}
	}

	return &parser.PosError{
		Pos: node.Pos(),
		Err: fmt.Errorf("%s: unknown type for string %T", name, node),
	}
}

func (hc *HC) decodeStringSlice(name string, node ast.Node, result reflect.Value) error {
	switch n := node.(type) {
	case *ast.ListType:
		rv := make([]string, 0, len(n.List))
		for i, ent := range n.List {
			switch lit := ent.(type) {
			case *ast.LiteralType:
				switch lit.Token.Type {
				case token.STRING, token.HEREDOC:
					rv = append(rv, lit.Token.Value().(string))
				default:
					return &parser.PosError{
						Pos: node.Pos(),
						Err: fmt.Errorf("%s[%d]: unknown entry type for string slice %T", name, i, lit),
					}
				}
			default:
				return &parser.PosError{
					Pos: node.Pos(),
					Err: fmt.Errorf("%s[%d]: unknown entry for string slice %T", name, i, ent),
				}
			}
		}
		result.Set(reflect.ValueOf(rv).Convert(result.Type()))
		return nil
	}

	return &parser.PosError{
		Pos: node.Pos(),
		Err: fmt.Errorf("%s: unknown type for string slice %T", name, node),
	}
}

func (hc *HC) decodePtr(name string, node ast.Node, result reflect.Value) error {
	// Create an element of the concrete (non pointer) type and decode
	// into that. Then set the value of the pointer to this type.
	resultType := result.Type()
	resultElemType := resultType.Elem()
	val := reflect.New(resultElemType)
	if err := hc.decodeInto(name, node, reflect.Indirect(val)); err != nil {
		return err
	}

	result.Set(val)
	return nil
}
