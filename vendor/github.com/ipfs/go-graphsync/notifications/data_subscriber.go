package notifications

import (
	"sync"
)

type TopicDataSubscriber struct {
	idMapLk sync.RWMutex
	data    map[Topic][]TopicData
	Subscriber
}

// NewTopicDataSubscriber produces a subscriber that will transform
// events and topics before passing them on to the given subscriber
func NewTopicDataSubscriber(sub Subscriber) *TopicDataSubscriber {
	return &TopicDataSubscriber{
		Subscriber: sub,
		data:       make(map[Topic][]TopicData),
	}
}

func (m *TopicDataSubscriber) AddTopicData(id Topic, data TopicData) {
	m.idMapLk.Lock()
	m.data[id] = append(m.data[id], data)
	m.idMapLk.Unlock()
}

func (m *TopicDataSubscriber) getData(id Topic) []TopicData {
	m.idMapLk.RLock()
	defer m.idMapLk.RUnlock()

	data, ok := m.data[id]
	if !ok {
		return []TopicData{}
	}
	newData := make([]TopicData, len(data))
	copy(newData, data)
	return newData
}

func (m *TopicDataSubscriber) OnNext(topic Topic, ev Event) {
	for _, data := range m.getData(topic) {
		m.Subscriber.OnNext(data, ev)
	}
}

func (m *TopicDataSubscriber) OnClose(topic Topic) {
	for _, data := range m.getData(topic) {
		m.Subscriber.OnClose(data)
	}
	m.idMapLk.Lock()
	delete(m.data, topic)
	m.idMapLk.Unlock()
}
