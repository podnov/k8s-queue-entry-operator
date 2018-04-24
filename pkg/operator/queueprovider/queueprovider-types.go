package queueprovider

type QueueProvider interface {
	GetQueueEntryKeys() ([]string, error)
}
