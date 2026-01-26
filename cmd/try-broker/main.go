/* https://stackoverflow.com/questions/36417199/how-to-broadcast-message-using-channel */

package main

import (
	"fmt"
	"time"
)

func main() {
	b := NewBroker[string]()
	go b.Start()

	go func() {
		msgCh := b.Subscribe()
		time.AfterFunc(time.Millisecond*30, func() {
			b.Unsubscribe(msgCh)
		})
		for msg := range msgCh {
			fmt.Println("Got: ", msg)
			time.Sleep(time.Millisecond * 100)
		}
		fmt.Println("Channel Done")
	}()

	go func() {
		for i := 0; ; i++ {
			b.Publish(fmt.Sprint(i))
		}
		/* When goroutine terminated, defer won't be executed */
	}()

	time.Sleep(time.Second)
}

type Broker[T any] struct {
	stopCh    chan struct{}
	publishCh chan T
	subCh     chan chan T
	unsubCh   chan chan T
}

func NewBroker[T any]() *Broker[T] {
	return &Broker[T]{
		stopCh:    make(chan struct{}),
		publishCh: make(chan T, 1),
		subCh:     make(chan chan T, 1),
		unsubCh:   make(chan chan T, 1),
	}
}

func (b *Broker[T]) Start() {
	subs := map[chan T]struct{}{}
	for {
		select {
		case <-b.stopCh:
			for msgCh := range subs {
				close(msgCh)
			}
			return
		case msgCh := <-b.subCh:
			subs[msgCh] = struct{}{}
		case msgCh := <-b.unsubCh:
			delete(subs, msgCh)
			close(msgCh)
		case msg := <-b.publishCh:
			for msgCh := range subs {
				select {
				case msgCh <- msg:
				default: /* drop message if buffer is full (the make() in Subscribe()) */
				}
			}
		}
	}
}

func (b *Broker[T]) Stop() {
	close(b.stopCh)
}

func (b *Broker[T]) Subscribe() chan T {
	msgCh := make(chan T, 5)
	b.subCh <- msgCh
	return msgCh
}

func (b *Broker[T]) Unsubscribe(msgCh chan T) {
	b.unsubCh <- msgCh
}

func (b *Broker[T]) Publish(msg T) {
	b.publishCh <- msg
}
