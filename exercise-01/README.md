# Exercise 01

## Scenario

One of our teammates has written a tiny Golang CLI `converter.go` to convert
JSON files into a YAML ones.

Most developers have been using the tool out-of-band to convert upstream JSON
files into YAML before checking them into our company's source control system.

We want to perform the JSON to YAML conversion as part of the build to simplify
the process, avoid them the manual out-of-band conversion step, and ensure that
the source and generated files are always in sync.

## Goal

To implement the `json_to_yaml` build rule to convert an input JSON file to
YAML using `converter.go`.

## Steps

### Add the `input` attribute to `json_to_yaml`:

If you look at `defs.bzl`, you will realize that the current `rule` definition
does not specify any attributes or, in other words, its `attr` parameter is
currently assigned to an empty dictionary.

This effectively means that the `json_to_yaml` rule only accepts the default
and implicit `name` attribute (as you can see in `BUILD.bazel`).

Let's change that and add an new `input` attribute to let users specify
which JSON file they want to convert to YAML. To define this new attribute,
we will use the [`attr.label`](https://bazel.build/rules/lib/toplevel/attr#label)
function to indicate that the attribute value must be a file.

```diff
--- a/exercise-01/defs.bzl
+++ b/exercise-01/defs.bzl

    json_to_yaml(
        implementation = _json_to_yaml_impl,
-       attrs = {},
+       attrs = {
+           "input": attr.label(
+               mandatory = True,                    # The attribute is mandatory.
+               allow_single_file = [".json"],       # Only accept a single .json file.
+               doc = "The JSON file to convert.",
+           ),
+       },
    )
```

To validate the change, let's try to build the `//exercise-01:convert` target
and see if it produces a validation error because the `json_to_yaml` rule
instantiation in `BUILD.bazel` does not specify the now mandatory `input`
attribute.

```shell
$ bazel build //exercise-01:convert
...omitted output...
ERROR: //exercise-01:convert: missing value for mandatory attribute 'input' in 'json_to_yaml' rule
```

Great! Let's fix that by adding the `input` attribute on the `json_to_yaml` rule
instantiantion in the `BUILD.bazel` file.

```diff
--- a/exercise-01/BUILD.bazel
+++ b/exercise-01/BUILD.bazel

    json_to_yaml(
        name = "convert",
+       input = "data.json",
    )
```

To validate the change and make sure that the previous error is gone, let's try
to build `//exercise-01:convert` again. This time, we should hopefully get a
successful build :tada: .

```shell
$ bazel build //exercise-01:convert
INFO: Analyzed target //exercise-01:convert (0 packages loaded, 2 targets configured).
INFO: Found 1 target...
Target //exercise-01:convert up-to-date (nothing to build)
INFO: Elapsed time: 0.089s, Critical Path: 0.00s
INFO: 1 process: 1 internal.
INFO: Build completed successfully, 1 total action
```

### Inject the `converter` tool into the `json_to_yaml` rule

The `converter` tool can be run directly to convert JSON to YAML files.

```shell
$ bazel run //exercise-01:converter -- -i exercise-01/data.json -o /dev/stdout
... omitted output ...
apiVersion: v1
data:
    tls.crt: SXQncyBhIHNlY3JldCAhCg==
    tls.key: SXQncyBhIHNlY3JldCAhCg==
kind: Secret
metadata:
    name: some-secret
    namespace: some-namespace
type: kubernetes.io/tls
```

Unfortunately, Bazel rules (including `json_to_yaml`) are not allowed to invoke
`bazel` or other binaries that have not been passed to them explicitly.

Hence, let's pass the `converter` binary into our `json_to_yaml` rule by adding
a new `converter` attribute on the rule. We will use the `attr.label` attribute
function (as we did in the previous section) because we also want to receive a
file but, this time, with a few extra fields.

```diff
--- a/exercise-01/defs.bzl
+++ b/exercise-01/defs.bzl

    json_to_yaml = rule(
        implementation = _json_to_yaml_impl,
        attrs = {
            "input": attr.label(
                mandatory = True,
                allow_single_file = [".json"],
                doc = "The JSON file to convert.",
            ),
+           "converter": attr.label(
+               mandatory = True,            # The attribute is mandatory.
+               allow_single_file = True,    # Only accept a single file.
+               executable = True,           # The file must be executable.
+               cfg = "exec",                # Build the executable for the execution platform.
+               doc = "The JSON to YAML converter."
+           ),
        },
    )
```

Since the new `converter` attribute is mandatory, we need to add
it to all the instantiations of `json_to_yaml`. This is far from ideal because,
in most cases, users of `json_to_yaml` will not be opinionated about which
tool to use to perform the conversion. Hence, our design will benefit from a
narrower API that does **NOT** require users to pass a reference to the
`converter` binary.

To improve the current API design, we could:

- Make the `converter` attribute not mandatory and specify a default value. This
  will allow unopinionated users to not have to specify the attribute but leave
  the door open for opinionated ones that want to supply their preferred
  alternative binary.

  ```diff
  --- a/exercise-01/defs.bzl
  +++ b/exercise-01/defs.bzl

    json_to_yaml = rule(
        implementation = _json_to_yaml_impl,
        attrs = {
            "input": attr.label(
                mandatory = True,
                allow_single_file = [".json"],
                doc = "The JSON file to convert.",
            ),
              "converter": attr.label(
  -               mandatory = True,                      # The attribute is mandatory.
  +               default = "//exercise-01:converter",   # The default converter binary (if unspecified).
                  allow_single_file = True,              # Only accept a single file.
                  executable = True,                     # The file must be executable.
                  cfg = "exec",                          # Build the executable for the execution platform.
                  doc = "The JSON to YAML converter."
              ),
        },
    )
  ```

- Make the `converter` attribute not mandatory, private and with a default value.
  This will remove the attribute from the public API as well as prevent
  opinionated consumers from supplying their preferred alternative binary.

  ```diff
  --- a/exercise-01/defs.bzl
  +++ b/exercise-01/defs.bzl

    json_to_yaml = rule(
        implementation = _json_to_yaml_impl,
        attrs = {
            "input": attr.label(
                mandatory = True,
                allow_single_file = [".json"],
                doc = "The JSON file to convert.",
            ),
  -            "converter": attr.label(
  -               mandatory = True,                      # The attribute is mandatory.
  +            "_converter": attr.label(                 # Attribute names that start with '_' are private.
  +               default = "//exercise-01:converter",   # The default converter binary (if unspecified).
                  allow_single_file = True,              # Only accept a single file.
                  executable = True,                     # The file must be executable.
                  cfg = "exec",                          # Build the executable for the execution platform.
                  doc = "The JSON to YAML converter."
              ),
        },
    )
  ```

For the current `json_to_yaml` use case, it makes more sense to pick the second
option because the `converter` is a bespoke tool which makes it unlikely for
alternatives to share the same CLI interface, and hence, be potential drop-in
replacement candidates.

Let's verify that our modifications have not introduced any unexpected errors
by running the following command:

```shell
$ bazel build //...
INFO: Analyzed 2 targets (2 packages loaded, 1 target configured).
INFO: Found 2 targets...
INFO: Elapsed time: 0.114s, Critical Path: 0.00s
INFO: 1 process: 1 internal.
INFO: Build completed successfully, 1 total action
```

### Implement the `json_to_yaml` rule

Until now, we have been focused on defining the `json_to_yaml` public API and
have left the rule implementation body empty. Consequently, even when Bazel
reports a successful build, all `json_to_yaml` rule instantiations behave as
a *no-op*.

Let's change that by writting some Starlark code!

To perform the JSON to YAML conversion, we need a hold of the following files:

- `converter`: The actual CLI binary to execute to perform the conversion.
- `input`: The file to be converted to YAML. This is the file that the user
  specified in the `input` attribute in the `json_to_yaml` rule instantiation.
- `output`: The YAML file resulting from the JSON to YAML conversion operation.

Let's start by trying to get a hold of the first two (`converter` and `input`).
To do so, we will use the functions available the `ctx` object that gets passed
to the rule implementation function.

In particular, we will use the [`ctx.file`](https://bazel.build/rules/lib/builtins/ctx#file)
and [`ctx.executable`](https://bazel.build/rules/lib/builtins/ctx#executable)
structs to get access to the `input` and `converter` files respectively.

```diff
--- a/exercise-01/defs.bzl
+++ b/exercise-01/defs.bzl

    def _json_to_yaml_impl(ctx):
-        # TODO: Implement this!
-       pass
+       input_file = ctx.file.input
+       converter_binary = ctx.executable._converter
```

Now that we have a hold of the necessary input files, let's try to do the same
for the output file. Because output files are produced by build actions, Bazel
requires them to be declared ahead of time or, in other words, before they even
exist in the filesystem. This is necessary to help Bazel keep track of which
actions produce which outputs and, more importantly, determine the dependencies
between actions without having to run them to inspect their generated outputs.

To declare that our `json_to_yaml` rule will produce an output file, we need to
use the [`ctx.actions.declare_file`](https://bazel.build/rules/lib/builtins/actions.html#declare_file)
API. This function takes a mandatory `filename` attribute that specifies the
expected path for the output file.

```diff
--- a/exercise-01/defs.bzl
+++ b/exercise-01/defs.bzl

    def _json_to_yaml_impl(ctx):
        input_file = ctx.file.input
        converter_binary = ctx.executable._converter
+
+       # Use the target name as a prefix for the output filename to
+       # avoid output name conflicts between rules instantiated in
+       # the same BUILD.bazel file.
+       output_name = "%s.yaml" % ctx.label.name
+       output_file = ctx.actions.declare_file(output_name)
```

Now that we have a reference to all the files that we need to perform the
JSON to YAML conversion, let's register a build action to perform it!

To do so we will use the [`ctx.actions.run`](https://bazel.build/rules/lib/builtins/actions.html#run)
and [`ctx.actions.args`](https://bazel.build/rules/lib/builtins/Args.html) APIs to register the build
action and efficiently construct its CLI arguments:

```diff
--- a/exercise-01/defs.bzl
+++ b/exercise-01/defs.bzl

    def _json_to_yaml_impl(ctx):
        input_file = ctx.file.input
        converter_binary = ctx.executable._converter

       # Use the target name as a prefix for the output fi
       # avoid output name conflicts between rules instant
       # the same BUILD.bazel file.
       output_name = "%s.yaml" % ctx.label.name
       output_file = ctx.actions.declare_file(output_name)

+       ctx.actions.run(
+           inputs = [input_file],          # The action's input files.
+           outputs = [output_file],        # The action's output files.
+           executable = converter_binary,  # The executable file to run.
+           arguments = [                   # The CLI arguments for the executable.
+               ctx.actions.args()
+                   .add("-i", input_file)
+                   .add("-o", output_file),
+           ],
+           # Optional: The message to report while the action is executing.
+           progress_message = "Compiling %{input} to %{output}",
+           # Optional: A one-word description of the action.
+           mnemonic = "JsonToYaml",
+       )
```

Now that we have registered the build action, let's run a build to verify that
everything works as expected and, hence, the build produces the desired YAML
file.

```shell
$ bazel build //exercise-01:convert
INFO: Analyzed target //exercise-01:convert (0 packages loaded, 0 targets configured).
INFO: Found 1 target...
Target //exercise-01:convert up-to-date (nothing to build)
INFO: Elapsed time: 0.087s, Critical Path: 0.00s
INFO: 1 process: 1 internal.
INFO: Build completed successfully, 1 total action
```

Although Bazel reports a successful build, if you squint at the output, you may
be able to spot a this suspitious line:

```text
Target //exercise-01:convert up-to-date (nothing to build)
```

Why is Bazel reporting that there is nothing to build? Well, it turns out that
Bazel actions are lazily executed. What this means is that, it is **not** enough
for an action to be registered to guarantee its execution.

Instead, actions are only executed when any of their produced outputs are listed
as inputs of other actions. This behaviour is necessary to ensure that Bazel
never does more work than the minimal required but, at the same time, presents
a challenge for leaf/terminal actions because no other actions depend on their
outputs.

In the `json_to_yaml` rule implementation function, we only register a single
leaf/terminal action and, because there are no more actions to depend on its
outputs, Bazel decides to not execute it.

Let's fix that by returning the [`DefaultInfo`](https://bazel.build/rules/lib/providers/DefaultInfo#DefaultInfo)
provider. The `DefaultInfo` provider is a mechanism for rules to express the
list of outputs that have to always be requested by Bazel.

```diff
--- a/exercise-01/defs.bzl
+++ b/exercise-01/defs.bzl

    def _json_to_yaml_impl(ctx):
        input_file = ctx.file.input
        converter_binary = ctx.executable._converter

       # Use the target name as a prefix for the output fi
       # avoid output name conflicts between rules instant
       # the same BUILD.bazel file.
       output_name = "%s.yaml" % ctx.label.name
       output_file = ctx.actions.declare_file(output_name)

        ctx.actions.run(
            inputs = [input_file],          # The action's input files.
            outputs = [output_file],        # The action's output files.
            executable = converter_binary,  # The executable file to run.
            arguments = [                   # The CLI arguments for the executable.
                ctx.actions.args()
                    .add("-i", input_file)
                    .add("-o", output_file),
            ],
            # Optional: The message to report while the action is executing.
            progress_message = "Compiling %{input} to %{output}",
            # Optional: A one-word description of the action.
            mnemonic = "JsonToYaml",
        )

+       return [
+           DefaultInfo(files = depset([output_file]))
+       ]
```

With the addition of the `DefaultInfo` provider, Bazel knows that every
time it has to build a `json_to_yaml` rule instantiation, the `output_file` must
be requested and, as a consequence, the `JsonToYaml` action must be executed to
produce it.

Let's try again the previous build command and validate that it now runs our
`JsonToYaml` action and produces the desired output YAML file:

```shell
$ bazel build //exercise-01:convert
```

If everything went as expected, the command should produce an output similar to
the one below:

```shell
INFO: Analyzed target //exercise-01:convert (99 packages loaded, 12024 targets configured).
INFO: Found 1 target...
Target //exercise-01:convert up-to-date:
  bazel-out/darwin_arm64-fastbuild/bin/exercise-01/convert.yaml
INFO: Elapsed time: 23.916s, Critical Path: 21.79s
INFO: 12 processes: 6 internal, 6 darwin-sandbox.
INFO: Build completed successfully, 12 total actions
```

Bazel is now reporting that the target `//exercise-01:convert` is up to date
and produced the `convert.yaml` output file.

Success :tada: !

## Resources

- [Bazel Rule Attribute API Spec](https://bazel.build/rules/lib/toplevel/attr)
- [Bazel Rule Context API Spec](https://bazel.build/rules/lib/builtins/ctx)
