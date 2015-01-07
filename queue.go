// a parallel operation queue.
package operationq

//go:generate counterfeiter -o fake_operationq/fake_operation.go . Operation

// The Operation interface is implemented externally, by the user of the queue.
type Operation interface {
	// Identifier for the operation's queue. Operations with the same key will be
	// executed in the order in which they were pushed. Operations with different
	// keys will be executed concurrently.
	Key() string

	// Work to execute when the operation is popped off of the queue.
	Execute()
}

//go:generate counterfeiter -o fake_operationq/fake_queue.go . Queue

// Queue executes operations, parallelized by operation key.
type Queue interface {
	// Enqueue an operation for execution.
	Push(Operation)

	// Prevent further operations from being added to the queue.
	// Close()
	//
	// // Wait for in-flight operations to complete. Call after Close.
	// Wait()
}

type multiQueue struct {
	queues       map[string][]Operation
	pushChan     chan Operation
	completeChan chan string
}

func NewQueue() Queue {
	q := &multiQueue{
		queues:       make(map[string][]Operation),
		pushChan:     make(chan Operation),
		completeChan: make(chan string),
	}
	go q.run()
	return q
}

func (q *multiQueue) run() {
	for {
		select {
		case queueKey := <-q.completeChan:
			queue := q.queues[queueKey]
			if len(queue) == 0 {
				delete(q.queues, queueKey)
			} else {
				o := queue[0]
				q.queues[queueKey] = queue[1:]
				go q.execute(o)
			}

		case o := <-q.pushChan:
			if queue, ok := q.queues[o.Key()]; ok {
				queue = append(queue, o)
				q.queues[o.Key()] = queue
			} else {
				q.queues[o.Key()] = []Operation{}
				go q.execute(o)
			}
		}
	}
}

func (q *multiQueue) Push(o Operation) {
	q.pushChan <- o
}

func (q *multiQueue) execute(o Operation) {
	o.Execute()
	q.completeChan <- o.Key()
}
