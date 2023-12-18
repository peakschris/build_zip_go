### Go port of rules_pkg build_zip tool

Replacement for https://github.com/bazelbuild/rules_pkg/blob/0.9.1/pkg/private/zip/build_zip.py

#### Performance issues with pkg_zip in bazel
There are significant performance issues with pkg_zip in a bazel environment. We were finding when bazel was simultaneously packaging many zips, each one could take 45s instead of 2s expected. We discovered that this is because Bazel's hermetic python toolchain (rules_python) uncompresses many files for every single python invocation.

There are two workarounds:
 - use a local python instead of rules_python
 - don't use python

#### Go version of pkg_zip
This has been ported to Go with help from GitHub Copilot. I don't know Go. Errors are likely. No warranties are given. It seems to be working

#### Limitations
Go's compress library only support 'Deflate' compression OOTB, so this is the only method supported. Only tested on Windows so far. This is a port of release 0.9.1 of rules_pkg, now some months behind latest.

#### Usage
To use, load zip.bzl from this repository instead of from rules_pkg.
