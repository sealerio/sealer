/*
The dagcbor package provides a DAG-CBOR codec implementation.

The Encode and Decode functions match the codec.Encoder and codec.Decoder function interfaces,
and can be registered with the go-ipld-prime/multicodec package for easy usage with systems such as CIDs.

Importing this package will automatically have the side-effect of registering Encode and Decode
with the go-ipld-prime/multicodec registry, associating them with the standard multicodec indicator numbers for DAG-CBOR.

This implementation follows most of the rules of DAG-CBOR, namely:

- by and large, it does emit and parse CBOR!

- only explicit-length maps and lists will be emitted by Encode;

- only tag 42 is accepted, and it must parse as a CID;

- only 64 bit floats will be emitted by Encode.

This implementation is also not strict about certain rules:

- Encode is order-passthrough when emitting maps (it does not sort, nor abort in error if unsorted data is encountered).
To emit sorted data, the node should be sorted before applying the Encode function.

- Decode is order-passthrough when parsing maps (it does not sort, nor abort in error if unsorted data is encountered).
To be strict about the ordering of data, additional validation must be applied to the result of the Decode function.

- Decode will accept indeterminate length lists and maps without complaint.
(These should not be allowed according to the DAG-CBOR spec, nor will the Encode function re-emit such values,
so this behavior should almost certainly be seen as a bug.)

- Decode does not consistently verify that ints and floats use the smallest representation possible (or, the 64-bit version, in the float case).
(Only these numeric encodings should be allowed according to the DAG-CBOR spec, and the Encode function will not re-emit variations,
so this behavior should almost certainly be seen as a bug.)

A note for future contributors: some functions in this package expose references to packages from the refmt module, and/or use them internally.
Please avoid adding new code which expands the visibility of these references.
In future work, we'd like to reduce or break this relationship entirely
(in part, to reduce dependency sprawl, and in part because several of
the imprecisions noted above stem from that library's lack of strictness).
*/
package dagcbor
