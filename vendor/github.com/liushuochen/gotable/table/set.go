package table

import (
	"fmt"
	"github.com/liushuochen/gotable/cell"
	"github.com/liushuochen/gotable/exception"
)

type Set struct {
	base []*cell.Column
}

func CreateSetFromString(columns ...string) (*Set, error) {
	if len(columns) <= 0 {
		return nil, exception.ColumnsLength()
	}

	set := &Set{base: make([]*cell.Column, 0)}
	for _, column := range columns {
		err := set.Add(column)
		if err != nil {
			return nil, err
		}
	}

	return set, nil
}

func (set *Set) Len() int {
	return len(set.base)
}

func (set *Set) Cap() int {
	return cap(set.base)
}

func (set *Set) Exist(element string) bool {
	return set.exist(element) != -1
}

func (set *Set) exist(element string) int {
	for index, data := range set.base {
		if data.Original() == element {
			return index
		}
	}

	return -1
}

func (set *Set) Clear() {
	set.base = make([]*cell.Column, 0)
}

func (set *Set) Add(element string) error {
	if set.Exist(element) {
		return fmt.Errorf("value %s has exit", element)
	}

	newHeader := cell.CreateColumn(element)
	set.base = append(set.base, newHeader)
	return nil
}

func (set *Set) Remove(element string) error {
	position := set.exist(element)
	if position == -1 {
		return fmt.Errorf("value %s has not exit", element)
	}

	set.base = append(set.base[:position], set.base[position+1:]...)
	return nil
}

func (set *Set) Get(name string) *cell.Column {
	for _, h := range set.base {
		if h.Original() == name {
			return h
		}
	}
	return nil
}

func (set *Set) Equal(other *Set) bool {
	if set.Len() != other.Len() {
		return false
	}

	c := make(chan bool)
	for index := range set.base {
		i := index
		go func(pos int) {
			if !set.base[pos].Equal(other.base[pos]) {
				c <- false
			} else {
				c <- true
			}
		}(i)
	}

	count := 0
	for {
		select {
		case equal := <-c:
			count += 1
			if !equal {
				return false
			}

			if count >= set.Len() {
				return true
			}
		}
	}
}
