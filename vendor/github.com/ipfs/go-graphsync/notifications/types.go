package notifications

// Topic is a topic that events appear on
type Topic interface{}

// Event is a publishable event
type Event interface{}

// TopicData is data added to every message broadcast on a topic
type TopicData interface{}

// Subscriber is a subscriber that can receive events
type Subscriber interface {
	OnNext(Topic, Event)
	OnClose(Topic)
}

// Subscribable is a stream that can be subscribed to
type Subscribable interface {
	Subscribe(topic Topic, sub Subscriber) bool
	Unsubscribe(sub Subscriber) bool
}

// Publisher is an publisher of events that can be subscribed to
type Publisher interface {
	Close(Topic)
	Publish(Topic, Event)
	Shutdown()
	Startup()
	Subscribable
}

// EventTransform if a fucntion transforms one kind of event to another
type EventTransform func(Event) Event

// Notifee is a topic data subscriber plus a set of data you want to add to any topics subscribed to
// (used to call SubscribeWithData to inject data when events for a given topic emit)
type Notifee struct {
	Data       TopicData
	Subscriber *TopicDataSubscriber
}

// SubscribeWithData subscribes to the given subscriber on the given topic, and adds the notifies
// custom data into the list of data injected into callbacks when events occur on that topic
func SubscribeWithData(p Subscribable, topic Topic, notifee Notifee) {
	notifee.Subscriber.AddTopicData(topic, notifee.Data)
	p.Subscribe(topic, notifee.Subscriber)
}
