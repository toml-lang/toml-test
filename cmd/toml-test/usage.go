package main

const usage = `Usage: %[1]s parser-cmd [ parser-cmd-flags ]

toml-test is a tool to verify the correctness of TOML parsers and writers.
https://github.com/BurntSushi/toml-test

The parser-cmd positional argument should be a program that accepts TOML data
on stdin until EOF, and is expected to write the corresponding JSON encoding on
stdout. Please see 'README.md' for details on how to satisfy the interface
expected by 'toml-test' with your own parser.

Any positional arguments are use as the parser-cmd; to pass flags stop
toml-test's flag parsing with --:

   $ %[1]s -- my-parser -x -y

There are two tests:

    decoder    This is the default.
    encoder    When -encoder is given.

Tests are split in to "valid" and "invalid" groups:

   valid           Valid TOML files
   invalid         Invalid TOML files that should be rejected with an error.

All tests are referred to relative to to the tests/ directory: valid/dir/name or
invalid/dir/name.

Flags:

    -h, -help     Show this help and exit.

    -V, -version  Show version and exit. Add twice to show detailed build info.

    -encoder      The given parser-cmd will be tested as a TOML encoder rather
                  than a decoder.

                  The parser-cmd will be sent JSON on stdin and is expected to
                  write TOML to stdout. The JSON will be in the same format as
                  specified in the toml-test README. Note that this depends on
                  the correctness of my TOML parser!

    -toml         Select TOML version to run tests for. Supported versions are
                  "1.0.0" and "1.1.0" (which isn't released yet and may change).
                  Defaults to 1.0.0.

    -list-files   List all test files, one file per line, and exit without
                  running anything. This takes the -toml flag in to account, but
                  none of the other flags.

    -v            List all tests, even passing ones. Add twice to show detailed
                  output for passing tests.

    -run          Specify list of tests to run; the default is to run all tests.

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

                  Quote glob characters so they won't be picked up by the shell.
                  Supported paterns: https://godocs.io/path/filepath#Match

    -skip         Tests to skip, this uses the same syntax as the -run flag.

    -color        Output color; possible values:

                       always   Show test failures in bold and red.
                       bold     Show test failures in bold only.
                       never    Never output any escape codes.

                  Default is "always", or "never" if NO_COLOR is set.

    -testdir      Location of the tests; the default is to use the tests
                  compiled in the binary; this is only useful if you want to add
                  or modify tests.

                  A test in the invalid directory is a TOML file that is known
                  to be invalid and should be rejected by the parser.

                  A test in the valid directory is a TOML and JSON file with the
                  same name, where the json file is the JSON representation of
                  the TOML file according to the syntax described in the README.

                  For encoders only valid tests are run.
`
