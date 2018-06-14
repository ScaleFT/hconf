package hconf

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type foo struct {
	Screensize String      `hconf:"screensize"`
	LikesCats  Bool        `hconf:"likes_cats"`
	LikesDogs  Bool        `hconf:"likes_dogs"`
	Friends    StringSlice `hconf:"friends"`
}

type myConf struct {
	Version string `hconf:"version"`
	Foo     foo    `hsection:"foo"`
	Bar     foo    `hsection:"bar"`
}

const conf = `
section "foo" {
    screensize = "hello world"
	likes_cats = true
	likes_dogs = false
	friends = ["alice", "bob"]
}
`

const conf2 = `
section "foo" {
	likes_cats = true
}
`

func TestBasicSection(t *testing.T) {
	hc, err := New(nil)
	require.NoError(t, err)
	require.NotNil(t, hc)

	out := &myConf{}
	require.Equal(t, "", out.Foo.Screensize.Value())
	require.False(t, out.Foo.LikesCats.IsSet())

	err = hc.Decode(out, "foo.conf", []byte(conf))

	require.NoError(t, err)
	require.Equal(t, "hello world", out.Foo.Screensize.Value())

	require.True(t, out.Foo.LikesCats.Value())
	require.True(t, out.Foo.LikesCats.IsSet())
	require.False(t, out.Foo.LikesDogs.Value())
	require.True(t, out.Foo.LikesDogs.IsSet())

	require.Len(t, out.Foo.Friends.Value(), 2)
	require.Equal(t, "alice", out.Foo.Friends.Value()[0])
	require.Equal(t, "bob", out.Foo.Friends.Value()[1])

	h, err := parseExpression(`local_Exec("test") == "test"`)
	require.NoError(t, err)
	require.Equal(t, h(hc), true)
}

func TestIsSet(t *testing.T) {
	hc, err := New(nil)
	require.NoError(t, err)
	require.NotNil(t, hc)

	out := &myConf{}
	require.False(t, out.Foo.LikesCats.IsSet())

	err = hc.Decode(out, "foo.conf", []byte(conf2))
	require.NoError(t, err)
	require.True(t, out.Foo.LikesCats.IsSet())
	require.False(t, out.Foo.LikesDogs.IsSet())
}

func TestEditNoSuchFile(t *testing.T) {
	d, err := ioutil.TempDir("", "hconf")
	require.NoError(t, err)
	defer os.RemoveAll(d)

	hc, err := New(nil)
	require.NoError(t, err)
	require.NotNil(t, hc)

	tpath := filepath.Join(d, "t.conf")

	err = hc.EditAndSave(tpath, "foo", "screensize", "giant")
	require.NoError(t, err)
}

func TestEditInplace(t *testing.T) {
	d, err := ioutil.TempDir("", "hconf")
	require.NoError(t, err)
	defer os.RemoveAll(d)

	hc, err := New(nil)
	require.NoError(t, err)
	require.NotNil(t, hc)

	tpath := filepath.Join(d, "t.conf")

	err = ioutil.WriteFile(tpath, []byte(conf), 0600)
	require.NoError(t, err)

	err = hc.EditAndSave(tpath, "foo", "screensize", "giant")
	require.NoError(t, err)

	err = hc.EditAndSave(tpath, "foo", "friends", []string{"marco", "polo", "charlie"})
	require.NoError(t, err)

	data, err := ioutil.ReadFile(tpath)
	require.NoError(t, err)
	c := &myConf{}
	err = hc.Decode(c, "t.conf", []byte(data))
	require.NoError(t, err)
	require.Equal(t, "giant", c.Foo.Screensize.Value())
	require.Equal(t, "", c.Bar.Screensize.Value())

	require.Len(t, c.Foo.Friends.Value(), 3)
	require.Equal(t, "marco", c.Foo.Friends.Value()[0])
	require.Equal(t, "polo", c.Foo.Friends.Value()[1])
	require.Equal(t, "charlie", c.Foo.Friends.Value()[2])
}

func TestEditNewSection(t *testing.T) {
	d, err := ioutil.TempDir("", "hconf")
	require.NoError(t, err)
	defer os.RemoveAll(d)

	hc, err := New(nil)
	require.NoError(t, err)
	require.NotNil(t, hc)

	tpath := filepath.Join(d, "t.conf")

	err = ioutil.WriteFile(tpath, []byte(conf), 0600)
	require.NoError(t, err)

	err = hc.EditAndSave(tpath, "bar", "screensize", "giant")
	require.NoError(t, err)

	data, err := ioutil.ReadFile(tpath)
	require.NoError(t, err)

	c := &myConf{}
	err = hc.Decode(c, "t.conf", []byte(data))
	require.NoError(t, err)
	require.Equal(t, "hello world", c.Foo.Screensize.Value())
	require.Equal(t, "giant", c.Bar.Screensize.Value())

	err = hc.Set(c, "foo", "screensize", "sunset")
	require.NoError(t, err)
	require.Equal(t, "sunset", c.Foo.Screensize.Value())

	err = hc.Set(c, "foo", "friends", []string{"bob"})
	require.NoError(t, err)
	require.Len(t, c.Foo.Friends.Value(), 1)
	require.Equal(t, "bob", c.Foo.Friends.Value()[0])
}
