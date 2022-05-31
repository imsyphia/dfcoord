package channels

import "sync"

// Split spawns a goroutine copying data from the input channel to one or more
// output channels. The output channels are closed when the input is closed.
func Split[T any](in <-chan T, c ...chan<- T) {
	go func() {
		defer func() {
			for _, ch := range c {
				close(ch)
			}
		}()

		for {
			v, ok := <-in
			if !ok {
				return
			}
			for _, ch := range c {
				ch <- v
			}
		}
	}()
}

// SplitS is equivalent to Split except that it accepts a slice of
// bidirectional channels, avoiding typing issues that occur when
// attempting to pass a slice to Split, while giving up some type safety.
func SplitS[T any](in <-chan T, c []chan T) {
	go func() {
		defer func() {
			for _, ch := range c {
				close(ch)
			}
		}()

		for {
			v, ok := <-in
			if !ok {
				return
			}
			for _, ch := range c {
				ch <- v
			}
		}
	}()
}

// Merge spawns goroutines aggregating the read data of multiple channels into one.
// The output channel is closed when all input channels have been closed.
func Merge[T any](out chan<- T, c ...<-chan T) {
	merge := func(in <-chan T, out chan<- T, w *sync.WaitGroup) {
		defer w.Done()
		for {
			r, ok := <-in
			if !ok {
				return
			}
			out <- r
		}
	}

	go func() {
		var n sync.WaitGroup

		n.Add(len(c))

		for _, ch := range c {
			go merge(ch, out, &n)
		}
		n.Wait()
		close(out)
	}()
}

// MergeS is equivalent to Merge except that it accepts a slice of
// bidirectional channels, avoiding typing issues that occur when
// attempting to pass a slice to Merge, while giving up some type safety.
func MergeS[T any](out chan<- T, c []chan T) {
	merge := func(in <-chan T, out chan<- T, w *sync.WaitGroup) {
		defer w.Done()
		for {
			r, ok := <-in
			if !ok {
				return
			}
			out <- r
		}
	}

	go func() {
		var n sync.WaitGroup

		n.Add(len(c))

		for _, ch := range c {
			go merge(ch, out, &n)
		}
		n.Wait()
		close(out)
	}()
}
