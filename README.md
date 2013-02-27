# A comprehensive language agnostic test suite for TOML parsers

`toml-test` is a higher-order program that tests other 
[TOML](https://github.com/mojombo/toml)
parsers. Tests are divided into two groups: invalid TOML data and valid TOML 
data. Parsers that reject invalid TOML data pass invalid TOML tests. Parsers 
that accept valid TOML data and output precisely what is expected pass valid 
tests. The output format is JSON, described below.

Compatible with toml commit
[3f4224ecdc](https://github.com/mojombo/toml/commit/3f4224ecdc4a65fdd28b4fb70d46f4c0bd3700aa).

## Assumptions of Truth

The following are taken as ground truths by `toml-test`:

* All tests classified as `invalid` **are** invalid.
* All tests classified as `valid` **are** valid.
* All expected outputs in `valid/*.json` are exactly correct.
* The Go standard library package `encoding/json` decodes JSON correctly.

Of particular note is that **no TOML parser** is taken as ground truth. This
means that most changes to the spec will only require an update of the tests
in `toml-test`. (Bigger changes may require an adjustment of how two things
are considered equal. Particularly if a new type of data is added.)

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

## Parsers that satisfy the `toml-test` interface

If you have an implementation, send a pull request adding to this list. Please 
note the commit SHA1 or version tag that your parser supports in your `README`.

* Go (@BurntSushi) - https://github.com/BurntSushi/toml/tree/master/toml-test-go


