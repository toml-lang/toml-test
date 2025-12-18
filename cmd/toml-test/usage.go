package main

import "strings"

var helpTopics = map[string]string{
	"":        usage,
	"test":    usageTest,
	"list":    usageList,
	"ls":      usageList,
	"copy":    usageCopy,
	"cp":      usageCopy,
	"version": usageVersion,
}

var usage = `
toml-test is a tool to verify the correctness of TOML parsers and writers.
https://github.com/toml-lang/toml-test

Use "toml-test help «cmd»" or "toml-test «cmd» -h" for more documentation on a
command.

Commands:

    help      Show help and exit.
    test      Run tests. See "help test" for details.
    copy      Write all test files to disk.
    list      List test filenames.
    version   Show version and exit.
`[1:]

var usageTest = strings.ReplaceAll(`
The "test" command runs the test cases.

The way toml-test works is that for every test case it executes a "decoder"
command, which reads TOML from stdin and then outputs JSON describing that TOML
(or returns an error if it thinks the TOML isn't valid). Encoder tests work the
same except in reverse: it reads JSON and transforms that to TOML.

There are three types of tests:

    valid      Valid TOML that the decoder command should describe as JSON.
    invalid    Invalid TOML files that should be rejected with an error.
    encoder    JSON that the encoder command should transform to TOML.

\x1b[1mImplementing a decoder:\x1b[0m

    A decoder reads TOML data from stdin and outputs a JSON description to
    stdout and exits with code 0. If the TOML data is invalid, it must exit
    with code 1. It's recommended to write an error message to stderr.

    An example in pseudocode:

        toml_data = read_stdin()

        parsed_toml = decode_toml(toml_data)

        if error_parsing_toml():
            print_error_to_stderr()
            exit(1)

        print_as_json_description(parsed_toml)
        exit(0)

    Details of the JSON format is explained below in "JSON description".

\x1b[1mImplementing an encoder:\x1b[0m

    An encoder is the reverse of a decoder; it reads a JSON description from
    stdin converts that to TOML.

    An example in pseudocode:

        json_data = read_stdin()

        json_description = decode_json(json_data)

        print_as_toml(json_description)
        exit(0)

\x1b[1mJSON description\x1b[0m

    TOML is described with JSON as follows:

    - TOML tables correspond to JSON objects.
    - TOML arrays correspond to JSON arrays.
    - TOML values correspond to a JSON object of the form:
      {"type": "{TOML_TYPE}", "value": "{TOML_VALUE}"}

    In the above, TOML_VALUE is always a JSON string (even for integers or
    bools), and TOML_TYPE may be one of: string, integer, float, bool,
    datetime, datetime-local, date-local, or time-local.

    Empty tables correspond to empty JSON objects ({}) and empty arrays
    correspond to empty JSON arrays ([]).

    Offset datetimes should be encoded in RFC 3339; Local datetimes should be
    encoded following RFC 3339 without the offset part. Local dates should be
    encoded as the date part of RFC 3339 and local times as the time part.

    Examples:

        ┌───────────────┬────────────────────────────────────────────────────┐
        │ TOML          │ JSON                                               │
        ├───────────────┼────────────────────────────────────────────────────┤
        │ a = 42        │   {"type": "integer", "value": "42"}               │
        ├───────────────┼────────────────────────────────────────────────────┤
        │ [tbl]         │   {"tbl": {                                        │
        │ a = 42        │       "a": {"type": "integer", "value": "42"}      │
        │               │   }}                                               │
        ├───────────────┼────────────────────────────────────────────────────┤
        │ a = ["a", 2]  │   {"a": [                                          │
        │               │       {"type": "string",  "value": "a"},           │
        │               │       {"type": "integer", "value": "2"}            │
        │               │   ]}                                               │
        ├───────────────┼────────────────────────────────────────────────────┤
        │ [[arr]]       │   {"arr": [                                        │
        │ a = 1         │       {                                            │
        │ b = 2         │           "a": {"type": "integer", "value": "1"},  │
        │ [[arr]]       │           "b": {"type": "integer", "value": "2"}   │
        │ a = 3         │       }, {                                         │
        │ b = 4         │           "a": {"type": "integer", "value": "3"},  │
        │               │           "b": {"type": "integer", "value": "4"}   │
        │               │       }                                            │
        │               │   ]}                                               │
        └───────────────┴────────────────────────────────────────────────────┘

\x1b[1mFlags:\x1b[0m

    -decoder       Decoder command to use: this should read TOML from stdin,
                   and output the JSON description on stdout. On errors it
                   should exit with code 1.

    -encoder       Encoder command to use; this should read JSON from stdin,
                   and output TOML on stdout. The JSON is in the same format as
                   specified in the toml-test README. May be omitted if writing
                   TOML isn't supported.

    -json          Output report as JSON rather than text.

    -script        Print a small bash/zsh script with -skip flag for failing
                   tests; useful to get a list of "known failures" for CI
                   integrations and such.

    -toml          TOML version to run tests for,  "1.0", "1.1", or "latest"
                   for the latest published TOML version. Default is latest.

    -timeout       Maximum time for a single test run, to detect infinite loops
                   or pathological cases. Defaults to "1s".

    -v             List all tests, even passing ones. Add twice to show
                   detailed output for passing tests.

    -run           Rests to run; the default is to run all tests.

                   Test names include the directory, i.e. "valid/test-name" or
                   "invalid/test-name". You can use globbing patterns , for
                   example to run all string tests:

                       % toml-test toml-test-decoder -run 'valid/string*'

                   You can specify this argument more than once, and/or specify
                   multiple tests by separating them with a comma:

                       % toml-test toml-test-decoder \
                           -run valid/string-empty \
                           -run valid/string-nl,valid/string-simple

                   This will run three tests (string-empty, string-nl,
                   string-simple).

                   Quote glob characters so they won't be picked up by the
                   shell.
                   Supported patterns: https://godocs.io/path/filepath#Match

    -skip          Tests to skip, this uses the same syntax as the -run flag.

    -skip-must-err It's an error if tests in -skip don't fail. Useful for CI.

    -parallel      Number of tests to run in parallel; defaults to GOMAXPROCS,
                   which is normally the number of cores available.

    -int-as-float  Treat all integers as floats, rather than integers. This
                   also skips the int64 test as that's outside of the safe
                   float range (it still tests the boundary of safe float64
                   natural numbers).

    -errors        TOML or JSON file with expected errors for invalid test
                   files; an invalid test is considered to be "failed" if the
                   output doesn't contain the string in the file. This is
                   useful to ensure/test that your errors are what you expect
                   them to be.

                   The key is the file name, with or without invalid/ or .toml,
                   and the value is the expected error. For example:

                       "table/equals-sign"              = "expected error text"
                       "invalid/float/exp-point-1.toml" = "error"

                   It's not an error if a file is missing in the file, but it
                   is an error if the filename in the errors.toml file doesn't
                   exist.

    -color         Output color; possible values:

                        always   Show test failures in bold and red.
                        bold     Show test failures in bold only.
                        never    Never output any escape codes.

                   Default is "always", or "never" if NO_COLOR is set.
`, `\x1b`, "\x1b")[1:]

var usageCopy = `
The "copy" command writes all test files to disk.

Must have a path as the first positional argument.

This is useful for parsers that want to use the test cases without using the
toml-test test runner.

This also creates a version.toml with some information about the toml-test
version, and a .gitattributes file which prevents line-ending transformations
on *.toml files.

Existing files are overwritten, but deleted or renamed test files are not
deleted. It's generally recommended to remove and re-create the directory on
updates.

\x1b[1mFlags:\x1b[0m

    -toml          TOML version to copy tests for (1.0, 1.1, or latest).
`

var usageList = `
The "list" command lists all testfiles.

This is mainly intended to generate the tests/files-toml-* files, which can
make it more convenient to use the test cases without running toml-test.

\x1b[1mFlags:\x1b[0m

    -toml          TOML version to list tests for (1.0, 1.1, or latest).
`

var usageVersion = `
Show version and exit.

\x1b[1mFlags:\x1b[0m

    -v             Show detailed version info.
`

// vim:et:tw=79
