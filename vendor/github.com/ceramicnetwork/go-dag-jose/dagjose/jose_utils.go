package dagjose

import (
	"bytes"
	"errors"
	"fmt"
	"gopkg.in/square/go-jose.v2/json"
	"reflect"

	"github.com/ipfs/go-cid"
	"github.com/ipld/go-ipld-prime/codec/dagjson"
	"github.com/ipld/go-ipld-prime/datamodel"
	"github.com/ipld/go-ipld-prime/node/basicnode"
	"github.com/ipld/go-ipld-prime/schema"
	"github.com/multiformats/go-multibase"
)

func unflattenJWE(n datamodel.Node) (datamodel.Node, error) {
	// Check for the fastpath where the passed node is already of type `_EncodedJWE__Repr` or `_EncodedJWE`
	if _, castOk := n.(*_EncodedJWE__Repr); !castOk {
		// This could still be `_EncodedJWE`, so check for that.
		if _, castOk := n.(*_EncodedJWE); !castOk {
			if ciphertext, err := n.LookupByString("ciphertext"); err != nil {
				// `ciphertext` is mandatory so if any error occurs, return from here
				return nil, err
			} else if ciphertextString, err := ciphertext.AsString(); err != nil {
				return nil, err
			} else if recipients, err := lookupIgnoreAbsent("recipients", n); err != nil {
				return nil, err
			} else {
				jwe := map[string]interface{}{
					"ciphertext": ciphertextString,
				}
				// If `recipients` is absent, this must be a "flattened" JWE.
				if recipients == nil {
					recipientList := make([]map[string]interface{}, 1)
					recipientList[0] = make(map[string]interface{}, 0) // all recipient fields are optional
					if encryptedKey, err := lookupIgnoreNoSuchField("encrypted_key", n); err != nil {
						return nil, err
					} else if encryptedKey != nil {
						if encryptedKeyString, err := encryptedKey.AsString(); err != nil {
							return nil, err
						} else {
							recipientList[0]["encrypted_key"] = encryptedKeyString
						}
					}
					if header, err := lookupIgnoreNoSuchField("header", n); err != nil {
						return nil, err
					} else if header != nil {
						var headerMap map[string]interface{}
						if err := ipldNodeToGoPrimitive(header, &headerMap); err != nil {
							return nil, err
						} else {
							recipientList[0]["header"] = headerMap
						}
					}
					// Only add `recipients` to the JWE if one or more fields were present in the first list entry
					if len(recipientList[0]) > 0 {
						jwe["recipients"] = recipientList
					}
				} else {
					// If `recipients` is present, this must be a "general" JWE and no changes are needed but make sure
					// that `header` and/or `encrypted_key` are not present since that would be a violation of the spec.
					if encryptedKey, err := lookupIgnoreNoSuchField("encrypted_key", n); err != nil {
						return nil, err
					} else if encryptedKey != nil {
						return nil, errors.New("invalid JWE serialization")
					}
					if header, err := lookupIgnoreNoSuchField("header", n); err != nil {
						return nil, err
					} else if header != nil {
						return nil, errors.New("invalid JWE serialization")
					}
					var recipientList []map[string]interface{}
					if err = ipldNodeToGoPrimitive(recipients, &recipientList); err != nil {
						return nil, err
					}
					// Only add `recipients` to the JWE if one or more fields were present in the first list entry
					if len(recipientList[0]) > 0 {
						jwe["recipients"] = recipientList
					}
				}
				if aad, err := lookupIgnoreAbsent("aad", n); err != nil {
					return nil, err
				} else if aad != nil {
					if aadString, err := aad.AsString(); err != nil {
						return nil, err
					} else {
						jwe["aad"] = aadString
					}
				}
				if iv, err := lookupIgnoreAbsent("iv", n); err != nil {
					return nil, err
				} else if iv != nil {
					if ivString, err := iv.AsString(); err != nil {
						return nil, err
					} else {
						jwe["iv"] = ivString
					}
				}
				if protected, err := lookupIgnoreAbsent("protected", n); err != nil {
					return nil, err
				} else if protected != nil {
					if protectedString, err := protected.AsString(); err != nil {
						return nil, err
					} else {
						jwe["protected"] = protectedString
					}
				}
				if tag, err := lookupIgnoreAbsent("tag", n); err != nil {
					return nil, err
				} else if tag != nil {
					if tagString, err := tag.AsString(); err != nil {
						return nil, err
					} else {
						jwe["tag"] = tagString
					}
				}
				if unprotected, err := lookupIgnoreAbsent("unprotected", n); err != nil {
					return nil, err
				} else if unprotected != nil {
					var unprotectedMap map[string]interface{}
					if err := ipldNodeToGoPrimitive(unprotected, &unprotectedMap); err != nil {
						return nil, err
					} else {
						jwe["unprotected"] = unprotectedMap
					}
				}
				if n, err = goPrimitiveToIpldNode(jwe); err != nil {
					return nil, err
				}
			}
			if tn, castOk := n.(schema.TypedNode); castOk {
				// The "representation" node gives an accurate view of fields that are actually present
				n = tn.Representation()
			}
		}
	}
	return n, nil
}

// Always ignore `link`, if it was present. That is part of the "presentation" of the JWS but doesn't need to be part of
// the schema.
func unflattenJWS(n datamodel.Node) (datamodel.Node, error) {
	// Check for the fastpath where the passed node is already of type `_EncodedJWES__Repr` or `_EncodedJWS`
	if _, castOk := n.(*_EncodedJWS__Repr); !castOk {
		// This could still be `_EncodedJWS`, so check for that.
		if _, castOk := n.(*_EncodedJWS); !castOk {
			if payload, err := n.LookupByString("payload"); err != nil {
				// `payload` is mandatory so if any error occurs, return from here
				return nil, err
			} else if payloadString, err := payload.AsString(); err != nil {
				return nil, err
			} else if _, err := cid.Decode(string(multibase.Base64url) + payloadString); err != nil {
				return nil, errors.New(fmt.Sprintf("payload is not a valid CID: %v", err))
			} else if payloadString, err := payload.AsString(); err != nil {
				return nil, err
			} else if signatures, err := lookupIgnoreAbsent("signatures", n); err != nil {
				return nil, err
			} else {
				jws := map[string]interface{}{
					"payload": payloadString,
				}
				// If `signatures` is absent, this must be a "flattened" JWS.
				if signatures == nil {
					signaturesList := make([]map[string]interface{}, 1)
					signaturesList[0] = make(map[string]interface{}, 1) // at least one `signature` must be present
					if header, err := lookupIgnoreNoSuchField("header", n); err != nil {
						return nil, err
					} else if header != nil {
						var headerMap map[string]interface{}
						if err := ipldNodeToGoPrimitive(header, &headerMap); err != nil {
							return nil, err
						} else {
							signaturesList[0]["header"] = headerMap
						}
					}
					if protected, err := lookupIgnoreNoSuchField("protected", n); err != nil {
						return nil, err
					} else if protected != nil {
						if protectedString, err := protected.AsString(); err != nil {
							return nil, err
						} else {
							signaturesList[0]["protected"] = protectedString
						}
					}
					if signature, err := lookupIgnoreNoSuchField("signature", n); err != nil {
						return nil, err
					} else if signature != nil {
						if signatureString, err := signature.AsString(); err != nil {
							return nil, err
						} else {
							signaturesList[0]["signature"] = signatureString
						}
					}
					jws["signatures"] = signaturesList
				} else {
					// If `signatures` is present, this must be a "general" JWS and no changes are needed but make sure
					// that `header`, `protected`, and/or `signature` are not also present since that would be a
					// violation of the spec.
					if header, err := lookupIgnoreNoSuchField("header", n); err != nil {
						return nil, err
					} else if header != nil {
						return nil, errors.New("invalid JWS serialization")
					}
					if protected, err := lookupIgnoreNoSuchField("protected", n); err != nil {
						return nil, err
					} else if protected != nil {
						return nil, errors.New("invalid JWS serialization")
					}
					if signature, err := lookupIgnoreNoSuchField("signature", n); err != nil {
						return nil, err
					} else if signature != nil {
						return nil, errors.New("invalid JWS serialization")
					}
					var signatureList []map[string]interface{}
					if err = ipldNodeToGoPrimitive(signatures, &signatureList); err != nil {
						return nil, err
					}
					jws["signatures"] = signatureList
				}
				if n, err = goPrimitiveToIpldNode(jws); err != nil {
					return nil, err
				}
			}
			if tn, castOk := n.(schema.TypedNode); castOk {
				// The "representation" node gives an accurate view of fields that are actually present
				n = tn.Representation()
			}
		}
	}
	return n, nil
}

func isJWS(n datamodel.Node) (bool, error) {
	if payload, err := lookupIgnoreNoSuchField("payload", n); err != nil {
		return false, err
	} else {
		return payload != nil, nil
	}
}

func isJWE(n datamodel.Node) (bool, error) {
	if ciphertext, err := lookupIgnoreNoSuchField("ciphertext", n); err != nil {
		return false, err
	} else {
		return ciphertext != nil, nil
	}
}

func goPrimitiveToIpldNode(g interface{}) (datamodel.Node, error) {
	if jsonBytes, err := json.Marshal(g); err != nil {
		return nil, err
	} else {
		na := basicnode.Prototype.Any.NewBuilder()
		if err := dagjson.Decode(na, bytes.NewReader(jsonBytes)); err != nil {
			return nil, err
		} else {
			return na.Build(), nil
		}
	}
}

// TODO: Doesn't currently work for IPLD nodes with (nested) `bytes` type fields
func ipldNodeToGoPrimitive(n datamodel.Node, g interface{}) error {
	jsonBytes := bytes.NewBuffer([]byte{})
	if err := (dagjson.EncodeOptions{
		EncodeLinks: false,
		EncodeBytes: false,
	}.Encode(n, jsonBytes)); err != nil {
		return err
	} else {
		if err := json.Unmarshal(jsonBytes.Bytes(), &g); err != nil {
			return err
		} else {
			if reflect.TypeOf(g).Kind() == reflect.Map {
				g = sanitizeMap(g.(map[string]interface{}))
			}
			return nil
		}
	}
}

// Remove all `nil` values from the top-level structure or from within nested maps or slices
func sanitizeMap(m map[string]interface{}) map[string]interface{} {
	for key, value := range m {
		if value == nil {
			delete(m, key)
		} else if reflect.ValueOf(value).Kind() == reflect.Slice {
			for idx, entry := range value.([]interface{}) {
				if reflect.ValueOf(entry).Kind() == reflect.Map {
					m[key].([]interface{})[idx] = sanitizeMap(m[key].([]interface{})[idx].(map[string]interface{}))
				}
			}
		} else if reflect.ValueOf(value).Kind() == reflect.Map {
			m[key] = sanitizeMap(value.(map[string]interface{}))
		}
	}
	return m
}

func lookupIgnoreAbsent(key string, n datamodel.Node) (datamodel.Node, error) {
	value, err := n.LookupByString(key)
	if err != nil {
		if _, notFoundErr := err.(datamodel.ErrNotExists); !notFoundErr {
			return nil, err
		}
	}
	if value == datamodel.Absent {
		value = nil
	}
	return value, nil
}

func lookupIgnoreNoSuchField(key string, n datamodel.Node) (datamodel.Node, error) {
	value, err := lookupIgnoreAbsent(key, n)
	if err != nil {
		if _, noSuchFieldErr := err.(schema.ErrNoSuchField); !noSuchFieldErr {
			return nil, err
		}
	}
	return value, nil
}
