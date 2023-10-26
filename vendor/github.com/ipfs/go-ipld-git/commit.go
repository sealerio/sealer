package ipldgit

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/ipld/go-ipld-prime"
	cidlink "github.com/ipld/go-ipld-prime/linking/cid"
	"github.com/ipld/go-ipld-prime/schema"
)

const prefixMergetag = 16 // the length of "mergetag object "

// DecodeCommit fills a NodeAssembler (from `Type.Commit__Repr.NewBuilder()`) from a stream of bytes
func DecodeCommit(na ipld.NodeAssembler, rd *bufio.Reader) error {
	if _, err := readNullTerminatedNumber(rd); err != nil {
		return err
	}

	c := _Commit{
		parents: _Commit_Link_List{[]_Commit_Link{}},
	}
	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		err = decodeCommitLine(&c, line, rd)
		if err != nil {
			return err
		}
	}

	return na.AssignNode(&c)
}

func decodeCommitLine(c Commit, line []byte, rd *bufio.Reader) error {
	switch {
	case bytes.HasPrefix(line, []byte("tree ")):
		sha, err := hex.DecodeString(string(line[5:]))
		if err != nil {
			return err
		}

		c.tree = _Tree_Link{cidlink.Link{Cid: shaToCid(sha)}}
	case bytes.HasPrefix(line, []byte("parent ")):
		psha, err := hex.DecodeString(string(line[7:]))
		if err != nil {
			return err
		}

		c.parents.x = append(c.parents.x, _Commit_Link{cidlink.Link{Cid: shaToCid(psha)}})
	case bytes.HasPrefix(line, []byte("author ")):
		a, err := parsePersonInfo(line)
		if err != nil {
			return err
		}

		c.author = _PersonInfo__Maybe{m: schema.Maybe_Value, v: a}
	case bytes.HasPrefix(line, []byte("committer ")):
		com, err := parsePersonInfo(line)
		if err != nil {
			return err
		}

		c.committer = _PersonInfo__Maybe{m: schema.Maybe_Value, v: com}
	case bytes.HasPrefix(line, []byte("encoding ")):
		c.encoding = _String__Maybe{m: schema.Maybe_Value, v: _String{string(line[9:])}}
	case bytes.HasPrefix(line, []byte("mergetag object ")):
		sha, err := hex.DecodeString(string(line)[prefixMergetag:])
		if err != nil {
			return err
		}

		mt, rest, err := readMergeTag(sha, rd)
		if err != nil {
			return err
		}

		c.mergetag.x = append(c.mergetag.x, *mt)

		if rest != nil {
			err = decodeCommitLine(c, rest, rd)
			if err != nil {
				return err
			}
		}
	case bytes.HasPrefix(line, []byte("gpgsig ")):
		sig, err := decodeGpgSig(rd)
		if err != nil {
			return err
		}
		c.signature = _GpgSig__Maybe{m: schema.Maybe_Value, v: sig}
	case len(line) == 0:
		rest, err := ioutil.ReadAll(rd)
		if err != nil {
			return err
		}

		c.message = _String{string(rest)}
	default:
		c.other.x = append(c.other.x, _String{string(line)})
	}
	return nil
}

func decodeGpgSig(rd *bufio.Reader) (_GpgSig, error) {
	out := _GpgSig{}

	line, _, err := rd.ReadLine()
	if err != nil {
		return out, err
	}

	if string(line) != " " {
		if strings.HasPrefix(string(line), " Version: ") || strings.HasPrefix(string(line), " Comment: ") {
			out.x += string(line) + "\n"
		} else {
			return out, fmt.Errorf("expected first line of sig to be a single space or version")
		}
	} else {
		out.x += " \n"
	}

	for {
		line, _, err := rd.ReadLine()
		if err != nil {
			return out, err
		}

		if bytes.Equal(line, []byte(" -----END PGP SIGNATURE-----")) {
			break
		}

		out.x += string(line) + "\n"
	}

	return out, nil
}

func encodeCommit(n ipld.Node, w io.Writer) error {
	ci := Type.Commit__Repr.NewBuilder()
	if err := ci.AssignNode(n); err != nil {
		return fmt.Errorf("not a Commit: %T %w", n, err)
	}
	c := ci.Build().(Commit)

	buf := new(bytes.Buffer)

	fmt.Fprintf(buf, "tree %s\n", hex.EncodeToString(c.tree.sha()))
	for _, p := range c.parents.x {
		fmt.Fprintf(buf, "parent %s\n", hex.EncodeToString(p.sha()))
	}
	fmt.Fprintf(buf, "author %s\n", c.author.v.GitString())
	fmt.Fprintf(buf, "committer %s\n", c.committer.v.GitString())
	if c.encoding.m == schema.Maybe_Value {
		fmt.Fprintf(buf, "encoding %s\n", c.encoding.v.x)
	}
	for _, mtag := range c.mergetag.x {
		fmt.Fprintf(buf, "mergetag object %s\n", hex.EncodeToString(mtag.object.sha()))
		fmt.Fprintf(buf, " type %s\n", mtag.typ.x)
		fmt.Fprintf(buf, " tag %s\n", mtag.tag.x)
		fmt.Fprintf(buf, " tagger %s\n \n", mtag.tagger.GitString())
		fmt.Fprintf(buf, "%s", mtag.message.x)
	}
	if c.signature.m == schema.Maybe_Value {
		fmt.Fprintln(buf, "gpgsig -----BEGIN PGP SIGNATURE-----")
		fmt.Fprint(buf, c.signature.v.x)
		fmt.Fprintln(buf, " -----END PGP SIGNATURE-----")
	}
	for _, line := range c.other.x {
		fmt.Fprintln(buf, line.x)
	}
	fmt.Fprintf(buf, "\n%s", c.message.x)

	// fmt.Printf("encode commit len: %d \n", buf.Len())
	//	fmt.Printf("out: %s\n", string(buf.Bytes()))
	_, err := fmt.Fprintf(w, "commit %d\x00", buf.Len())
	if err != nil {
		return err
	}

	_, err = buf.WriteTo(w)
	return err
}
