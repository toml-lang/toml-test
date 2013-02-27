# A comprehensive test suite for TOML parsers

Example parser that satisfies interface:
[toml-test-go](https://github.com/BurntSushi/toml/tree/master/toml-test-go).

The bare bones are working. Polishing, documentation and tests forthcoming.

## Assumptions of Truth

The following are taken as ground truths by `toml-test`:

* All tests classified as `invalid` *are* invalid.
* All tests classified as `valid` *are* valid.

## Why JSON?

In order for a language agnostic test suite to work, we need some kind of data 
exchange format. TOML cannot be used, as it would imply that a particular 
parser has a blessing of correctness.

My decision to use JSON was not a careful one. It was based on expediency. The 
Go standard library has an excellent `encoding/json` package built in, which 
made it easy to compare JSON data.

The problem with JSON is that the types in TOML are not in one-to-one 
correspondence with JSON. This is why every TOML value represented in JSON is 
tagged with a type annotation, as described above.

YAML may be closer in correspondence with TOML, but I don't believe we should
rely on that correspondence. Making things explicit with JSON means that 
writing tests is a little more cumbersome, but it also reduces the number of 
assumptions we need to make.

