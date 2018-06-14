package hconf

import (
	"github.com/hashicorp/hcl/hcl/token"
)

type stringSetter interface {
	SetValue(v string)
}

type stringSliceSetter interface {
	SetValue(v []string)
}

type int64Setter interface {
	SetValue(v int64)
}

type boolSetter interface {
	SetValue(v bool)
}

type sourceSetter interface {
	SetSource(p token.Pos)
}

type sourceGetter interface {
	Source() token.Pos
}

type String struct {
	source token.Pos
	value  string
	isset  bool
}

func (s *String) Duplicate() String {
	return String{
		source: s.source,
		value:  s.value,
		isset:  s.isset,
	}
}

func (s *String) SetSource(p token.Pos) {
	s.source = p
}

func (s *String) Source() token.Pos {
	return s.source
}

func (s *String) SetValue(v string) {
	s.value = v
	s.isset = true
}

func (s *String) Value() string {
	return s.value
}

func (s *String) ValueString() string {
	return s.value
}

func (s *String) IsSet() bool {
	return s.isset
}

type StringSlice struct {
	source token.Pos
	value  []string
	isset  bool
}

func (s *StringSlice) Duplicate() StringSlice {
	b := make([]string, 0, len(s.value))
	b = append(b, s.value...)

	return StringSlice{
		source: s.source,
		value:  b,
		isset:  s.isset,
	}
}

func (s *StringSlice) SetSource(p token.Pos) {
	s.source = p
}

func (s *StringSlice) Source() token.Pos {
	return s.source
}

func (s *StringSlice) SetValue(v []string) {
	s.value = v
	s.isset = true
}

func (s *StringSlice) Value() []string {
	return s.value
}

func (s *StringSlice) ValueStringSlice() []string {
	return s.value
}

func (s *StringSlice) IsSet() bool {
	return s.isset
}

type Int64 struct {
	source token.Pos
	value  int64
	isset  bool
}

func (s *Int64) SetSource(p token.Pos) {
	s.source = p
}

func (s *Int64) Source() token.Pos {
	return s.source
}

func (s *Int64) SetValue(v int64) {
	s.value = v
	s.isset = true
}

func (s *Int64) Value() int64 {
	return s.value
}

func (s *Int64) ValueInt64() int64 {
	return s.value
}

func (s *Int64) IsSet() bool {
	return s.isset
}

type Bool struct {
	source token.Pos
	value  bool
	isset  bool
}

func (s *Bool) Duplicate() Bool {
	return Bool{
		source: s.source,
		value:  s.value,
		isset:  s.isset,
	}
}

func (s *Bool) SetSource(p token.Pos) {
	s.source = p
}

func (s *Bool) Source() token.Pos {
	return s.source
}

func (s *Bool) SetValue(v bool) {
	s.value = v
	s.isset = true
}

func (s *Bool) Value() bool {
	return s.value
}

func (s *Bool) ValueBool() bool {
	return s.value
}

func (s *Bool) IsSet() bool {
	return s.isset
}
