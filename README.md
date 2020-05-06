# yq

Another portable wrapper to jq. Written in Go and aims to be used exactly like jq, just that the input and output document is YAML. There are a [several other options](https://github.com/stedolan/jq/issues/467), but haven't found one which is a simple binary *and* is just like jq.

Forwards stdin (after translating to JSON) and arguments (where applicable) directly to jq. Forwards jq output (after translating to YAML) directly to stdout. jq errors are directly written to stderr.

So, just use it as you would use jq, no gimmicks. You need to have jq installed and available within your `$PATH`.

All YAML features what [gopkg.in/yaml.v3](https://github.com/go-yaml/yaml/tree/v3#compatibility) offers are supported.

## Installation

### Go Get

```bash
# go get github.com/dionysius/yq
```

## Usage

Just like jq. If you try to get the help, you get directly the output of jq:

```bash
# yq --help
jq - commandline JSON processor [version 1.5-1-a5b5cbe]
Usage: jq [options] <jq filter> [file...]
...
```

Same with the exit code, it's just forwarded. Except if an error happens during yq processing, then the exit code 128 is returned.

## Examples

As pretty print

```bash
# echo -e "foo: |\n  bar\n  baz\nlist:\n- lang: go\n- lang: c\n- lang: python" | yq '.'
foo: |
    bar
    baz
list:
  - lang: go
  - lang: c
  - lang: python
```

Use whatever jq offers in the filter

```bash
# echo -e "foo: |\n  bar\n  baz\nlist:\n- lang: go\n- lang: c\n- lang: python" | yq '.list'
- lang: go
- lang: c
- lang: python
# echo -e "foo: |\n  bar\n  baz\nlist:\n- lang: go\n- lang: c\n- lang: python" | yq '.list[] | select(.lang == "go")'
lang: go
```

You can also use the `-r`/`--raw-output` option:

```bash
# echo -e 'hello: "world:"' | yq '.hello' # without -r
'world:'
# echo -e 'hello: "world:"' | yq -r '.hello' # with -r
world:
# echo -e "foo: |\n  bar\n  baz\nlist:\n- lang: go\n- lang: c\n- lang: python" | yq '.foo' # without -r
|
    bar
    baz
# echo -e "foo: |\n  bar\n  baz\nlist:\n- lang: go\n- lang: c\n- lang: python" | yq -r '.foo' # with -r
bar
baz
```

## Caveats

- One document in, one document out. Doesn't support multiple documents (yet - Unsure how to detect that).
- Some options of jq may modify input parsing or output formatting, you might not be able to use them practically.
- Project is pretty young, expect missing (special) handling of some jq parameters.

## Debugging

Set or export `YQ_DEBUG=1` to get additional output on stderr for looking at how the JSON document for stdin or stdout of jq looks like. Current output can be improved.
