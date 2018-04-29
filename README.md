# k8s-queue-entry-operator

![Build Status](https://travis-ci.org/podnov/k8s-queue-entry-operator.svg)

Create an operator to watch for new queues. Define queues to be picked up by operator. The operator creates a queue worker per queue.
The queue worker responds to queue contents and creates kubernetes jobs to process queue contents.