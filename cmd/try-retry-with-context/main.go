package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

/**
 * 2026.01.27 Conclusion:
 * 1. use `context.cancel` to cancel everything
 * 2. `Port.Read()` doesn't support cancel, so use this to close when context is done: 	`go func() { <-ctx.Done(); port.Close()}()`
 * 3. replace `time.Sleep()` with `select { case <-time.After(time.Second) }` to make it cancelable by ctx.Done()
 */

/**
 * port.Read() simulator
 *
 * 1. an action block for unlimited time
 * 2. use a way to stop it (itself doesn't support context)
 * 3. if it returned an error, retry after some time
 */
func main() {
	/* which can be called to simulate error occurs and manually close (like port.Close()) */
	errCh := make(chan struct{})
	stopCh := make(chan struct{})

	/**
	 * the main context, cancel() should stop everything
	 * so where do i cancel()? and where do I close(stopCh)?
	 * the loop is blocking,
	 * I should call cancel, it's offical way to cancel everything
	 * but the Read() is blocking...
	 * 	1. I must use another goroutine to handle ctx.Done can call close(stopCh)
	 * If I unblock it by close(stopCh), then ehh... I have no way to put cancel...?
	 * 	2. or I need to wrap close(stopCh) and cancel in a function...
	 *
	 * Once cancelled, there is no way to recover, everything should shutdown
	 * You should create a new instance from stratch
	 */
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	/**
	 * seems good, approved by ChatGPT,
	 * it is a translaion between context.cancel and traditional API which doesn't support context
	 *
	 * the ownership of closer should bound to exactly one person, it's here, no elsewhere
	 */
	go func() {
		<-ctx.Done()
		fmt.Println("context done, also close the blocking operation..")
		close(stopCh)
	}()

	go tryGetchar(errCh, stopCh, cancel)

retry_loop:
	for {
		/**
		 * before blocking, directly exit instead of run then error, is this approach good?
		 * i have to check ctx.Done() twice ...
		 *
		 * maybe it is safe to ignore this
		 * if i cancelled, the Read() should be closed, and it always returns with error immediately
		 */
		// select {
		// case <-ctx.Done():
		// 	break retry_loop
		// default:
		// }

		/**
		 * the block operation, it only return when error occurs
		 * while the error can be manually triggered, like port.Close()
		 * and if port.Close() called, further call will always return error immediately
		 */
		err := runSomeWorkThatBlock(errCh, stopCh)

		/**
		 * two possible situations:
		 * 1. error
		 * 2. cancelled
		 *
		 * maybe it is not easy to judge from the error whether it is stopped by error or cancellation
		 */
		if errors.Is(err, &PekoError{}) {
			/* error occurs, should retry */
			fmt.Println("Error, retrying...")
		} else if err == nil {
			/* manually cancel */
			/* in real environment, it may be same as error? */
			fmt.Println("Manually cancelled")
		}

		/**
		 * after blocking, check ctx.Done() again,
		 * this select make the Sleep can be cancelled
		 */
		select {
		case <-time.After(time.Second):
			/* continue to retry */
		case <-ctx.Done():
			/**
			 * should i return or break?
			 */
			fmt.Println("Context is Done, stop retry")
			/**
			 * break the loop ~= return
			 * because cancel the context means everything in this function is no more needed
			 * but with break, you can do some cleanup before function exits
			 */
			break retry_loop

		}
	}

}

type PekoError struct{}

func (e *PekoError) Error() string {
	return "PekoPeko"
}

/**
 * run forever
 * get errCh, stopCh
 * which can be called to simulate error occurs and manually close (like port.Close())
 */
func runSomeWorkThatBlock(errCh <-chan struct{}, stopCh <-chan struct{}) error {
	for {
		fmt.Println("working...")
		select {
		case <-time.After(time.Second):
		case <-errCh:
			return &PekoError{}
		case <-stopCh:
			return nil
		}
	}
}

func tryGetchar(errCh chan<- struct{}, stopCh chan<- struct{}, cancel func()) {
	// switch stdin into 'raw' mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	fmt.Println("Press `q` to Quit, `c` to cancel the context")
	b := make([]byte, 1)
	for {
		_, err = os.Stdin.Read(b)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("the char %q was hit\n", string(b[0]))
		switch string(b[0]) {
		case "e":
			errCh <- struct{}{}
		case "s":
			close(stopCh)
		case "c":
			cancel()
		case "q", "\x03": /* handle ctrl-c */
			os.Exit(0)
		}
	}
}
