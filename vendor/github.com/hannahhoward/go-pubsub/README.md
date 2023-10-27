# go-pubsub

A simple single topic thread safe event publish/subscribe interface for go

## Table of Contents

* [Description](./README.md#description)
* [Install](./README.md#install)
* [Usage](./README.md#usage)
* [License](./README.md#license)

## Description

This module provides a generic way to for consumers of a library to subscribe to events that library publishes. Consumers can also unsubscribe to stop receiving events.

Because go lacks generics, the library should provide a function to dispatch events to subscribers when constructing a pubsub instance. The dispatch function should convert generic events to specific events and generic subscriber functions to specific callbacks. You may want to keep the pubsub instance private to your library and expose your own subscribe function to provide type safety.

## Install

```
go get github.com/hannahhoward/go-pubsub
```

## Usage

```golang
package mylibrary

import(
  "errors"
  "github.com/hannahhoward/go-pubsub"
)

type MyLibrary struct {
  pubSub pubsub.PubSub
}

type MyLibarySubscriber func(event int)

type internalEvent int

func dispatcher(evt pubsub.Event, subscriber pubsub.SubscriberFn) error {
  ie, ok := evt.(internalEvent)
  if !ok {
    return errors.New("wrong type of event")
  }
  myLibarySubscriber, ok := subscriber.(MyLibrarySubscriber)
  if !ok {
    return errors.new("wrong type of subscriber")
  }
  myLibraySubscriber(ie)
  return nil
}

func New() * MyLibrary {
  return &MyLibrary{
    pubSub: pubsub.New(dispatcher)
  }
}

func (ml * MyLibrary) Subscribe(subscriber MyLibrarySubscriber) pubsub.Unsubscribe {
  return ml.pubSub.Subscribe(subscriber)
}

func (ml * MyLibrary) SomeFunc(value int) {
  ml.pubSub.Publish(internalEvent(value))
}
```


## License

Dual-licensed under [MIT](./LICENSE-MIT) + [Apache 2.0](./LICENSE-APACHE)
