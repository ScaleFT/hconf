package hconf

import (
	"fmt"
	"github.com/vulcand/predicate"
)

type hcpredicate func(*HC) bool

type toInt func(c *HC) int
type toFloat64 func(c *HC) float64
type toString func(c *HC) string

// parseExpression parses expression in the go language into predicates.
func parseExpression(in string) (hcpredicate, error) {
	p, err := predicate.NewParser(predicate.Def{
		Operators: predicate.Operators{
			AND: and,
			OR:  or,
			EQ:  eq,
			NEQ: neq,
			LT:  lt,
			LE:  le,
			GT:  gt,
			GE:  ge,
		},
		Functions: map[string]interface{}{
			"local_Exec": localExec,
		},
	})
	if err != nil {
		return nil, err
	}
	out, err := p.Parse(in)
	if err != nil {
		return nil, err
	}
	pr, ok := out.(hcpredicate)
	if !ok {
		return nil, fmt.Errorf("expected predicate, got %T: %#v", out, out)
	}
	return pr, nil
}

// or returns predicate by joining the passed predicates with logical 'or'
func or(fns ...hcpredicate) hcpredicate {
	return func(c *HC) bool {
		for _, fn := range fns {
			if fn(c) {
				return true
			}
		}
		return false
	}
}

// and returns predicate by joining the passed predicates with logical 'and'
func and(fns ...hcpredicate) hcpredicate {
	return func(c *HC) bool {
		for _, fn := range fns {
			if !fn(c) {
				return false
			}
		}
		return true
	}
}

// not creates negation of the passed predicate
func not(p hcpredicate) hcpredicate {
	return func(c *HC) bool {
		return !p(c)
	}
}

// eq returns predicate that tests for equality of the value of the mapper and the constant
func eq(m interface{}, value interface{}) (hcpredicate, error) {
	switch mapper := m.(type) {
	case toInt:
		return intEQ(mapper, value)
	case toFloat64:
		return float64EQ(mapper, value)
	case toString:
		return stringEQ(mapper, value)
	}
	return nil, fmt.Errorf("eq: unsupported argument: %T", m)
}

// neq returns predicate that tests for inequality of the value of the mapper and the constant
func neq(m interface{}, value interface{}) (hcpredicate, error) {
	p, err := eq(m, value)
	if err != nil {
		return nil, err
	}
	return not(p), nil
}

// lt returns predicate that tests that value of the mapper function is less than the constant
func lt(m interface{}, value interface{}) (hcpredicate, error) {
	switch mapper := m.(type) {
	case toInt:
		return intLT(mapper, value)
	case toFloat64:
		return float64LT(mapper, value)
	}
	return nil, fmt.Errorf("lt: unsupported argument: %T", m)
}

// le returns predicate that tests that value of the mapper function is less or equal than the constant
func le(m interface{}, value interface{}) (hcpredicate, error) {
	l, err := lt(m, value)
	if err != nil {
		return nil, err
	}
	e, err := eq(m, value)
	if err != nil {
		return nil, err
	}
	return func(c *HC) bool {
		return l(c) || e(c)
	}, nil
}

// gt returns predicate that tests that value of the mapper function is greater than the constant
func gt(m interface{}, value interface{}) (hcpredicate, error) {
	switch mapper := m.(type) {
	case toInt:
		return intGT(mapper, value)
	case toFloat64:
		return float64GT(mapper, value)
	}
	return nil, fmt.Errorf("gt: unsupported argument: %T", m)
}

// ge returns predicate that tests that value of the mapper function is less or equal than the constant
func ge(m interface{}, value interface{}) (hcpredicate, error) {
	g, err := gt(m, value)
	if err != nil {
		return nil, err
	}
	e, err := eq(m, value)
	if err != nil {
		return nil, err
	}
	return func(c *HC) bool {
		return g(c) || e(c)
	}, nil
}

func intEQ(m toInt, val interface{}) (hcpredicate, error) {
	value, ok := val.(int)
	if !ok {
		return nil, fmt.Errorf("expected int, got %T", val)
	}
	return func(c *HC) bool {
		return m(c) == value
	}, nil
}

func float64EQ(m toFloat64, val interface{}) (hcpredicate, error) {
	value, ok := val.(float64)
	if !ok {
		return nil, fmt.Errorf("expected float64, got %T", val)
	}
	return func(c *HC) bool {
		return m(c) == value
	}, nil
}

func stringEQ(m toString, val interface{}) (hcpredicate, error) {
	value, ok := val.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", val)
	}
	return func(c *HC) bool {
		return m(c) == value
	}, nil
}

func intLT(m toInt, val interface{}) (hcpredicate, error) {
	value, ok := val.(int)
	if !ok {
		return nil, fmt.Errorf("expected int, got %T", val)
	}
	return func(c *HC) bool {
		return m(c) < value
	}, nil
}

func intGT(m toInt, val interface{}) (hcpredicate, error) {
	value, ok := val.(int)
	if !ok {
		return nil, fmt.Errorf("expected int, got %T", val)
	}
	return func(c *HC) bool {
		return m(c) > value
	}, nil
}

func float64LT(m toFloat64, val interface{}) (hcpredicate, error) {
	value, ok := val.(float64)
	if !ok {
		return nil, fmt.Errorf("expected int, got %T", val)
	}
	return func(c *HC) bool {
		return m(c) < value
	}, nil
}

func float64GT(m toFloat64, val interface{}) (hcpredicate, error) {
	value, ok := val.(float64)
	if !ok {
		return nil, fmt.Errorf("expected int, got %T", val)
	}
	return func(c *HC) bool {
		return m(c) > value
	}, nil
}

func localExec(x string) toString {
	return func(c *HC) string {
		return x
	}
}
