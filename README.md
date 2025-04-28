`toml-test` is a language-agnostic test suite to verify the correctness of
[TOML] parsers and writers.

Tests are divided into two groups: "invalid" and "valid". Decoders or encoders
that reject "invalid" tests pass the tests, and decoders that accept "valid"
tests and output precisely what is expected pass the tests. The output format is
JSON, described below.

Both encoders and decoders share valid tests, except an encoder accepts JSON and
outputs TOML rather than the reverse. The TOML representations are read with a
blessed decoder and is compared. The JSON given to a TOML encoder is in the same
format as the JSON that a TOML decoder should output.

If you find something in your parser that's not exactly covered by toml-test
already then it should be added here; just creating an issue is enough: don't
*need* to create a PR.

Compatible with TOML version [v1.0.0].

[TOML]: https://toml.io
[v1.0.0]: https://toml.io/en/v1.0.0

Installation
------------
There are binaries on the [release page]; these are statically compiled and
should run in most environments. It's recommended you use a binary, or a tagged
release if you build from source especially in CI environments. This prevents
your tests from breaking on changes to tests in this tool.

To compile from source you will need Go 1.19 or newer:

    % go install github.com/toml-lang/toml-test/cmd/toml-test@latest

This will build a `toml-test` binary in the `~/go/bin` directory. You can change
that directory by setting `GOBIN`; for example to use the current directory:

    % GOBIN="$(pwd)" go install github.com/toml-lang/toml-test/cmd/toml-test@latest

[release page]: https://github.com/toml-lang/toml-test/releases

Running in CI
-------------
The [setup-toml-test] action can be used in GitHub. See the README for more
details.

For other CI systems: the action essentially just downloads a release binary.
See index.js. Should be easy enough to reproduce elsewhere.

[setup-toml-test]: https://github.com/toml-lang/setup-toml-test

Usage
-----
`toml-test` accepts an encoder or decoder as the first positional argument, for
example:

    % toml-test my-toml-decoder
    % toml-test my-toml-encoder -encoder

The `-encoder` flag is used to signal that this is an encoder rather than a
decoder.

For example, to run the tests against the Go TOML library:

    # Install my parser
    % go install github.com/BurntSushi/toml/cmd/toml-test-decoder@master
    % go install github.com/BurntSushi/toml/cmd/toml-test-encoder@master

    % toml-test toml-test-decoder
    toml-test v2023-10-23 [toml-test-decoder]: using embeded tests: 278 passed

    % toml-test -encoder toml-test-encoder
    toml-test v2023-10-23 [toml-test-encoder]: using embeded tests:  94 passed,  0 failed

You can use `-run [name]` or `-skip [name]` to run or skip specific tests. Both
flags can be given more than once and accept glob patterns: `-run
'valid/string/*'`.

See `toml-test -help` for detailed usage.

### Implementing a decoder
For your decoder to be compatible with `toml-test` it **must** satisfy the
expected interface:

- Your decoder **must** accept TOML data on `stdin`.
- If the TOML data is invalid, your decoder **must** return with a non-zero
  exit code, indicating an error.
- If the TOML data is valid, your decoder **must** output a JSON encoding of
  that data on `stdout` and return with a zero exit code, indicating success.

An example in pseudocode:

    toml_data = read_stdin()

    parsed_toml = decode_toml(toml_data)

    if error_parsing_toml():
        print_error_to_stderr()
        exit(1)

    print_as_tagged_json(parsed_toml)
    exit(0)

Details on the tagged JSON is explained below in "JSON encoding".

### Implementing an encoder
For your encoder to be compatible with `toml-test`, it **must** satisfy the
expected interface:

- Your encoder **must** accept JSON data on `stdin`.
- If the JSON data cannot be converted to a valid TOML representation, your
  encoder **must** return with a non-zero exit code, indicating an error.
- If the JSON data can be converted to a valid TOML representation, your encoder
  **must** output a TOML encoding of that data on `stdout` and return with a
  zero exit code, indicating success.

An example in pseudocode:

    json_data = read_stdin()

    parsed_json_with_tags = decode_json(json_data)

    if error_parsing_json():
        print_error_to_stderr()
        exit(1)

    print_as_toml(parsed_json_with_tags)
    exit(0)

JSON encoding
-------------
The following JSON encoding applies equally to both encoders and decoders:

- TOML tables correspond to JSON objects.
- TOML table arrays correspond to JSON arrays.
- TOML values correspond to a special JSON object of the form:
  `{"type": "{TOML_TYPE}", "value": "{TOML_VALUE}"}`

In the above, `TOML_TYPE` may be one of:

- string
- integer
- float
- bool
- datetime
- datetime-local
- date-local
- time-local

`TOML_VALUE` is always a JSON string.

Empty hashes correspond to empty JSON objects (`{}`) and empty arrays correspond
to empty JSON arrays (`[]`).

Offset datetimes should be encoded in RFC 3339; Local datetimes should be
encoded following RFC 3339 without the offset part. Local dates should be
encoded as the date part of RFC 3339 and Local times as the time part.

Examples:

    TOML                JSON

    a = 42              {"type": "integer", "value": "42"}

<!-- -->

    [tbl]               {"tbl": {
    a = 42                  "a": {"type": "integer", "value": "42"}
                        }}

<!-- -->

    a = ["a", 2]        {"a": [
                            {"type": "string",  "value": "a"},
                            {"type": "integer", "value": "2"}
                        ]}

Or a more complex example:

```toml
best-day-ever = 1987-07-05T17:45:00Z

[numtheory]
boring     = false
perfection = [6, 28, 496]
```

And the JSON encoding expected by `toml-test` is:

```json
{
  "best-day-ever": {"type": "datetime", "value": "1987-07-05T17:45:00Z"},
  "numtheory": {
    "boring": {"type": "bool", "value": "false"},
    "perfection": [
      {"type": "integer", "value": "6"},
      {"type": "integer", "value": "28"},
      {"type": "integer", "value": "496"}
    ]
  }
}
```

Note that the only JSON values ever used are objects, arrays and strings.

An example implementation can be found in the BurnSushi/toml:

- [Add tags](https://github.com/BurntSushi/toml/blob/master/internal/tag/add.go)
- [Remove tags](https://github.com/BurntSushi/toml/blob/master/internal/tag/rm.go)

Untested and implementation-defined behaviour
---------------------------------------------
This only tests behaviour that's should be true for every encoder implementing
TOML; a few things are left up to implementations, and are not tested here.

- TOML does not mandate a specific integer or float size, but recommends int64
  and float64, which is what this tests. You'll have to manually -skip these
  tests if your implementation doesn't support it.

- Many values can be expressed in more than one way: for example `0xff` and
  `255` are equal, as are `0.0` and `-0.0`.

  Some encoders may choose to always write back in a certain format (e.g.
  decimal), others may choose to use the same as the input format.

  Testing how exactly a value is written in encoder tests is left up to the
  implementation, as long as they're semantically identical the test is
  considered to pass.

- Millisecond precision (3 digits) is required for datetimes and times, and
  further precision is implementation-specific, and any greater precision than
  is supported must be truncated (not rounded).

  This tests only millisecond precision, and not any further precision or the
  truncation of it.

Assumptions of Truth
--------------------
The following are taken as ground truths by `toml-test`:

- All tests classified as `invalid` **are** invalid.
- All tests classified as `valid` **are** valid.
- All expected outputs in `valid/test-name.json` are exactly correct.
- The Go standard library package `encoding/json` decodes JSON correctly.
- When testing encoders the
  [BurntSushi/toml](https://github.com/BurntSushi/toml) TOML decoder is assumed
  to be correct. (Note that this assumption is not made when testing decoders!)

Of particular note is that **no TOML decoder** is taken as ground truth when
testing decoders. This means that most changes to the spec will only require an
update of the tests in `toml-test`. (Bigger changes may require an adjustment of
how two things are considered equal. Particularly if a new type of data is
added.) Obviously, this advantage does not apply to testing TOML encoders since
there must exist a TOML decoder that conforms to the specification in order to
read the output of a TOML encoder.

Usage without `toml-test` binary
--------------------------------
While the `toml-test` tool is a convenient way to run the tests, you can also
re-implement its behaviour in your own language's test-suite, which may be an
easier way to run the tests.

Because multiple TOML versions are supported, not all tests are valid for every
version of TOML; for example the 1.0.0 tests contain a test that trailing commas
in tables are invalid, but in 1.1.0 this should be considered valid.

In short: you can't "just" copy all .toml and .json files from the tests/
directory. The easiest way to copy the correct files is to use `-copy`:

    # Default of TOML 1.0
    % toml-test -copy ./tests

    # Use TOML 1.1
    % toml-test -copy ./tests -toml 1.1.0

Alternatively, the [tests/files-toml-1.0.0] and [tests/files-toml-1.1.0] files
contain a list of files you should run for that TOML version. You can use them
with something like:

    <files-toml-1.0.0 while read l; do
        mkdir -p ~/my-test/"$(dirname "$l")"
        cp -r "$l" ~/my-test/"$l"
    done

[tests/files-toml-1.0.0]: tests/files-toml-1.0.0
[tests/files-toml-1.1.0]: tests/files-toml-1.1.0

Adding tests
------------
`toml-test` was designed so that tests can be easily added and removed. As
mentioned above, tests are split into two groups: invalid and valid tests.

Invalid tests **only check if a decoder rejects invalid TOML data**. Or, in the
case of testing encoders, invalid tests **only check if an encoder rejects an
invalid representation of TOML** (e.g., a hetergeneous array). Therefore, all
invalid tests should try to **test one thing and one thing only**. Invalid tests
should be named after the fault it is trying to expose. Invalid tests for
decoders are in the `tests/invalid` directory while invalid tests for encoders
are in the `tests/invalid-encoder` directory.

Valid tests check that a decoder accepts valid TOML data **and** that the parser
has the correct representation of the TOML data. Therefore, valid tests need a
JSON encoding in addition to the TOML data. The tests should be small enough
that writing the JSON encoding by hand will not give you brain damage. The exact
reverse is true when testing encoders.

A valid test without either a `.json` or `.toml` file will automatically fail.

If you have tests that you'd like to add, please submit a pull request.

Why JSON?
---------
In order for a language agnostic test suite to work, we need some kind of data
exchange format. TOML cannot be used, as it would imply that a particular parser
has a blessing of correctness.

My decision to use JSON was not a careful one. It was based on expediency. The
Go standard library has an excellent `encoding/json` package built in, which
made it easy to compare JSON data.

The problem with JSON is that the types in TOML are not in one-to-one
correspondence with JSON. This is why every TOML value represented in JSON is
tagged with a type annotation, as described above.

YAML may be closer in correspondence with TOML, but I don't believe we should
rely on that correspondence. Making things explicit with JSON means that writing
tests is a little more cumbersome, but it also reduces the number of assumptions
we need to make.
