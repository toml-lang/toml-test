### A language agnostic test suite for TOML encoders and decoders

`toml-test` is a higher-order program that tests other 
[TOML](https://github.com/mojombo/toml)
decoders or encoders. The goal is to make it comprehensive.
Tests are divided into two groups: invalid TOML data and valid TOML 
data. Decoders that reject invalid TOML data pass invalid TOML tests. Deocoders 
that accept valid TOML data and output precisely what is expected pass valid 
tests. The output format is JSON, described below.

Both decoders and encoders share valid tests, except an encoder accepts JSON 
and outputs TOML. The TOML representations are read with a blessed decoder and 
compared. Note though that encoders have their own set of invalid tests in the 
invalid-encoder directory. The JSON given to a TOML encoder is in the same 
format as the JSON that a TOML decoder should output.

Version: v0.4.0 (in sync with TOML)

Compatible with TOML version
[v0.4.0](https://github.com/mojombo/toml/blob/master/versions/toml-v0.4.0.md)

Dependencies: [Go](http://golang.org).


### Try it out

All you need is to have [Go](http://golang.org) installed. Then simply
use:

```bash
cd
export GOPATH=$HOME/go # if it isn't already set
go get github.com/BurntSushi/toml-test # install test suite
go get github.com/BurntSushi/toml/cmd/toml-test-decoder # e.g., install my parser
$HOME/go/bin/toml-test $HOME/go/bin/toml-test-decoder # e.g., run tests on my parser
# Outputs: 64 passed, 0 failed
```

The `go get` commands install Go packages and binaries into your `GOPATH`.

To test your decoder, you will have to satisfy the interface expected by 
`toml-test` described below. Then just execute `toml-test your-decoder` in the
`toml-test` directory to run your decoder against all tests.

To test your encoder, the instructions are the same, except the input/output
is reversed, and you'll need to run `toml-test -encoder your-encoder`.
(You can install my TOML encoder with `go get 
github.com/BurntSushi/toml/cmd/toml-test-encoder`.)


### Interface of a decoder

For your decoder to be compatible with `toml-test`, it **must** satisfy the 
interface expected.

Your decoder **must** accept TOML data on `stdin` until EOF.

If the TOML data is invalid, your decoder **must** return with a non-zero
exit code indicating an error.

If the TOML data is valid, your decoder **must** output a JSON encoding of that 
data on `stdout` and return with a zero exit code indicating success.


### Interface of an encoder

For your encoder to be compatible with `toml-test`, it **must** satisfy the 
interface expected.

Your encoder **must** accept JSON data on `stdin` until EOF.

If the JSON data cannot be converted to a valid TOML representation, your 
encoder **must** return with a non-zero exit code indicating an error.

If the JSON data can be converted to a valid TOML representation, your encoder 
**must** output a TOML encoding of that data on `stdout` and return with a zero 
exit code indicating success.


### JSON encoding

The following JSON encoding applies equally to both encoders and decoders.

* TOML tables correspond to JSON objects.
* TOML table arrays correspond to JSON arrays.
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

Empty hashes correspond to empty JSON objects (i.e., `{}`) and empty arrays 
correspond to empty JSON arrays (i.e., `[]`).

Datetime should be encoded following RFC 3339.

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


### Assumptions of Truth

The following are taken as ground truths by `toml-test`:

* All tests classified as `invalid` **are** invalid.
* All tests classified as `valid` **are** valid.
* All expected outputs in `valid/test-name.json` are exactly correct.
* The Go standard library package `encoding/json` decodes JSON correctly.
* When testing encoders, the TOML decoder at
  [BurntSushi/toml](https://github.com/BurntSushi/toml) is assumed to be 
  correct. (Note that this assumption is not made when testing decoders!)

Of particular note is that **no TOML decoder** is taken as ground truth when 
testing decoders. This means that most changes to the spec will only require an 
update of the tests in `toml-test`. (Bigger changes may require an adjustment 
of how two things are considered equal. Particularly if a new type of data is 
added.) Obviously, this advantage does not apply to testing TOML encoders since 
there must exist a TOML decoder that conforms to the specification in order to 
read the output of a TOML encoder.


### Adding tests

`toml-test` was designed so that tests can be easily added and removed. As 
mentioned above, tests are split into two groups: invalid and valid tests. 

Invalid tests **only check if a decoder rejects invalid TOML data**. Or, in the 
case of testing encoders, invalid tests **only check if an encoder rejects an 
invalid representation of TOML** (e.g., a hetergeneous array).
Therefore, all invalid tests should try to **test one thing and one thing 
only**. Invalid tests should be named after the fault it is trying to expose.
Invalid tests for decoders are in the `tests/invalid` directory while invalid 
tests for encoders are in the `tests/invalid-encoder` directory.

Valid tests check that a decoder accepts valid TOML data **and** that 
the parser has the correct representation of the TOML data. Therefore, valid 
tests need a JSON encoding in addition to the TOML data. The tests should be 
small enough that writing the JSON encoding by hand will not give you brain 
damage. The exact reverse is true when testing encoders.

A valid test without either a `.json` or `.toml` file will automatically fail.

If you have tests that you'd like to add, please submit a pull request.


### Why JSON?

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


### Decoders or encoders that satisfy the `toml-test` interface

If you have an implementation, send a pull request adding to this list. Please 
note the commit SHA1 or version tag that your parser supports in your `README`.

* C (@ajwans) - https://github.com/ajwans/libtoml
* C++ (@skystrife) - https://github.com/skystrife/cpptoml
* Go (@thompelletier) - https://github.com/pelletier/go-toml
* Go w/ Reflection (@BurntSushi) - https://github.com/BurntSushi/toml/tree/master/cmd/toml-test-decoder
* LabVIEW (@dbtaylor) - https://github.com/erdosmiller/lv-toml
* Node.js/Browser (@redhotvengeance) - https://github.com/redhotvengeance/topl
* PHP (@leonelquinteros) - https://github.com/leonelquinteros/php-toml
* Python (@uiri) - https://github.com/uiri/toml
* Python (@marksteve) - https://github.com/marksteve/toml-ply
* Racket (@greghendershott) - https://github.com/greghendershott/toml
* Ruby (@jm, @cespare) - https://gist.github.com/cespare/5052442
* Rust (@mneumann) - https://github.com/mneumann/rust-toml

N.B. Your decoder/encoder doesn't need to pass all tests to be on this list. 


### TOML projects using the test suite

I'm not sure why, but some projects seem to build their own testing harness 
while using the tests in this repository. That's OK, but it's probably more 
work than necessary. Plus, I claim that `toml-test` outputs nice error 
messages.

* Haskell (@cies) - https://github.com/cies/htoml
* Julia (@pygy) - https://github.com/pygy/TOML.jl
* PHP (@yosymfony) - https://github.com/yosymfony/toml
* Python (@f03lipe) - https://github.com/f03lipe/toml-python
* JavaScript (@iarna) - https://github.com/iarna/iarna-toml
