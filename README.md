# hconf: HCL library for configuration files

[![GoDoc](https://godoc.org/github.com/ScaleFT/hconf?status.svg)](https://godoc.org/github.com/ScaleFT/hconf)
[![Build Status](https://travis-ci.org/ScaleFT/hconf.svg?branch=master)](https://travis-ci.org/ScaleFT/hconf)

`hconf` extends the [Hashicorp HCL library](https://github.com/hashicorp/hcl) and is intended to be used as configuration file format.

## Configuration file API

`hconf` is based around a configuration file with sections, and each section can have key/value pairs.

Given a configuration file like this:

```
section "autoupdate" {
  release_channel = "test"
}
```

And a Golang structure like this:

```
type Config struct {
	Autoupdate      Autoupdate      `hsection:"autoupdate"`
}

type Autoupdate struct {
	ReleaseChannel hconf.String `hconf:"release_channel"`
}
```

This could be parsed like this:

```
hc := hconf.New(&hconf.Config{})
config := &Config{}

err := hc.DecodeFile(config, "path/to/config.conf")

// config.Autoupdate.ReleaseChannel now contains "test"
```

## Future Ideas

- Adding conditional `when` support based on predicates or local command execution to allow a more flexiable configuration file. See `predicate.go`.

# License

`hconf` is licensed under the Mozilla Public License version 2.0. See the [LICENSE file](./LICENSE) for details.
