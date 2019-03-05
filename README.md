# Safedown

Safedown is for ensuring that an application shuts down gracefully even when termination or interruption signals are sent. 
The following snippet shows how to ensure text is printed when the application ends naturally or is terminated. 

```go
sa := NewShutdownActions(FirstInLastDone, syscall.SIGTERM, syscall.SIGINT)
defer sa.Shutdown()
sa.SetOnSignal(func(signal os.Signal) {
	fmt.Printf("A signal was received: %s\n", signal.String())
})

sa.AddActions(func() {
	fmt.Println("... and this will be done last.")
})

sa.AddActions(func() {
	fmt.Println("This will be done first ...")
})
```