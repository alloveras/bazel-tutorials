"""A Bazel rule to convert a JSON file to YAML."""

def _json_to_yaml_impl(ctx):
    # TODO: Implement this!
    pass

json_to_yaml = rule(
    implementation = _json_to_yaml_impl,
    attrs = {},
)
