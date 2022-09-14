/* Copyright (c) 2021 Bram Vandenbogaerde And Contributors
 * You may use, distribute or modify this code under the
 * terms of the Mozilla Public License 2.0, which is distributed
 * along with the source code.
 */

package scp

import "io"

// An adaptation of io.CopyN that keeps reading if it did not return
// a sufficient amount of bytes.
func CopyN(writer io.Writer, src io.Reader, size int64) (int64, error) {
	var total int64
	total = 0
	for total < size {
		n, err := io.CopyN(writer, src, size)
		if err != nil {
			return 0, err
		}
		total += n
	}

	return total, nil
}
