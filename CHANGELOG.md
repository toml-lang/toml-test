v2.1.0 v2.0.0 2025-12-27
------------------------
Fix two issues in the 2.0 release:

- Add back the `list` command and `tests/files-toml-*` files that were removed
  in 2.0, as it turns out people were using them.

- Fix printing of version of `test` and `version` commands when built with 
  `go install go install github.com/toml-lang/toml-test/v2/cmd/toml-test@latest`.

v2.0.0 2025-12-18
-----------------
This release has a number of incompatible changes. It also sets TOML 1.1 as the
default version to test.

Note the correct import path changed from `github.com/toml-lang/toml-test` to
`github.com/toml-lang/toml-test/v2`. If you're installing toml-test from source
you now need to do:

    % go install github.com/toml-lang/toml-test/v2/cmd/toml-test@latest

Without the /v2, `@latest` will install the latest 1.x release.

### Incompatible changes
- The CLI syntax has been cleaned up to make it easier to use. The main
  motivation was to enable running all tests (decoder and encoder) with just one
  `toml-test` invocation, which was hard to do in a compatible way.

  To run the tests, you now need to use the `test` subcommand. And the decoder
  and encoder binaries are specified via the `-decoder` and `-encoder` flags
  rather than as positional arguments.

  Before:

      % toml-test my-decoder
      % toml-test -encoder my-encoder

  Now:

      % toml-test test -decoder=my-decoder -encoder=my-encoder
  
- `toml-test -copy` is now `toml-test copy`.

- `toml-test -version` is now `toml-test version`.

- `toml-test -list` has been removed; I don't think anyone was using it(?) and
  the `copy` command should cover most use cases. It can be added again if
  someone needs it.

- Rename `-print-skip` flag to `-script`.

- Remove `-testdir` flag to load test files from the filesystem. Tests have been
  built in the binary since 1.0.0-beta1 (2021), and this flag has not really
  been useless ever since.

- It now tests TOML 1.1 by default. Use `-toml=1.0` if you want to test TOML
  1.0.

### Other changes
- Add various new tests, and rename some existing tests for consistency.

- Add `-json` flag to the `test` command, to print test output as JSON rather
  than text.

- Allow setting `-toml=latest`.

- Add `-skip-must-err` flag to the `test` command, to treat skipped tests that
  don't fail as an error.

v1.6.0, 2025-04-15
------------------
This is a small maintenance release which fixes a few small bugs in the test
runner and adds a few small tests. See the git log for details: v1.5.0...v1.6.0

v1.5.0, 2024-05-31
------------------
### Changes
- This release requires Go 1.19 to build.

- Add quite a lot of new test.

- Only "pass" an invalid test if the decoder exits with exactly exit 1, rather
  than any exit >0. This catches segfaults, panics, and other crashes which
  shouldn't be considered "passing".

- Tests are now run in parallel, defaulting to the number of available cores.
  Use the `-parallel` flag to set the number of cores to use.

- Few small improvements to toml-test runner output.

### New features
- The `-copy` flag copies all tests to the given directory (taking the `-toml`
  flag in to account). This is much easier than manually copying the files.

- Add `-errors` flag to test expected error messages for invalid tests. See
  `-help` for details.

- Add `-print-skip`, to print out a small bash/zsh script with `-skip` flags for
  tests that failed. Useful to get a list of "known failures" for CI
  integrations and such.

- Add `-timeout` flag to set the maximum execution time per test, to catch
  infinite loops and/or pathological cases. This defaults to 1s, but can
  probably be set (much) lower for most implementations.

- Add `-int-as-float` flag, for implementations that treat all numbers as
  floats.

- Add `-cat` flag to create a large (valid) TOML document, for benchmarks and
  such.

v1.4.0, 2023-09-29
------------------
- Move from github.com/BurntSushi/toml-test to github.com/toml-lang/toml-test

  In most cases things should keep working as GitHub will redirect things, but
  you'll have to update the path if you install from source with `go install`.

- Both TOML 1.0 and the upcoming TOML 1.1 are now supported.

  If you implemented your own test-runner, then you should only copy/use the
  files listed in `tests/files-toml-1.0.0` (or `tests/files-toml-1.1.0`). Some
  things that are invalid in 1.0 are now valid in 1.1.

  Also see "Usage without toml-test binary" in the README.md.

  For the `toml-test` tool the default remains 1.0; add `-toml 1.1.0` to use
  TOML 1.1.

- Add a few tests, and improve output on test failures a bit.

v1.2.0, 2023-01-15
------------------
A few minor fixes and additional tests; see the git log for details:
v1.2.0...v1.3.0

v1.2.0, 2022-06-02
------------------
A few minor fixes and additional tests; see the git log for details:
v1.1.0...v1.2.0

v1.1.0, 2022-01-12
------------------
Adds various tests; see the git log for details: v1.0.0...v1.1.0

v1.0.0, 2021-08-04
------------------
Many changes since the last release in 2013: much improved error output, support
TOML 1.0.0, add several flags to give more control over which tests to run/skip.

Some minor incompatibilities in the test tool:

- You no longer need to add a type hint to arrays.
- Tests are always referenced as valid/[...] or invalid/[..]
- The datetime-local, date-local, and time-local types are added. You will need
  to add support for this in your -encode and -decode test helpers.
