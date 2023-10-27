# CHANGELOG

## v0.1.0

This release includes BREAKING CHANGES

* go-ipld-git is now a [go-ipld-prime](https://github.com/ipld/go-ipld-prime) IPLD codec. Use `Decode(na ipld.NodeAssembler, r io.Reader) error` and `Encode(n ipld.Node, w io.Writer) error` for direct use if required.
* There is now only one `Tag` type, `MergeTag` has been removed which had a `text` property. Use `Tag`'s `message` property instead to retrieve the tag message from a commit's `mergetag`. i.e. `<commit>/mergetag/message` instead of `<commit>/mergetag/text`.
* `PersonInfo` no longer exposes the human-readable RFC3339 format `date` field as a DAG node. The `date` and `timezone` fields are kept as their original string forms (to enable precise round-trips) as they exist in encoded Git data. e.g. `<commit>/author/date` now returns seconds in string form rather than an RFC3339 date string. Use this value and `<commit>/author/timezone` to reconstruct the original if needed.
