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

Compatible with TOML versions [v1.0.0] and [v1.1.0].

[TOML]: https://toml.io
[v1.0.0]: https://toml.io/en/v1.0.0
[v1.1.0]: https://toml.io/en/v1.1.0

Installation
------------
There are binaries on the [release page]; these are statically compiled and
should run in most environments. It's recommended you use a binary or a tagged
release if you build from source especially in CI environments. This prevents
your tests from breaking on changes to tests in this tool.

To compile from source you will need Go 1.19 or newer:

    % go install github.com/toml-lang/toml-test/v2/cmd/toml-test@latest

This will build a `toml-test` binary in the `~/go/bin` directory. You can change
that directory by setting `GOBIN`; for example to use the current directory:

    % GOBIN="$(pwd)" go install github.com/toml-lang/toml-test/v2/cmd/toml-test@latest

See [CHANGELOG.md] for a list of changes.

[release page]: https://github.com/toml-lang/toml-test/releases
[CHANGELOG.md]: ./CHANGELOG.md

Usage
-----
`toml-test test` runs the test suite, and requires an `-decoder` and/or
`-encoder` flag; for example:

    # Install my parser
    % go install github.com/BurntSushi/toml/cmd/toml-test-decoder@master
    % go install github.com/BurntSushi/toml/cmd/toml-test-encoder@master

    # Run tests
    % toml-test test \
       -decoder=toml-test-decoder \
       -encoder=toml-test-encoder

    [..]

    toml-test v2025-12-16 [toml-test-decoder] [toml-test-encoder]
      valid tests: 205 passed,  0 failed
    encoder tests: 205 passed,  0 failed
    invalid tests: 460 passed, 15 failed

You can use `-run [name]` or `-skip [name]` to run or skip specific tests. Both
flags can be given more than once and accept glob patterns: `-run
'valid/string/*'`.

See `toml-test test -help` for detailed usage.

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
- TOML arrays correspond to JSON arrays.
- TOML values correspond to a JSON object of the form:
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

Empty tables correspond to empty JSON objects (`{}`) and empty arrays correspond
to empty JSON arrays (`[]`).

Offset datetimes should be encoded in RFC 3339; Local datetimes should be
encoded following RFC 3339 without the offset part. Local dates should be
encoded as the date part of RFC 3339 and local times as the time part.

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

<!-- -->

    [[arr]]             {"arr": [
    a = 1                   {
    b = 2                       "a": {"type": "integer", "value": "1"},
    [[arr]]                     "b": {"type": "integer", "value": "2"}
    a = 3                   }, {
    b = 4                       "a": {"type": "integer", "value": "3"},
                                "b": {"type": "integer", "value": "4"}
                            }
                        ]}

Note that the only JSON values ever used are objects, arrays and strings.

An example implementation can be found in the BurnSushi/toml:

- [Add tags](https://github.com/BurntSushi/toml/blob/master/internal/tag/add.go)
- [Remove tags](https://github.com/BurntSushi/toml/blob/master/internal/tag/rm.go)

Untested and implementation-defined behaviour
---------------------------------------------
This only tests behaviour that should be true for every encoder implementing
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

Usage without `toml-test` binary
--------------------------------
While the `toml-test` tool is a convenient way to run the tests, you can also
re-implement its behaviour in your own language's test-suite, which may be an
easier way to run the tests.

Because multiple TOML versions are supported, not all tests are valid for every
version of TOML; for example the 1.0.0 tests contain a test that trailing commas
in tables are invalid, but in 1.1.0 this should be considered valid.

In short: you can't "just" copy all .toml and .json files from the tests/
directory. The easiest way to copy the correct files is to use `copy`:

    # Default of TOML 1.0
    % toml-test copy ./tests

    # Use TOML 1.1
    % toml-test copy -toml 1.1.0 ./tests

Alternatively, the [tests/files-toml-1.0.0] and [tests/files-toml-1.1.0] files
contain a list of files that should be run for that TOML version. This list is
generated from the `toml-test list` output.

[tests/files-toml-1.0.0]: tests/files-toml-1.0.0
[tests/files-toml-1.1.0]: tests/files-toml-1.1.0

Adding tests
------------
`toml-test` was designed so that tests can be easily added and removed. As
mentioned above, tests are split into two groups: invalid and valid tests.

Invalid tests **only check if a decoder rejects invalid TOML data**. Or, in the
case of testing encoders, invalid tests **only check if an encoder rejects an
invalid representation of TOML** (e.g., a heterogeneous array). Therefore, all
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
