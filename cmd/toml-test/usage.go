package main

const usage = `Usage: %[1]s parser-cmd [ parser-cmd-flags ]

toml-test is a tool to verify the correctness of TOML parsers and writers.
https://github.com/toml-lang/toml-test

The parser-cmd positional argument should be a program that accepts TOML data
on stdin until EOF, and is expected to write the corresponding JSON encoding on
stdout. See README.md for details on how to write parser-cmd.

Any further positional arguments are passed to parser-cmd; stop toml-test's
flag parsing with -- to use flags; for example to use -A as a flag for
toml-test and -B as a flag for my-parser:

   $ %[1]s -A -- my-parser -B

There are two tests:

    decoder    This is the default.
    encoder    When -encoder is given.

Tests are split in to "valid" and "invalid" groups:

   valid           Valid TOML files
   invalid         Invalid TOML files that should be rejected with an error.

All tests are referred to relative to to the tests/ directory: valid/dir/name
or invalid/dir/name.

Flags:

    -h, -help      Show this help and exit.

    -V, -version   Show version and exit. Add twice to show detailed info.

    -encoder       The given parser-cmd will be tested as a TOML encoder rather
                   than a decoder.

                   The parser-cmd will be sent JSON on stdin and is expected to
                   write TOML to stdout. The JSON will be in the same format as
                   specified in the toml-test README. Note that this depends on
                   the correctness of my TOML parser!

    -json          Output as JSON.

    -toml          Select TOML version to run tests for. Supported versions are
                   "1.0" and "1.1" (which isn't released yet and may change).
                   Use "latest" to use the latest published TOML version.
                   Default is latest.

    -timeout       Maximum time for a single test run, to detect infinite loops
                   or pathological cases. Defaults to "1s".

    -list-files    List all test files, one file per line, and exit without
                   running anything. This takes the -toml flag in to account,
                   but none of the other flags.

    -cat           Keep outputting (valid) TOML from testcases until the file
                   reaches this many KB. Useful for generating benchmarks.

                   E.g. to create 1M and 100M files:

                       $ toml-test -cat 1024              >1M.toml
                       $ toml-test -cat $(( 1024 * 100 )) >100M.toml

                   The -skip, -run, and -toml flags can be used in combination
                   with -cat.

    -copy          Copy all test files to the given directory. This will take
                   the -toml flag in to account, so it only copies files for
                   the given version. (The test files are compiled in the
                   binary, this will only require the toml-test binary).

    -v             List all tests, even passing ones. Add twice to show
                   detailed output for passing tests.

    -run           List of tests to run; the default is to run all tests.

                   Test names include the directory, i.e. "valid/test-name" or
                   "invalid/test-name". You can use globbing patterns , for
                   example to run all string tests:

                       $ toml-test toml-test-decoder -run 'valid/string*'

                   You can specify this argument more than once, and/or specify
                   multiple tests by separating them with a comma:

                       $ toml-test toml-test-decoder \
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

    -script        Print a small bash/zsh script with -skip flag for failing
                   tests; useful to get a list of "known failures" for CI
                   integrations and such.

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

    -no-number     Don't add line numbers to output.

    -color         Output color; possible values:

                        always   Show test failures in bold and red.
                        bold     Show test failures in bold only.
                        never    Never output any escape codes.

                   Default is "always", or "never" if NO_COLOR is set.
`

// vim:et:tw=79
