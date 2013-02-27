# A comprehensive language agnostic test suite for TOML parsers

`toml-test` is a higher-order program that tests other 
[TOML](https://github.com/mojombo/toml)
parsers. Tests are divided into two groups: invalid TOML data and valid TOML 
data. Parsers that reject invalid TOML data pass invalid TOML tests. Parsers 
that accept valid TOML data and output precisely what is expected pass valid 
tests. The output format is JSON, described below.

Compatible with TOML commit
[3f4224ecdc](https://github.com/mojombo/toml/commit/3f4224ecdc4a65fdd28b4fb70d46f4c0bd3700aa).

## Interface of a parser

For your parser to be compatible with `toml-test`, it **must** satisfy the 
interface expected.

Your parser **must** accept TOML data on `stdin` until EOF.

If the TOML data is invalid, your parser **must** return with a non-zero
exit code indicating an error.

If the TOML data is valid, Your parser **must** output a JSON encoding of that 
data on `stdout` and return with a zero exit code indicating success.

The rest of this section is dedicated to describing that JSON encoding.

### JSON encoding

* TOML hashes correspond to JSON objects.
* TOML values correspond to a special JSON object of the form
  `{"type": "{TTYPE}", "value": {TVALUE}}`

In the above, `TTYPE` may be one of:

* string
* integer
* float
* datetime
* bool
* array

and `TVALUE` is always a JSON string, except when `TTYPE` is `array` in which
`TVALUE` is a JSON array containing TOML values.

Empty hashes map to empty JSON objects (i.e., `{}`) and empty arrays map to
empty JSON arrays (i.e., `[]`).

### Example JSON encoding

Here is the TOML data:

```toml
best-day-ever = 1987-07-05T17:45:00Z

[numtheory]
boring = false
perfection = [6, 28, 496]
```

And the JSON encoding expected by `toml-test` is:

```json
{
  "best-day-ever": {"type": "datetime", "value": "1987-07-05T17:45:00Z"},
  "numtheory": {
    "boring": {"type": "bool", "value": "false"},
    "perfection": {
      "type": "array",
      "value": [
        {"type": "integer", "value": "6"},
        {"type": "integer", "value": "28"},
        {"type": "integer", "value": "496"}
      ]
    }
  }
}
```

Note that the only JSON values ever used are objects, arrays and strings.

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


